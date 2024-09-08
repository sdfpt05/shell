package tests

import (
	"shell/internal/config"
	"shell/internal/shell"
	"testing"
)

func TestShellInitialization(t *testing.T) {
	cfg := &config.Config{}
	sh, err := shell.New(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize shell: %v", err)
	}
	if sh == nil {
		t.Fatal("Shell is nil after initialization")
	}
}
