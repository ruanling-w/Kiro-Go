// Package main provides the entry point for Kiro API Proxy.
//
// Kiro API Proxy is a reverse proxy service that translates Kiro API requests
// into OpenAI and Anthropic (Claude) compatible formats. Key features include:
//   - Multi-account pool with round-robin load balancing
//   - Automatic OAuth token refresh
//   - Streaming response support for real-time AI interactions
//   - Admin panel for account and configuration management
//
// The service exposes the following endpoints:
//   - /v1/messages - Claude API compatible endpoint
//   - /v1/chat/completions - OpenAI API compatible endpoint
//   - /admin - Web-based administration panel
package main

import (
	"flag"
	"fmt"
	"kiro-go/config"
	"kiro-go/logger"
	"kiro-go/pool"
	"kiro-go/proxy"
	"kiro-go/web"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func main() {
	// Flags exist mainly for the `kiroproxy` npm launcher, which needs to point
	// the server at a per-user config and to move it off a busy port without
	// editing config.json. Env vars keep working for Docker.
	var (
		flagConfig  = flag.String("config", "", "path to config.json (default: $CONFIG_PATH, ./data/config.json, or ~/.kiroproxy/config.json)")
		flagHost    = flag.String("host", "", "bind host, overrides config for this run")
		flagPort    = flag.Int("port", 0, "bind port, overrides config for this run")
		flagVersion = flag.Bool("version", false, "print version and exit")
	)
	flag.Parse()

	if *flagVersion {
		fmt.Println(config.Version)
		return
	}

	// Serve the admin SPA from the binary when it was built with the frontend
	// bundled, so /admin works from any working directory. Falls through to
	// reading web/dist off disk in a repo checkout.
	if dist, ok := web.DistFS(); ok {
		proxy.SetAssetFS(dist)
	}

	configPath := config.ResolveConfigPath(*flagConfig)

	// 确保数据目录存在
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// 加载配置
	if err := config.Init(configPath); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	config.SetRuntimeBind(*flagHost, *flagPort)

	// Initialize log level: LOG_LEVEL env var takes priority over config, defaulting to "info".
	logger.Init(config.GetLogLevel())

	// 环境变量覆盖密码
	if envPassword := os.Getenv("ADMIN_PASSWORD"); envPassword != "" {
		config.SetPassword(envPassword)
	}

	// 初始化账号池
	pool.GetPool()

	// 创建 HTTP 处理器（包含后台刷新任务）
	handler := proxy.NewHandler()
	defer func() {
		if err := handler.Close(); err != nil {
			logger.Errorf("close runtime store: %v", err)
		}
	}()

	// 启动服务器
	addr := fmt.Sprintf("%s:%d", config.GetBindHost(), config.GetBindPort())
	logger.Infof("Kiro-Go v%s starting on http://%s (log level: %s)", config.Version, addr, logger.LevelName(logger.GetLevel()))
	logger.Infof("Config: %s", configPath)
	logger.Infof("Admin panel: http://%s/admin", addr)
	logger.Infof("Claude API: http://%s/v1/messages", addr)
	logger.Infof("OpenAI API: http://%s/v1/chat/completions", addr)

	// WriteTimeout intentionally 0: SSE streams can run for minutes while the
	// upstream model produces tokens. ReadHeaderTimeout + ReadTimeout still
	// guard against slowloris-style header/body stalls.
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Errorf("Server failed: %v", err)
		os.Exit(1)
	}
}
