package main

import (
	"log"
	"os"

	"github.com/innomon/aigen-cms/framework"
)

func main() {
	configPath := ""
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	config, err := framework.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	if err := framework.Start(config); err != nil {
		log.Fatalf("Framework failed to start: %v", err)
	}
}
