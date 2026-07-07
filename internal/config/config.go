package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

// Config 是应用的全部配置
type Config struct {
	HTTP     HTTPConfig   `yaml:"http"     env-prefix:"HTTP_"`
	DeepSeek LLMConfig    `yaml:"deepseek" env-prefix:"DEEPSEEK_"`
	OpenAI   OpenAIConfig `yaml:"openai"   env-prefix:"OPENAI_"` // 新增 OpenAI 配置入口
	Log      LogConfig    `yaml:"log"      env-prefix:"LOG_"`
}

// HTTPConfig 保持不变
type HTTPConfig struct {
	Port string `yaml:"port" env:"PORT" env-default:"8091"`
}

// LLMConfig 目前专门给 DeepSeek 用
type LLMConfig struct {
	APIKey         string `yaml:"-"        env:"API_KEY"`
	BaseURL        string `yaml:"base_url" env:"BASE_URL" env-default:"https://api.deepseek.com"`
	Model          string `yaml:"model"    env:"MODEL"    env-default:"deepseek-chat"`
	TimeoutSeconds int    `yaml:"timeout"  env:"TIMEOUT"  env-default:"30"`
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
