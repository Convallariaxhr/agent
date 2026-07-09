// internal/tools/registry.go
package tools

import (
	"context"
	"errors"
	"sync"
)

var ErrUnknownTool = errors.New("unknown tool")

// Result is the output of a tool execution.
type Result struct {
	Output  string `json:"output"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// Tool defines the interface for all executable tools.
type Tool interface {
	Name() string
	Description() string
	Schema() map[string]any // JSON Schema for function calling
	Execute(ctx context.Context, params map[string]any) (*Result, error)
}

// Registry manages tool registration and dispatch.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

func (r *Registry) Register(name string, tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[name] = tool
}

func (r *Registry) Execute(ctx context.Context, name string, params map[string]any) (*Result, error) {
	r.mu.RLock()
	tool, ok := r.tools[name]
	r.mu.RUnlock()
	if !ok {
		return &Result{Success: false, Error: "unknown tool: " + name}, ErrUnknownTool
	}
	return tool.Execute(ctx, params)
}

func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tools := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}