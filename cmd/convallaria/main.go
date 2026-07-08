// cmd/convallaria/main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

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

	// Create LLM provider
	var provider llm.Provider
	if apiKey != "" {
		switch cfg.LLM.Provider {
		case "openai":
			fmt.Printf("Using OpenAI provider (model: %s)\n", cfg.LLM.Model)
			provider = llm.NewOpenAI(apiKey, cfg.LLM.Model)
		case "anthropic":
			fmt.Printf("Using Anthropic provider (model: %s)\n", cfg.LLM.Model)
			provider = llm.NewAnthropic(apiKey, cfg.LLM.Model)
		default:
			fmt.Printf("Using DeepSeek provider (model: %s)\n", cfg.LLM.Model)
			provider = llm.NewDeepSeek(apiKey, cfg.LLM.Model)
		}
	} else {
		fmt.Println("No API key found — using mock provider for demo")
		provider = llm.NewMockProvider()
		// Pre-populate with demo responses
		provider.(*llm.MockProvider).AddResponse(llm.MockTextResponse("Hello! I'm Convallaria, your coding agent. I can help you write, modify, and test code. What would you like to build today?"))
		provider.(*llm.MockProvider).AddResponse(llm.MockTextResponse("Sure! Let me write that for you. I'll create a simple Go program with a main function and proper error handling."))
		provider.(*llm.MockProvider).AddResponse(llm.MockTextResponse("The code looks good. I've added comments and followed Go best practices. Want me to write tests for it as well?"))
	}

	// Create agent
	ag := agent.New(agent.Config{
		MaxTurns:  cfg.Agent.MaxTurns,
		Provider:  provider,
		Workspace: cfg.Agent.Workspace,
	})

	// Create session manager (SQLite-backed for persistence)
	sessMgr, err := session.NewSQLiteManager("convallaria.db")
	if err != nil {
		log.Fatalf("Failed to create session manager: %v", err)
	}

	// Start server
	srv := server.New(server.Config{
		Port:      *port,
		StaticDir: "web",
	}, ag, sessMgr)

	fmt.Printf("Convallaria starting on http://localhost:%d\n", *port)

	// Start server in background
	httpServer := &http.Server{Addr: fmt.Sprintf(":%d", *port), Handler: srv}
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	fmt.Println("\nShutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Shutdown error: %v", err)
	}
	fmt.Println("Server stopped.")
}