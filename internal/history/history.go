package history

import (
	"bufio"
	"os"
	"sync"
)

type History struct {
	items    []string
	file     string
	maxItems int
	mu       sync.Mutex
}

func New(file string) (*History, error) {
	h := &History{
		file:     file,
		maxItems: 1000,
	}
	if err := h.load(); err != nil {
		return nil, err
	}
	return h, nil
}

func (h *History) Add(item string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.items = append(h.items, item)
	if len(h.items) > h.maxItems {
		h.items = h.items[len(h.items)-h.maxItems:]
	}
	h.save()
}

func (h *History) GetAll() []string {
	h.mu.Lock()
	defer h.mu.Unlock()

	return append([]string{}, h.items...)
}

func (h *History) load() error {
	file, err := os.Open(h.file)
	if err != nil {
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		h.items = append(h.items, scanner.Text())
	}
	return scanner.Err()
}

func (h *History) save() error {
	file, err := os.Create(h.file)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, item := range h.items {
		if _, err := writer.WriteString(item + "\n"); err != nil {
			return err
		}
	}
	return writer.Flush()
}
