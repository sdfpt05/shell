package shell

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func (s *Shell) setupSignalHandling() {
	signal.Notify(s.signalChan, syscall.SIGINT, syscall.SIGTSTP, syscall.SIGCHLD)
	go s.handleSignals()
}

func (s *Shell) handleSignals() {
	for sig := range s.signalChan {
		switch sig {
		case syscall.SIGINT:
			fmt.Println("\nReceived SIGINT")
		case syscall.SIGTSTP:
			fmt.Println("\nReceived SIGTSTP")
		case syscall.SIGCHLD:
			s.reapChildren()
		}
	}
}

// Handle child process status changes
func (s *Shell) reapChildren() {
	for {
		pid, _ := syscall.Wait4(-1, nil, syscall.WNOHANG, nil)
		if pid <= 0 {
			break
		}
	}
}
