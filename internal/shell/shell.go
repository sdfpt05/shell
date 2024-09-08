package shell

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"shell/internal/config"
	"shell/internal/history"
	"shell/internal/plugin"
)

type Shell struct {
	config     *config.Config
	history    *history.History
	plugins    []plugin.Plugin
	jobs       map[int]*Job
	nextJobID  int
	signalChan chan os.Signal
	reader     *readline.Instance
}

func New(cfg *config.Config) (*Shell, error) {
	hist, err := history.New(cfg.HistoryFile)
	if err != nil {
		return nil, fmt.Errorf("error initializing history: %w", err)
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:      "> ",
		HistoryFile: cfg.HistoryFile,
	})
	if err != nil {
		return nil, fmt.Errorf("error initializing readline: %w", err)
	}

	return &Shell{
		config:     cfg,
		history:    hist,
		jobs:       make(map[int]*Job),
		nextJobID:  1,
		signalChan: make(chan os.Signal, 1),
		reader:     rl,
	}, nil
}

func (s *Shell) Run() {
	for {
		line, err := s.reader.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		s.history.Add(line)

		if err := s.Execute(line); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
	}
}

func (s *Shell) Execute(input string) error {
	args := strings.Split(input, " ")
	if ok, err := s.executeBuiltin(args); ok {
		return err
	}
	return s.runExternal(args)
}
