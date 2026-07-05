// Package store 封装数据持久化。P0 使用 SQLite（纯 Go 驱动，无需 CGO）。
package store

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// Store 持有数据库连接。P0 先用一个聚合的 Store，
// 待模型稳定、迁移 MySQL/多用户时再抽细 repository 接口。
type Store struct {
	db *sql.DB
}

// Open 打开 SQLite 数据库并执行幂等建表迁移。
// dsn 为文件路径，会自动创建所在目录。
func Open(dsn string) (*Store, error) {
	if dir := filepath.Dir(dsn); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("创建数据库目录失败: %w", err)
		}
	}

	// 通过连接串开启 PRAGMA，保证连接池里每条连接都生效：
	//   foreign_keys(1)   —— 外键约束（SQLite 默认关闭）
	//   busy_timeout(5000)—— 写锁等待，避免立即报 database is locked
	//   journal_mode(WAL) —— 读写并发更好
	connDSN := dsn + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"

	db, err := sql.Open("sqlite", connDSN)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// SQLite 单写入者，连接数限制为 1 可避免写锁竞争。
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// migrate 执行内嵌的建表脚本，脚本使用 IF NOT EXISTS，可重复执行。
func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, schemaSQL); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}
	return nil
}

// DB 返回底层 *sql.DB，供上层 service 使用。
func (s *Store) DB() *sql.DB {
	return s.db
}

// Ping 检查数据库连通性，供健康检查使用。
func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Close 关闭数据库连接。
func (s *Store) Close() error {
	return s.db.Close()
}
