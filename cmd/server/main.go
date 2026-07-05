// Command server 是健康管理 Agent 的 HTTP 服务入口。
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	apphttp "healthAgent/internal/http"

	"healthAgent/internal/config"
	"healthAgent/internal/llm"
	"healthAgent/internal/logger"
	"healthAgent/internal/store"
)

func main() {
	if err := run(); err != nil {
		slog.Error("服务启动失败", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// 1. 加载配置（yaml + env）。配置路径可用 CONFIG_PATH 覆盖，便于容器/多环境部署。
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	// 2. 初始化日志
	log := logger.New(cfg.Log.Level, cfg.Log.Debug)
	slog.SetDefault(log)

	// 3. 打开数据库并执行迁移。
	// S1 阶段对话不依赖 DB：连不上只告警、降级启动；等 S2 接入 MySQL 后再改回强依赖（连不上直接 return err）。
	st, err := store.Open(cfg.DB.DSN)
	if err != nil {
		log.Warn("数据库未就绪，降级启动（S1 对话不依赖 DB）", "error", err)
		st = nil
	}
	defer func() {
		if st == nil {
			return
		}
		if cerr := st.Close(); cerr != nil {
			log.Error("关闭数据库失败", "error", cerr)
		}
	}()

	// 4. 构建 LLM 客户端（DeepSeek）。API Key 缺省时不阻断启动，调用时走降级兜底。
	if cfg.LLM.APIKey == "" {
		log.Warn("未配置 DEEPSEEK_API_KEY，对话将返回降级兜底回复（请在 .env 中填入）")
	}
	llmClient := llm.New(cfg.LLM.APIKey, cfg.LLM.BaseURL, cfg.LLM.Model, time.Duration(cfg.LLM.TimeoutSeconds)*time.Second)

	// 5. 构建 HTTP Server
	handler := apphttp.NewServer(st, llmClient, log).Handler()
	srv := &http.Server{
		Addr:              ":" + cfg.HTTP.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      60 * time.Second, // LLM 调用可能较慢，给足余量
		IdleTimeout:       60 * time.Second,
	}

	// 6. 启动监听（独立 goroutine），错误回传主协程
	serverErr := make(chan error, 1)
	go func() {
		log.Info("HTTP 服务启动", "port", cfg.HTTP.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// 7. 等待退出信号或启动错误
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return err
	case sig := <-sigCh:
		log.Info("收到退出信号，开始优雅关闭", "signal", sig.String())
	}

	// 8. 真优雅关闭：停止接收新请求，等待在途请求处理完毕，最多等 10s
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		return err
	}

	log.Info("服务已优雅关闭")
	return nil
}
