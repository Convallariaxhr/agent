// internal/config/config_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_FromYAML(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `
llm:
  provider: deepseek
  model: deepseek-chat
  max_tokens: 4096
agent:
  max_turns: 50
  workspace: /tmp/test
`
	yamlPath := filepath.Join(dir, "convallaria.yaml")
	os.WriteFile(yamlPath, []byte(yamlContent), 0644)

	cfg, err := Load(yamlPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LLM.Provider != "deepseek" {
		t.Errorf("expected provider deepseek, got %q", cfg.LLM.Provider)
	}
	if cfg.LLM.MaxTokens != 4096 {
		t.Errorf("expected max_tokens 4096, got %d", cfg.LLM.MaxTokens)
	}
	if cfg.Agent.MaxTurns != 50 {
		t.Errorf("expected max_turns 50, got %d", cfg.Agent.MaxTurns)
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "convallaria.yaml")
	os.WriteFile(yamlPath, []byte(""), 0644)

	cfg, err := Load(yamlPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Agent.MaxTurns != 30 {
		t.Errorf("expected default max_turns 30, got %d", cfg.Agent.MaxTurns)
	}
	if cfg.LLM.Provider != "deepseek" {
		t.Errorf("expected default provider deepseek, got %q", cfg.LLM.Provider)
	}
}

func TestLoadConfig_EnvOverride(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "convallaria.yaml")
	os.WriteFile(yamlPath, []byte("llm:\n  provider: openai"), 0644)

	os.Setenv("CONVALLARIA_PROVIDER", "deepseek")
	defer os.Unsetenv("CONVALLARIA_PROVIDER")

	cfg, err := Load(yamlPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LLM.Provider != "deepseek" {
		t.Errorf("expected env override deepseek, got %q", cfg.LLM.Provider)
	}
}