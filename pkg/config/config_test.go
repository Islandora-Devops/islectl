package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndSave(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmpDir)

	cfg := &Config{
		CurrentContext: "test-context",
		Contexts: []Context{
			{
				Name:           "test-context",
				DockerHostType: ContextLocal,
				DockerSocket:   "/var/run/docker.sock",
			},
		},
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.CurrentContext != cfg.CurrentContext {
		t.Errorf("expected current context %s, got %s", cfg.CurrentContext, loaded.CurrentContext)
	}
	if len(loaded.Contexts) != len(cfg.Contexts) {
		t.Errorf("expected %d contexts, got %d", len(cfg.Contexts), len(loaded.Contexts))
	}
	if loaded.Contexts[0].Name != cfg.Contexts[0].Name {
		t.Errorf("expected context name %s, got %s", cfg.Contexts[0].Name, loaded.Contexts[0].Name)
	}
}

func TestLoadEmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmpDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if cfg.CurrentContext != "" {
		t.Errorf("expected empty current context, got %s", cfg.CurrentContext)
	}
	if len(cfg.Contexts) != 0 {
		t.Errorf("expected 0 contexts, got %d", len(cfg.Contexts))
	}
}

func TestConfigFilePath(t *testing.T) {
	home := os.Getenv("HOME")
	expected := filepath.Join(home, ".islectl", "config.yaml")
	if path := ConfigFilePath(); path != expected {
		t.Errorf("expected config file path %s, got %s", expected, path)
	}
}
