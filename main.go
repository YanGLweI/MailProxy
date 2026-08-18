// MailProxy 是一个 SMTP 邮件代理网关：对外提供 SMTP over SSL 服务，
// 接收业务程序的邮件后按路由规则转发到后端真实 SMTP 服务器。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mailproxy/internal/config"
	"mailproxy/internal/metrics"
	"mailproxy/internal/relay"
	"mailproxy/internal/server"
)

func main() {
	configPath := flag.String("config", "config.yaml", "配置文件路径")
	logPath := flag.String("log", "", "日志文件路径（覆盖配置文件中的 log.file；留空则按配置）")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "加载配置失败:", err)
		os.Exit(1)
	}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, "配置校验失败:", err)
		os.Exit(1)
	}
	if *logPath != "" {
		cfg.Log.File = *logPath
	}

	logger, err := setupLogger(cfg.Log)
	if err != nil {
		fmt.Fprintln(os.Stderr, "初始化日志失败:", err)
		os.Exit(1)
	}
	cfg.WarnIfInsecureFilePerms(*configPath, func(format string, args ...any) {
		logger.Warn(fmt.Sprintf(format, args...))
	})

	provider := config.NewProvider(*configPath, cfg)
	logger.Info("MailProxy 启动中", "config", *configPath, "listen", cfg.Server.Listen)

	// 启动前对每组后端做连通性检测
	if cfg.ValidateOnStart {
		ok := true
		for _, b := range cfg.Backends {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			err := relay.CheckBackend(ctx, relay.RefFromBackend(b))
			cancel()
			if err != nil {
				ok = false
				logger.Error("后端 SMTP 连通性检测失败", "backend", b.ID, "host", b.Host, "error", err)
			} else {
				logger.Info("后端 SMTP 连通性检测通过", "backend", b.ID, "host", b.Host, "username", b.Username)
			}
		}
		if !ok {
			logger.Error("存在不可用的后端配置，启动中止；如希望跳过检测可设置 validate_on_start: false")
			os.Exit(1)
		}
	}

	// 可选 Prometheus 指标端点
	var m *metrics.Metrics
	if cfg.Metrics.Enabled {
		m = metrics.New()
		mux := http.NewServeMux()
		mux.Handle("/metrics", metrics.Handler())
		hs := &http.Server{Addr: cfg.Metrics.Listen, Handler: mux}
		go func() {
			if err := hs.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Error("metrics 端点异常退出", "error", err)
			}
		}()
		logger.Info("Prometheus 指标端点已启动", "listen", cfg.Metrics.Listen)
		defer hs.Close()
	}

	srv := server.New(provider, logger, m)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Run(ctx) }()

	// SIGHUP 触发热加载
	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)

loop:
	for {
		select {
		case err := <-errCh:
			if err != nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, http.ErrServerClosed) {
				logger.Error("SMTP 服务异常退出", "error", err)
				os.Exit(1)
			}
			break loop
		case <-hup:
			if _, err := provider.Reload(); err != nil {
				logger.Error("配置热加载失败，沿用旧配置", "error", err)
			} else {
				logger.Info("配置热加载成功（监听地址/证书变更需重启生效）")
			}
		case <-ctx.Done():
			logger.Info("收到退出信号，等待现有连接处理完成...")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			done := make(chan error, 1)
			go func() { done <- srv.Stop() }()
			select {
			case err := <-done:
				if err != nil {
					logger.Error("优雅关闭异常", "error", err)
				}
			case <-shutdownCtx.Done():
				logger.Warn("优雅关闭超时，强制退出")
			}
			cancel()
			break loop
		}
	}
	logger.Info("MailProxy 已退出")
}

// setupLogger 初始化 slog：控制台始终输出，可选追加文件输出。
func setupLogger(lc config.LogConfig) (*slog.Logger, error) {
	var level slog.Level
	switch lc.Level {
	case "debug":
		level = slog.LevelDebug
	case "", "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		return nil, fmt.Errorf("非法日志级别 %q（支持 debug/info/warn/error）", lc.Level)
	}

	var w io.Writer = os.Stdout
	if lc.File != "" {
		f, err := os.OpenFile(lc.File, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
		if err != nil {
			return nil, err
		}
		w = io.MultiWriter(os.Stdout, f)
	}
	handler := slog.NewTextHandler(w, &slog.HandlerOptions{Level: level})
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger, nil
}
