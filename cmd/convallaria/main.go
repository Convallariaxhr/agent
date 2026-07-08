// cmd/convallaria/main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Convallariaxhr/convallaria/internal/agent"
	"github.com/Convallariaxhr/convallaria/internal/config"
	"github.com/Convallariaxhr/convallaria/internal/credential"
	"github.com/Convallariaxhr/convallaria/internal/llm"
	"github.com/Convallariaxhr/convallaria/internal/server"
	"github.com/Convallariaxhr/convallaria/internal/session"
)

func main() {
	configPath := flag.String("config", "convallaria.yaml", "Path to config file")
	port := flag.Int("port", 8080, "HTTP server port")
	flag.Parse()

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Load credentials
	store := credential.NewMemoryStore()
	apiKey, err := store.Get(cfg.LLM.Provider)
	if err != nil {
		apiKey = os.Getenv(cfg.LLM.APIKeyEnv)
	}
	if apiKey == "" {
		fmt.Println("Warning: No API key found. Run 'convallaria init' to configure.")
	}

	// Create LLM provider (currently only mock is supported without real API)
	// In production: deepseek.New(apiKey), openai.New(apiKey), etc.
	provider := llm.NewMockProvider()
	// Pre-populate with demo responses for testing
	provider.AddResponse(llm.MockTextResponse("Hello! I'm Convallaria, your coding agent. I can help you write, modify, and test code. What would you like to build today?"))
	provider.AddResponse(llm.MockTextResponse("Sure! Let me write that for you. I'll create a simple Go program with a main function and proper error handling."))
	provider.AddResponse(llm.MockTextResponse("The code looks good. I've added comments and followed Go best practices. Want me to write tests for it as well?"))

	// Create agent
	ag := agent.New(agent.Config{
		MaxTurns:  cfg.Agent.MaxTurns,
		Provider:  provider,
		Workspace: cfg.Agent.Workspace,
	})

	// Create session manager
	sessMgr := session.NewManager()

	// Start server
	srv := server.New(server.Config{
		Port:      *port,
		StaticDir: "web",
	}, ag, sessMgr)

	fmt.Printf("Convallaria starting on http://localhost:%d\n", *port)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}