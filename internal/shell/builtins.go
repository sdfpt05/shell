package shell

import (
	"fmt"
	"os"
)

func (s *Shell) executeBuiltin(args []string) (bool, error) {
	switch args[0] {
	case "cd":
		return true, s.changeDirectory(args[1:])
	case "exit":
		s.exit()
		return true, nil
	case "history":
		return true, s.showHistory()
	default:
		return false, nil
	}
}

func (s *Shell) changeDirectory(args []string) error {
	var dir string
	if len(args) == 0 {
		dir = s.config.HomeDir
	} else {
		dir = args[0]
	}

	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("cd: %w", err)
	}
	return nil
}

func (s *Shell) exit() {
	os.Exit(0)
}

func (s *Shell) showHistory() error {
	for i, cmd := range s.history.GetAll() {
		fmt.Printf("%d: %s\n", i+1, cmd)
	}
	return nil
}
