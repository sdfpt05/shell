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

type Job struct {
	cmd      *exec.Cmd
	status   string
	jobID    int
	stopChan chan struct{}
}

type Shell struct {
	history        []string
	historyFile    string
	currentDir     string
	signalChan     chan os.Signal
	interruptCount int
	env            map[string]string
	aliases        map[string]string
	jobs           map[int]*Job
	nextJobID      int
	variables      map[string]string
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
		aliases:     make(map[string]string),
		jobs:        make(map[int]*Job),
		nextJobID:   1,
		variables:   make(map[string]string),
	}, nil
}

func (s *Shell) Run() {
	signal.Notify(s.signalChan, syscall.SIGINT, syscall.SIGTSTP, syscall.SIGCHLD)
	defer signal.Stop(s.signalChan)

	go s.handleSignals()

	rl, err := readline.NewEx(&readline.Config{
		Prompt:      s.getPrompt(),
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
		rl.SetPrompt(s.getPrompt())
	}

	s.saveHistory()
}

func (s *Shell) getPrompt() string {
	return fmt.Sprintf("%s $ ", s.currentDir)
}

func (s *Shell) executeCommand(input string) error {
	// Expand variables
	for k, v := range s.variables {
		input = strings.ReplaceAll(input, "$"+k, v)
	}

	parts, err := shellquote.Split(input)
	if err != nil {
		return fmt.Errorf("error parsing command: %w", err)
	}

	if len(parts) == 0 {
		return nil
	}

	// Check for aliases
	if alias, ok := s.aliases[parts[0]]; ok {
		aliasParts, err := shellquote.Split(alias)
		if err != nil {
			return fmt.Errorf("error parsing alias: %w", err)
		}
		parts = append(aliasParts, parts[1:]...)
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
	case "alias":
		return s.setAlias(parts)
	case "jobs":
		s.listJobs()
		return nil
	case "fg":
		return s.foregroundJob(parts)
	case "bg":
		return s.backgroundJob(parts)
	case "set":
		return s.setVariable(parts)
	}

	return s.runExternal(parts)
}

func (s *Shell) runExternal(parts []string) error {
	background := false
	if parts[len(parts)-1] == "&" {
		background = true
		parts = parts[:len(parts)-1]
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Env = os.Environ()
	for k, v := range s.env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	if background {
		cmd.Stdin = nil
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Start(); err != nil {
			return err
		}
		job := &Job{
			cmd:      cmd,
			status:   "Running",
			jobID:    s.nextJobID,
			stopChan: make(chan struct{}),
		}
		s.jobs[s.nextJobID] = job
		s.nextJobID++
		fmt.Printf("[%d] %d\n", job.jobID, cmd.Process.Pid)
		go s.waitForJob(job)
		return nil
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (s *Shell) waitForJob(job *Job) {
	err := job.cmd.Wait()
	select {
	case <-job.stopChan:
		return
	default:
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				job.status = fmt.Sprintf("Exited (%d)", exitErr.ExitCode())
			} else {
				job.status = "Errored"
			}
		} else {
			job.status = "Done"
		}
		fmt.Printf("[%d]+ %s\t%s\n", job.jobID, job.status, job.cmd.Args[0])
	}
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

func (s *Shell) setAlias(parts []string) error {
	if len(parts) != 2 {
		return fmt.Errorf("alias: invalid syntax")
	}
	kv := strings.SplitN(parts[1], "=", 2)
	if len(kv) != 2 {
		return fmt.Errorf("alias: invalid syntax")
	}
	s.aliases[kv[0]] = kv[1]
	return nil
}

func (s *Shell) listJobs() {
	for _, job := range s.jobs {
		fmt.Printf("[%d] %s\t%s\n", job.jobID, job.status, job.cmd.Args[0])
	}
}

func (s *Shell) foregroundJob(parts []string) error {
	if len(parts) != 2 {
		return fmt.Errorf("fg: invalid syntax")
	}
	jobID := 0
	_, err := fmt.Sscanf(parts[1], "%d", &jobID)
	if err != nil {
		return fmt.Errorf("fg: invalid job ID")
	}
	job, ok := s.jobs[jobID]
	if !ok {
		return fmt.Errorf("fg: job not found")
	}
	delete(s.jobs, jobID)
	close(job.stopChan)

	job.cmd.Stdin = os.Stdin
	job.cmd.Stdout = os.Stdout
	job.cmd.Stderr = os.Stderr

	err = job.cmd.Wait()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			fmt.Printf("Exited (%d)\n", exitErr.ExitCode())
		} else {
			fmt.Printf("Error: %v\n", err)
		}
	}
	return nil
}

func (s *Shell) backgroundJob(parts []string) error {
	if len(parts) != 2 {
		return fmt.Errorf("bg: invalid syntax")
	}
	jobID := 0
	_, err := fmt.Sscanf(parts[1], "%d", &jobID)
	if err != nil {
		return fmt.Errorf("bg: invalid job ID")
	}
	job, ok := s.jobs[jobID]
	if !ok {
		return fmt.Errorf("bg: job not found")
	}
	if job.status != "Stopped" {
		return fmt.Errorf("bg: job is not stopped")
	}
	job.status = "Running"
	job.cmd.Process.Signal(syscall.SIGCONT)
	return nil
}

func (s *Shell) setVariable(parts []string) error {
	if len(parts) != 2 {
		return fmt.Errorf("set: invalid syntax")
	}
	kv := strings.SplitN(parts[1], "=", 2)
	if len(kv) != 2 {
		return fmt.Errorf("set: invalid syntax")
	}
	s.variables[kv[0]] = kv[1]
	return nil
}

func (s *Shell) handleSignals() {
	for sig := range s.signalChan {
		switch sig {
		case syscall.SIGINT:
			// Handle Ctrl+C
			fmt.Println("\nInterrupted")
		case syscall.SIGTSTP:
			// Handle Ctrl+Z
			fmt.Println("\nStopped")
		case syscall.SIGCHLD:
			// Handle child process status changes
			s.reapChildren()
		}
	}
}

func (s *Shell) reapChildren() {
	for {
		pid, _ := syscall.Wait4(-1, nil, syscall.WNOHANG, nil)
		if pid <= 0 {
			break
		}
	}
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
