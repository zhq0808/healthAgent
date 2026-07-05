package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config 是应用的全部配置。敏感项（LLM API Key）只从环境变量读取，不落 yaml。
type Config struct {
	HTTP HTTPConfig `yaml:"http"`
	DB   DBConfig   `yaml:"db"`
	LLM  LLMConfig  `yaml:"llm"`
	Log  LogConfig  `yaml:"log"`
}

// HTTPConfig 是 HTTP 服务配置。
type HTTPConfig struct {
	Port string `yaml:"port"`
}

// DBConfig 是数据库配置。P0 使用 SQLite。
type DBConfig struct {
	DSN string `yaml:"dsn"`
}

// LLMConfig 是大模型配置。APIKey 只从环境变量注入。
type LLMConfig struct {
	APIKey         string `yaml:"-"`
	BaseURL        string `yaml:"base_url"`
	Model          string `yaml:"model"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
}

// LogConfig 是日志配置。Debug 为 true 时才允许打印敏感原文，仅限本地。
type LogConfig struct {
	Level string `yaml:"level"`
	Debug bool   `yaml:"debug"`
}

// defaultConfig 返回带兜底值的配置，防止 yaml 缺项时出现零值。
func defaultConfig() *Config {
	return &Config{
		HTTP: HTTPConfig{Port: "8080"},
		DB:   DBConfig{DSN: "./data/health.db"},
		LLM:  LLMConfig{BaseURL: "https://api.deepseek.com", Model: "deepseek-chat", TimeoutSeconds: 30},
		Log:  LogConfig{Level: "info", Debug: false},
	}
}

// Load 加载配置：先读 yaml 文件，再用环境变量覆盖敏感/可覆盖项。
// yaml 文件不存在时使用默认值，不视为错误。
func Load(path string) (*Config, error) {
	// 尽力加载 .env，不存在也无妨。
	_ = godotenv.Load()

	cfg := defaultConfig()

	if data, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("解析配置文件 %s 失败: %w", path, err)
		}
	}

	cfg.overrideFromEnv()

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// overrideFromEnv 用环境变量覆盖配置。敏感项（API Key）仅此一处来源。
func (c *Config) overrideFromEnv() {
	if v := os.Getenv("HTTP_PORT"); v != "" {
		c.HTTP.Port = v
	}
	if v := os.Getenv("DB_DSN"); v != "" {
		c.DB.DSN = v
	}
	if v := os.Getenv("DEEPSEEK_API_KEY"); v != "" {
		c.LLM.APIKey = v
	}
	if v := os.Getenv("DEEPSEEK_BASE_URL"); v != "" {
		c.LLM.BaseURL = v
	}
	if v := os.Getenv("DEEPSEEK_MODEL"); v != "" {
		c.LLM.Model = v
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		c.Log.Level = v
	}
	if v := os.Getenv("LOG_DEBUG"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			c.Log.Debug = b
		}
	}
}

// validate 校验必填项。P0 阶段 LLM Key 允许缺省（骨架可先启动），仅在使用时才要求。
func (c *Config) validate() error {
	if c.HTTP.Port == "" {
		return fmt.Errorf("http.port 不能为空")
	}
	if c.DB.DSN == "" {
		return fmt.Errorf("db.dsn 不能为空")
	}
	return nil
}
