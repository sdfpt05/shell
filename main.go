package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/chzyer/readline"
	"github.com/kballard/go-shellquote"
)

const (
	historyFileName = ".shell_history"
	maxHistorySize  = 1000
)

type Shell struct {
	history        []string
	historyFile    string
	currentDir     string
	signalChan     chan os.Signal
	interruptCount int
	env            map[string]string
}

func NewShell() (*Shell, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("error getting home directory: %w", err)
	}

	historyFile := filepath.Join(homeDir, historyFileName)
	history, err := loadHistory(historyFile)
	if err != nil {
		return nil, fmt.Errorf("error loading history: %w", err)
	}

	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("error getting current directory: %w", err)
	}

	return &Shell{
		history:     history,
		historyFile: historyFile,
		currentDir:  currentDir,
		signalChan:  make(chan os.Signal, 1),
		env:         make(map[string]string),
	}, nil
}

func (s *Shell) Run() {
	signal.Notify(s.signalChan, syscall.SIGINT)
	defer signal.Stop(s.signalChan)

	rl, err := readline.NewEx(&readline.Config{
		Prompt:      fmt.Sprintf("%s $ ", s.currentDir),
		HistoryFile: s.historyFile,
	})
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt {
			if s.interruptCount++; s.interruptCount >= 2 {
				fmt.Println("\nForced exit")
				break
			}
			continue
		} else if err == io.EOF {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		s.addToHistory(line)

		if err := s.executeCommand(line); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}

		s.interruptCount = 0
		rl.SetPrompt(fmt.Sprintf("%s $ ", s.currentDir))
	}

	s.saveHistory()
}

func (s *Shell) executeCommand(input string) error {
	parts, err := shellquote.Split(input)
	if err != nil {
		return fmt.Errorf("error parsing command: %w", err)
	}

	if len(parts) == 0 {
		return nil
	}

	switch parts[0] {
	case "exit":
		s.saveHistory()
		os.Exit(0)
	case "history":
		s.printHistory()
		return nil
	case "cd":
		return s.changeDirectory(parts)
	case "export":
		return s.exportVar(parts)
	}

	return s.runExternal(parts)
}

func (s *Shell) runExternal(parts []string) error {
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Env = os.Environ()
	for k, v := range s.env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (s *Shell) changeDirectory(parts []string) error {
	if len(parts) < 2 {
		return fmt.Errorf("cd: missing argument")
	}
	path := os.ExpandEnv(parts[1])
	if err := os.Chdir(path); err != nil {
		return fmt.Errorf("cd: %s: %w", path, err)
	}
	var err error
	s.currentDir, err = os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting current directory: %w", err)
	}
	return nil
}

func (s *Shell) exportVar(parts []string) error {
	if len(parts) != 2 {
		return fmt.Errorf("export: invalid syntax")
	}
	kv := strings.SplitN(parts[1], "=", 2)
	if len(kv) != 2 {
		return fmt.Errorf("export: invalid syntax")
	}
	s.env[kv[0]] = kv[1]
	return nil
}

func (s *Shell) addToHistory(command string) {
	s.history = append(s.history, command)
	if len(s.history) > maxHistorySize {
		s.history = s.history[len(s.history)-maxHistorySize:]
	}
}

func (s *Shell) printHistory() {
	for i, cmd := range s.history {
		fmt.Printf("%d: %s\n", i+1, cmd)
	}
}

func (s *Shell) saveHistory() {
	if err := writeHistory(s.history, s.historyFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving history: %v\n", err)
	}
}

func loadHistory(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var history []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		history = append(history, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(history) > maxHistorySize {
		history = history[len(history)-maxHistorySize:]
	}

	return history, nil
}

func writeHistory(history []string, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range history {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return err
		}
	}

	return writer.Flush()
}

func main() {
	shell, err := NewShell()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing shell: %v\n", err)
		os.Exit(1)
	}

	shell.Run()
}
