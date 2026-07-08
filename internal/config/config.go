// internal/config/config.go
package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for Convallaria.
type Config struct {
	LLM        LLMConfig        `yaml:"llm"`
	Agent      AgentConfig      `yaml:"agent"`
	Context    ContextConfig    `yaml:"context"`
	Tools      ToolsConfig      `yaml:"tools"`
	Guardrails GuardrailsConfig `yaml:"guardrails"`
	Memory     MemoryConfig     `yaml:"memory"`
}

type LLMConfig struct {
	Provider    string  `yaml:"provider"`
	Model       string  `yaml:"model"`
	MaxTokens   int     `yaml:"max_tokens"`
	Temperature float64 `yaml:"temperature"`
	APIKeyEnv   string  `yaml:"api_key_env"`
}

type AgentConfig struct {
	MaxTurns  int    `yaml:"max_turns"`
	Workspace string `yaml:"workspace"`
}

type ContextConfig struct {
	MaxContextTokens     int     `yaml:"max_context_tokens"`
	CompressionThreshold float64 `yaml:"compression_threshold"`
}

type ToolsConfig struct {
	Shell ShellConfig `yaml:"shell"`
	File  FileConfig  `yaml:"file"`
	Git   GitConfig   `yaml:"git"`
}

type ShellConfig struct {
	Enabled bool `yaml:"enabled"`
	Timeout int  `yaml:"timeout"`
}

type FileConfig struct {
	AllowedPaths []string `yaml:"allowed_paths"`
}

type GitConfig struct {
	AutoCommit bool `yaml:"auto_commit"`
}

type GuardrailsConfig struct {
	DangerousCommands bool `yaml:"dangerous_commands"`
	FileScope         bool `yaml:"file_scope"`
	GitDangerousOps   bool `yaml:"git_dangerous_ops"`
}

type MemoryConfig struct {
	VectorStore    string `yaml:"vector_store"`
	TopK           int    `yaml:"top_k"`
	EmbeddingModel string `yaml:"embedding_model"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		LLM: LLMConfig{
			Provider:    "deepseek",
			Model:       "deepseek-chat",
			MaxTokens:   4096,
			Temperature: 0.0,
			APIKeyEnv:   "DEEPSEEK_API_KEY",
		},
		Agent: AgentConfig{
			MaxTurns:  30,
			Workspace: ".",
		},
		Context: ContextConfig{
			MaxContextTokens:     64000,
			CompressionThreshold: 0.8,
		},
		Tools: ToolsConfig{
			Shell: ShellConfig{Enabled: true, Timeout: 120},
			File:  FileConfig{AllowedPaths: []string{"."}},
			Git:   GitConfig{AutoCommit: false},
		},
		Guardrails: GuardrailsConfig{
			DangerousCommands: true,
			FileScope:         true,
			GitDangerousOps:   true,
		},
		Memory: MemoryConfig{
			VectorStore:    "sqlite",
			TopK:           5,
			EmbeddingModel: "local",
		},
	}
}

// Load reads a YAML config file and applies defaults + env overrides.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg.applyEnvOverrides()
			return cfg, nil
		}
		return nil, err
	}

	if len(data) > 0 {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	cfg.applyEnvOverrides()
	return cfg, nil
}

func (c *Config) applyEnvOverrides() {
	if v := os.Getenv("CONVALLARIA_PROVIDER"); v != "" {
		c.LLM.Provider = v
	}
	if v := os.Getenv("CONVALLARIA_MODEL"); v != "" {
		c.LLM.Model = v
	}
	if v := os.Getenv("CONVALLARIA_API_KEY"); v != "" {
		c.LLM.APIKeyEnv = "CONVALLARIA_API_KEY"
		_ = v
	}
}