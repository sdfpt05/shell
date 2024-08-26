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
	}, nil
}

func (s *Shell) Run() {
	signal.Notify(s.signalChan, syscall.SIGINT)
	defer signal.Stop(s.signalChan)

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s $ ", s.currentDir)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			continue
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		s.addToHistory(input)

		if err := s.executeCommand(input); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}

		s.interruptCount = 0
	}
}

func (s *Shell) executeCommand(input string) error {
	commands := strings.Split(input, "|")
	var cmds []*exec.Cmd
	var inputPipe io.Reader = os.Stdin

	for i, command := range commands {
		command = strings.TrimSpace(command)
		parts := strings.Fields(command)
		if len(parts) == 0 {
			continue
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
		}

		cmd := exec.Command(parts[0], parts[1:]...)
		cmd.Stdin = inputPipe
		cmd.Stderr = os.Stderr

		if i < len(commands)-1 {
			outputPipe, err := cmd.StdoutPipe()
			if err != nil {
				return fmt.Errorf("error creating stdout pipe: %w", err)
			}
			inputPipe = outputPipe
		} else {
			cmd.Stdout = os.Stdout
		}

		cmds = append(cmds, cmd)
	}

	return s.runCommands(cmds)
}

func (s *Shell) runCommands(cmds []*exec.Cmd) error {
	for _, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("error starting command: %w", err)
		}
	}

	done := make(chan error, len(cmds))
	for _, cmd := range cmds {
		go func(cmd *exec.Cmd) {
			done <- cmd.Wait()
		}(cmd)
	}

	go s.handleSignals(cmds)

	for range cmds {
		if err := <-done; err != nil {
			return fmt.Errorf("error waiting for command: %w", err)
		}
	}

	return nil
}

func (s *Shell) handleSignals(cmds []*exec.Cmd) {
	for sig := range s.signalChan {
		if sig == syscall.SIGINT {
			s.interruptCount++
			if s.interruptCount >= 2 {
				fmt.Println("\nForced exit")
				s.saveHistory()
				os.Exit(1)
			}
			fmt.Println("\nPress Ctrl+C again to exit")
			for _, cmd := range cmds {
				if cmd.Process != nil {
					_ = cmd.Process.Signal(sig)
				}
			}
		}
	}
}

func (s *Shell) changeDirectory(parts []string) error {
	if len(parts) < 2 {
		return fmt.Errorf("cd: missing argument")
	}
	if len(parts) > 2 {
		return fmt.Errorf("cd: too many arguments")
	}
	path := parts[1]
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
