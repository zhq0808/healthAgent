package config

import "testing"

func TestResolveDerivedDefaultsMemoryExtractorInheritsChatModel(t *testing.T) {
	cfg := Config{DeepSeek: LLMConfig{Model: "deepseek/deepseek-v4-flash"}}

	cfg.resolveDerivedDefaults()

	if cfg.Memory.ExtractorModel != cfg.DeepSeek.Model {
		t.Fatalf("extractor model = %q, want %q", cfg.Memory.ExtractorModel, cfg.DeepSeek.Model)
	}
}

func TestResolveDerivedDefaultsKeepsExplicitMemoryExtractorModel(t *testing.T) {
	cfg := Config{
		DeepSeek: LLMConfig{Model: "deepseek/deepseek-v4-flash"},
		Memory:   MemoryConfig{ExtractorModel: "memory-model"},
	}

	cfg.resolveDerivedDefaults()

	if cfg.Memory.ExtractorModel != "memory-model" {
		t.Fatalf("extractor model = %q, want explicit override", cfg.Memory.ExtractorModel)
	}
}
