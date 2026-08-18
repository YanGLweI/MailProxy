// Command sendmail 通过 MailProxy 代理发送一封测试邮件（SMTP over SSL）。
//
// 用法示例：
//
//	go run ./testtools/sendmail -addr 127.0.0.1:10465 \
//	  -from lw.yang@ho-brostech.com -to someone@example.com
package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"os"
	"time"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:10465", "代理地址 host:port")
	host := flag.String("host", "", "TLS ServerName，默认取 -addr 的主机部分")
	from := flag.String("from", "", "MAIL FROM 发件人")
	to := flag.String("to", "", "收件人，逗号分隔多个")
	user := flag.String("user", "", "代理登录账号（mode=auth 时需要）")
	pass := flag.String("pass", "", "代理登录密码（mode=auth 时需要）")
	subject := flag.String("subject", "MailProxy test mail", "邮件主题")
	insecure := flag.Bool("insecure", true, "跳过自签名证书校验（测试用）")
	flag.Parse()

	if *from == "" || *to == "" {
		fmt.Fprintln(os.Stderr, "必须指定 -from 与 -to")
		os.Exit(2)
	}

	if *host == "" {
		h, _, err := net.SplitHostPort(*addr)
		if err != nil {
			log.Fatal("非法地址:", *addr)
		}
		*host = h
	}

	recipients := splitRecipients(*to)

	conn, err := tls.Dial("tcp", *addr, &tls.Config{
		ServerName:         *host,
		InsecureSkipVerify: *insecure,
	})
	if err != nil {
		log.Fatal("TLS 连接失败:", err)
	}
	c, err := smtp.NewClient(conn, *host)
	if err != nil {
		log.Fatal("SMTP 握手失败:", err)
	}
	defer c.Close()

	if err := c.Hello("mailproxy-test"); err != nil {
		log.Fatal("EHLO 失败:", err)
	}
	if *user != "" {
		if err := c.Auth(smtp.PlainAuth("", *user, *pass, *host)); err != nil {
			log.Fatal("AUTH 失败:", err)
		}
	}
	if err := c.Mail(*from); err != nil {
		log.Fatal("MAIL FROM 失败:", err)
	}
	for _, rcpt := range recipients {
		if err := c.Rcpt(rcpt); err != nil {
			log.Fatal("RCPT TO 失败:", err)
		}
	}
	w, err := c.Data()
	if err != nil {
		log.Fatal("DATA 失败:", err)
	}
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMessage-ID: <mailproxy-test-%d@localhost>\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n"+
		"这是一封经由 MailProxy 代理发送的测试邮件。\r\n发送时间: %s\r\n",
		*from, *to, *subject, time.Now().Format(time.RFC1123Z),
		time.Now().UnixNano(), time.Now().Format(time.RFC3339))
	if _, err := w.Write([]byte(msg)); err != nil {
		log.Fatal("写入邮件内容失败:", err)
	}
	if err := w.Close(); err != nil {
		log.Fatal("邮件投递失败:", err)
	}
	if err := c.Quit(); err != nil {
		log.Println("QUIT 异常:", err)
	}
	log.Println("测试邮件发送成功 ->", recipients)
}

func splitRecipients(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ',' {
			if t := trim(cur); t != "" {
				out = append(out, t)
			}
			cur = ""
			continue
		}
		cur += string(r)
	}
	if t := trim(cur); t != "" {
		out = append(out, t)
	}
	return out
}

func trim(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
