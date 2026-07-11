// Command server 是健康管理 Agent 的 HTTP 服务入口。
package main

import (
	"context"
	"embed"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"healthAgent/internal/config"
	"healthAgent/internal/handler"
	"healthAgent/internal/llm"
	"healthAgent/internal/logger"
	"healthAgent/internal/service"
	"healthAgent/internal/store"
)

// migrationsFS 把 migrations/ 下的 SQL 打进二进制，部署时无需额外携带脚本。
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

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

	// 3. 构建 LLM 客户端（DeepSeek）。API Key 缺省时不阻断启动，调用时走降级兜底。
	if cfg.DeepSeek.APIKey == "" {
		log.Warn("未配置 DEEPSEEK_API_KEY，对话将返回降级兜底回复（请在 .env 中填入）")
	}
	client := llm.NewDeepSeekClient(cfg.DeepSeek.APIKey, cfg.DeepSeek.BaseURL, cfg.DeepSeek.Model, time.Duration(cfg.DeepSeek.TimeoutSeconds)*time.Second)

	// 3.1 初始化 PostgreSQL（对话历史 source of truth）。连不上直接失败——不允许无存储启动。
	db, err := store.NewPostgres(cfg.Postgres)
	if err != nil {
		return err
	}
	defer db.Close()
	log.Info("PostgreSQL 连接就绪", "addr", cfg.Postgres.Host+":"+strconv.Itoa(cfg.Postgres.Port), "db", cfg.Postgres.DBName)

	// 3.1b 执行数据库迁移（golang-migrate）。结构不到位不允许启动——与 source of truth fail-fast 一致。
	if err := store.RunMigrations(cfg.Postgres, migrationsFS, "migrations"); err != nil {
		return err
	}
	log.Info("数据库迁移已应用至最新版本")

	// 3.2 初始化 Redis（缓热上下文）。连不上不阻断启动，运行时降级为直读 PostgreSQL。
	rdb, err := store.NewRedis(cfg.Redis)
	if err != nil {
		log.Warn("Redis 连接失败，将降级为直读 PostgreSQL", "error", err)
		rdb = nil
	} else {
		defer rdb.Close()
		log.Info("Redis 连接就绪", "addr", cfg.Redis.Addr)
	}
	_ = rdb // TODO(P2): 注入 repository / server，当前仅完成建连与优雅关闭

	// writeTimeout 是单个请求最长处理时间（LLM 调用可能较慢，给足余量）。
	// 优雅关闭的等待时间必须 >= 它，否则在途慢请求会被提前掐断，见下方 shutdownGrace。
	const writeTimeout = 60 * time.Second

	// 4. 在 composition root 组装业务服务；HTTP handler 只依赖 service。
	chatService := service.NewChatService(client, cfg.Chat.MaxReplyChars)
	identityRepository := store.NewPostgresIdentityRepository(db)
	identityService := service.NewIdentityService(identityRepository, time.Duration(cfg.Identity.GuestTokenTTLHours)*time.Hour)
	sessionRepository := store.NewPostgresSessionRepository(db)
	sessionService := service.NewSessionService(sessionRepository)
	messageRepository := store.NewPostgresMessageRepository(db)
	messageService := service.NewMessageService(messageRepository)
	turnLeaseRepository := store.NewPostgresTurnLeaseRepository(db)
	turnLeaseService := service.NewTurnLeaseService(turnLeaseRepository)
	srvHandler := handler.NewServer(chatService, identityService, sessionService, messageService, turnLeaseService, cfg.Identity, log).Handler()
	srv := &http.Server{
		Addr:              ":" + cfg.HTTP.Port,
		Handler:           srvHandler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      writeTimeout,
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

	// 7. 真优雅关闭：停止接收新请求，等待在途请求处理完毕。
	// 等待上限必须 >= writeTimeout，再加几秒缓冲，确保最慢的在途 LLM 请求也能跑完，
	// 而不是刚等到一半就被 cancel 掐断连接（那样对慢请求来说优雅关闭形同虚设）。
	shutdownGrace := writeTimeout + 5*time.Second
	ctx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		return err
	}

	log.Info("服务已优雅关闭")
	return nil
}
