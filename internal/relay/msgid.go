package relay

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"strings"
)

// MessageID 从邮件报文头解析 Message-ID；缺失时生成一个内部 ID（仅用于日志追踪，不改写报文）。
func MessageID(data []byte) string {
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 4096), 64*1024)
	var folded strings.Builder
	current := ""
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			break // 头部结束
		}
		if line[0] == ' ' || line[0] == '\t' {
			// 折叠续行
			if current == "message-id" {
				folded.WriteString(strings.TrimSpace(line))
			}
			continue
		}
		// 提交上一个头部
		if current == "message-id" && folded.Len() > 0 {
			return cleanID(folded.String())
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		current = strings.ToLower(strings.TrimSpace(key))
		folded.Reset()
		if current == "message-id" {
			folded.WriteString(strings.TrimSpace(val))
		}
	}
	if current == "message-id" && folded.Len() > 0 {
		return cleanID(folded.String())
	}
	return GenerateMessageID()
}

func cleanID(s string) string {
	s = strings.TrimSpace(s)
	return strings.Trim(s, "<>")
}

// GenerateMessageID 生成一个仅用于日志的内部追踪 ID。
func GenerateMessageID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return "mailproxy-" + hex.EncodeToString(b[:])
}
