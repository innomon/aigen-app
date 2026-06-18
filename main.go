package main

import (
	"os"

	"github.com/innomon/aigen-app/framework"
	"github.com/innomon/aigen-app/utils/logger"
)

func main() {
	configPath := ""
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	config, err := framework.LoadConfig(configPath)
	if err != nil {
		logger.Fatalf("Error loading configuration: %v", err)
	}

	if err := framework.Start(config); err != nil {
		logger.Fatalf("Framework failed to start: %v", err)
	}
}
