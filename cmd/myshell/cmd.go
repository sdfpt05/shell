package main

import (
	"fmt"
	"os"

	"shell/internal/config"
	"shell/internal/shell"
)

func main() {
	cfg, err := config.Load("config.yml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	s, err := shell.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing shell: %v\n", err)
		os.Exit(1)
	}

	s.Run()
}
