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

	// 3. 打开数据库并执行迁移
	st, err := store.Open(cfg.DB.DSN)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := st.Close(); cerr != nil {
			log.Error("关闭数据库失败", "error", cerr)
		}
	}()

	// 4. 构建 HTTP Server
	handler := apphttp.NewServer(st, log).Handler()
	srv := &http.Server{
		Addr:              ":" + cfg.HTTP.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      60 * time.Second, // LLM 调用可能较慢，给足余量
		IdleTimeout:       60 * time.Second,
	}

	// 5. 启动监听（独立 goroutine），错误回传主协程
	serverErr := make(chan error, 1)
	go func() {
		log.Info("HTTP 服务启动", "port", cfg.HTTP.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// 6. 等待退出信号或启动错误
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return err
	case sig := <-sigCh:
		log.Info("收到退出信号，开始优雅关闭", "signal", sig.String())
	}

	// 7. 真优雅关闭：停止接收新请求，等待在途请求处理完毕，最多等 10s
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		return err
	}

	log.Info("服务已优雅关闭")
	return nil
}
