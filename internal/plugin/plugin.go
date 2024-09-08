package plugin

import (
	"fmt"
	"plugin"
)

type Plugin interface {
	Name() string
	Execute(args []string) error
}

func Load(path string) (Plugin, error) {
	p, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin: %w", err)
	}

	symPlugin, err := p.Lookup("Plugin")
	if err != nil {
		return nil, fmt.Errorf("plugin does not export 'Plugin' symbol: %w", err)
	}

	plug, ok := symPlugin.(Plugin)
	if !ok {
		return nil, fmt.Errorf("plugin does not implement Plugin interface")
	}

	return plug, nil
}
