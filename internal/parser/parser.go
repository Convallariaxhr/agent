// internal/parser/parser.go
package parser

import (
	"encoding/json"

	"github.com/Convallariaxhr/convallaria/internal/llm"
)

// Action represents a parsed tool call that the agent should execute.
type Action struct {
	ToolCallID string
	Tool       string
	Params     map[string]any
	ParseError error // non-nil if arguments JSON was malformed
}

// ActionList is a list of actions with helper methods.
type ActionList []Action

// IsStop returns true if the LLM indicated it's done (no tool calls).
func (al ActionList) IsStop() bool {
	return len(al) == 0
}

// Parse extracts actions from an LLM response.
func Parse(resp *llm.Response) ActionList {
	actions := make([]Action, 0, len(resp.ToolCalls))
	for _, tc := range resp.ToolCalls {
		action := Action{
			ToolCallID: tc.ID,
			Tool:       tc.Function.Name,
		}
		var params map[string]any
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
			action.ParseError = err
			action.Params = map[string]any{"_raw": tc.Function.Arguments}
		} else {
			action.Params = params
		}
		actions = append(actions, action)
	}
	return ActionList(actions)
}