package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

// Config 是应用的全部配置
type Config struct {
	HTTP     HTTPConfig     `yaml:"http"     env-prefix:"HTTP_"`
	Identity IdentityConfig `yaml:"identity" env-prefix:"IDENTITY_"`
	Chat     ChatConfig     `yaml:"chat"     env-prefix:"CHAT_"`
	DeepSeek LLMConfig      `yaml:"deepseek" env-prefix:"DEEPSEEK_"`
	OpenAI   OpenAIConfig   `yaml:"openai"   env-prefix:"OPENAI_"`
	Postgres PostgresConfig `yaml:"postgres" env-prefix:"POSTGRES_"`
	Redis    RedisConfig    `yaml:"redis"    env-prefix:"REDIS_"`
	Log      LogConfig      `yaml:"log"      env-prefix:"LOG_"`
}

// HTTPConfig 保持不变
type HTTPConfig struct {
	Port string `yaml:"port" env:"PORT" env-default:"8091"`
}

// IdentityConfig 控制 Guest 设备凭证和 Cookie 的生命周期。
type IdentityConfig struct {
	GuestCookieName    string `yaml:"guest_cookie_name"     env:"GUEST_COOKIE_NAME"      env-default:"health_guest"`
	GuestTokenTTLHours int    `yaml:"guest_token_ttl_hours" env:"GUEST_TOKEN_TTL_HOURS" env-default:"8760"`
	CookieSecure       bool   `yaml:"cookie_secure"         env:"COOKIE_SECURE"         env-default:"false"`
}

// LLMConfig 目前专门给 DeepSeek 用
type LLMConfig struct {
	APIKey         string  `yaml:"-"        env:"API_KEY"`
	BaseURL        string  `yaml:"base_url" env:"BASE_URL" env-default:"https://api.deepseek.com"`
	Model          string  `yaml:"model"    env:"MODEL"    env-default:"deepseek-chat"`
	Temperature    float64 `yaml:"temperature" env:"TEMPERATURE" env-default:"0"`
	TimeoutSeconds int     `yaml:"timeout"  env:"TIMEOUT"  env-default:"30"`
}

// ChatConfig 控制聊天业务层（不是 LLM 传输层）的行为。
type ChatConfig struct {
	// MaxReplyChars 是单条 assistant 回复累积的最大字符数上限，防止模型异常时（例如陷入重复输出）
	// 无限占用内存并写入一条超大的数据库行。<=0 时回退为 service.DefaultMaxReplyChars。
	MaxReplyChars  int    `yaml:"max_reply_chars" env:"MAX_REPLY_CHARS" env-default:"4000"`
	PromptPath     string `yaml:"prompt_path" env:"PROMPT_PATH" env-default:"prompts/health_chat_v2.tmpl"`
	PromptVersion  string `yaml:"prompt_version" env:"PROMPT_VERSION" env-default:"health-chat-v2"`
	SafetyBoundary string `yaml:"safety_boundary" env:"SAFETY_BOUNDARY" env-default:"健康建议仅供一般信息参考，不能替代医生面诊、诊断或治疗。"`
}

// OpenAIConfig 是你后续新增的 OpenAI 配置
type OpenAIConfig struct {
	APIKey         string `yaml:"-"        env:"API_KEY"` // 实际读取环境变量 OPENAI_API_KEY
	BaseURL        string `yaml:"base_url" env:"BASE_URL" env-default:"https://api.openai.com/v1"`
	Model          string `yaml:"model"    env:"MODEL"    env-default:"gpt-4-turbo"`
	TimeoutSeconds int    `yaml:"timeout"  env:"TIMEOUT"  env-default:"60"` // OpenAI 可能更慢，单独设超时
}

type LogConfig struct {
	Level string `yaml:"level" env:"LEVEL" env-default:"info"`
	Debug bool   `yaml:"debug" env:"DEBUG" env-default:"false"`
}

// PostgresConfig 是 PostgreSQL 连接配置。密码走环境变量（POSTGRES_PASSWORD），不写进 yaml。
type PostgresConfig struct {
	Host            string `yaml:"host"              env:"HOST"              env-default:"127.0.0.1"`
	Port            int    `yaml:"port"              env:"PORT"              env-default:"5433"`
	User            string `yaml:"user"              env:"USER"              env-default:"postgres"`
	Password        string `yaml:"-"                 env:"PASSWORD"          env-default:"root"`
	DBName          string `yaml:"dbname"            env:"DBNAME"            env-default:"health_db"`
	MaxOpenConns    int    `yaml:"max_open_conns"    env:"MAX_OPEN_CONNS"    env-default:"50"`
	MaxIdleConns    int    `yaml:"max_idle_conns"    env:"MAX_IDLE_CONNS"    env-default:"10"`
	ConnMaxLifetime int    `yaml:"conn_max_lifetime" env:"CONN_MAX_LIFETIME" env-default:"3600"` // 单位：秒
}

// DSN 组装 PostgreSQL 连接串（URL 形式，pgx 可直接解析）。
// sslmode=disable 用于本地/内网；生产跨网络应设 require / verify-full。
func (c PostgresConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		c.User, c.Password, c.Host, c.Port, c.DBName)
}

// RedisConfig 是 Redis 连接配置。密码走环境变量（REDIS_PASSWORD），本地无密码时留空。
type RedisConfig struct {
	Addr         string `yaml:"addr"           env:"ADDR"           env-default:"127.0.0.1:6379"`
	Password     string `yaml:"-"              env:"PASSWORD"`
	DB           int    `yaml:"db"             env:"DB"             env-default:"0"`
	PoolSize     int    `yaml:"pool_size"      env:"POOL_SIZE"      env-default:"50"`
	MinIdleConns int    `yaml:"min_idle_conns" env:"MIN_IDLE_CONNS" env-default:"5"`
}

// Load 加载配置：cleanenv 按扩展名解析 yaml 文件，并用环境变量覆盖。
// 注意：cleanenv 只读“进程环境变量”，不会解析 .env 文件；
// 所以必须先用 godotenv 把 .env 灌进环境，API Key 等敏感项才读得到。
func Load(path string) (*Config, error) {
	// 尽力加载 .env，不存在也无妨（线上用真实环境变量注入）。
	_ = godotenv.Load()

	var cfg Config

	// cleanenv 会利用反射，自动把新增的 OpenAIConfig 解析并填充好
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, fmt.Errorf("配置加载失败: %w", err)
	}

	return &cfg, nil
}
