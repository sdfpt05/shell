package main

import (
	"fmt"
	"shell/internal/plugin"
)

type ExamplePlugin struct{}

func (p *ExamplePlugin) Name() string {
	return "example"
}

func (p *ExamplePlugin) Execute(args []string) error {
	fmt.Println("Example plugin executed with args:", args)
	return nil
}

var Plugin ExamplePlugin
