package handlers

import (
	"encoding/json"
	"strings"
)

// normalizeMessagesInJSON converts Anthropic-format tool messages to OpenAI format within a raw JSON body.
//
// Anthropic format (sent by Kilo Code / Claude SDK clients):
//
//	Assistant: content = [{"type":"tool_use","id":"...","name":"...","input":{...}}, ...]
//	User:      content = [{"type":"tool_result","tool_use_id":"...","content":[{"type":"text","text":"..."}]}]
//
// OpenAI format (expected by vLLM / SGLang / TGI):
//
//	Assistant: content = "text", tool_calls = [{"id":"...","type":"function","function":{"name":"...","arguments":"..."}}]
//	Tool:      role = "tool", content = "...", tool_call_id = "..."
//
// If messages are already in OpenAI format (content is a string, no tool_use blocks) the body is
// returned unchanged.  This function is a no-op for purely OpenAI-format traffic.
func normalizeMessagesInJSON(body []byte) []byte {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return body
	}

	msgsRaw, ok := raw["messages"]
	if !ok {
		return body
	}

	var messages []map[string]json.RawMessage
	if err := json.Unmarshal(msgsRaw, &messages); err != nil {
		return body
	}

	converted, changed := convertAnthropicMessages(messages)
	if !changed {
		return body
	}

	newMsgs, err := json.Marshal(converted)
	if err != nil {
		return body
	}
	raw["messages"] = newMsgs

	result, err := json.Marshal(raw)
	if err != nil {
		return body
	}
	return result
}

// convertAnthropicMessages iterates messages, expanding Anthropic-format blocks into OpenAI messages.
// A single Anthropic user message with N tool_result blocks becomes N role:"tool" messages.
// Returns the converted slice and whether any change was made.
func convertAnthropicMessages(messages []map[string]json.RawMessage) ([]map[string]json.RawMessage, bool) {
	result := make([]map[string]json.RawMessage, 0, len(messages))
	changed := false

	for _, msg := range messages {
		roleRaw, hasRole := msg["role"]
		contentRaw, hasContent := msg["content"]
		if !hasRole || !hasContent {
			result = append(result, msg)
			continue
		}

		var role string
		if err := json.Unmarshal(roleRaw, &role); err != nil {
			result = append(result, msg)
			continue
		}

		// Content must be a JSON array to be Anthropic format; string = already OpenAI
		var blocks []json.RawMessage
		if err := json.Unmarshal(contentRaw, &blocks); err != nil || len(blocks) == 0 {
			result = append(result, msg)
			continue
		}

		// Peek at the first block's type
		var firstBlock struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(blocks[0], &firstBlock); err != nil {
			result = append(result, msg)
			continue
		}

		switch role {
		case "assistant":
			// Convert if any block is tool_use
			if firstBlock.Type == "tool_use" || firstBlock.Type == "text" {
				if converted, ok := convertAssistantMessage(msg, blocks); ok {
					result = append(result, converted)
					changed = true
					continue
				}
			}
		case "user":
			// Expand tool_result blocks into separate role:"tool" messages
			if firstBlock.Type == "tool_result" {
				if toolMsgs, ok := convertToolResultMessages(blocks); ok {
					result = append(result, toolMsgs...)
					changed = true
					continue
				}
			}
		}

		result = append(result, msg)
	}

	return result, changed
}

// convertAssistantMessage converts an Anthropic assistant message with tool_use blocks to OpenAI format.
//
//	Input:  {"role":"assistant","content":[{"type":"text","text":"..."}, {"type":"tool_use","id":"...","name":"...","input":{...}}]}
//	Output: {"role":"assistant","content":"...","tool_calls":[{"id":"...","type":"function","function":{...}}]}
func convertAssistantMessage(orig map[string]json.RawMessage, blocks []json.RawMessage) (map[string]json.RawMessage, bool) {
	var textParts []string
	var toolCalls []map[string]any
	hasToolUse := false

	for _, blockRaw := range blocks {
		var b struct {
			Type  string          `json:"type"`
			ID    string          `json:"id"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
			Text  string          `json:"text"`
		}
		if err := json.Unmarshal(blockRaw, &b); err != nil {
			continue
		}

		switch b.Type {
		case "text":
			if b.Text != "" {
				textParts = append(textParts, b.Text)
			}
		case "tool_use":
			hasToolUse = true
			argsStr := "{}"
			if len(b.Input) > 0 && string(b.Input) != "null" {
				argsStr = string(b.Input)
			}
			toolCalls = append(toolCalls, map[string]any{
				"id":   b.ID,
				"type": "function",
				"function": map[string]any{
					"name":      b.Name,
					"arguments": argsStr,
				},
			})
		}
	}

	if !hasToolUse {
		return orig, false
	}

	// Build OpenAI assistant message — copy all non-content fields first
	out := make(map[string]json.RawMessage, len(orig))
	for k, v := range orig {
		if k != "content" && k != "tool_calls" {
			out[k] = v
		}
	}

	// content: joined text or null
	if len(textParts) > 0 {
		contentJSON, _ := json.Marshal(strings.Join(textParts, ""))
		out["content"] = contentJSON
	} else {
		out["content"] = json.RawMessage("null")
	}

	// tool_calls
	if len(toolCalls) > 0 {
		tcJSON, err := json.Marshal(toolCalls)
		if err == nil {
			out["tool_calls"] = tcJSON
		}
	}

	return out, true
}

// convertToolResultMessages converts an Anthropic user message with tool_result blocks into
// one role:"tool" message per block.
//
//	Input:  {"role":"user","content":[{"type":"tool_result","tool_use_id":"...","content":[{"type":"text","text":"..."}]}]}
//	Output: [{"role":"tool","content":"...","tool_call_id":"..."}]
func convertToolResultMessages(blocks []json.RawMessage) ([]map[string]json.RawMessage, bool) {
	var results []map[string]json.RawMessage

	for _, blockRaw := range blocks {
		var b struct {
			Type      string          `json:"type"`
			ToolUseID string          `json:"tool_use_id"`
			Content   json.RawMessage `json:"content"`
		}
		if err := json.Unmarshal(blockRaw, &b); err != nil || b.Type != "tool_result" {
			continue
		}

		contentStr := extractToolResultContent(b.Content)

		roleJSON, _ := json.Marshal("tool")
		contentJSON, _ := json.Marshal(contentStr)
		toolIDJSON, _ := json.Marshal(b.ToolUseID)

		results = append(results, map[string]json.RawMessage{
			"role":         roleJSON,
			"content":      contentJSON,
			"tool_call_id": toolIDJSON,
		})
	}

	return results, len(results) > 0
}

// extractToolResultContent extracts plain text from Anthropic tool_result content, which can be:
//   - a JSON string:  "result text"
//   - an array of text blocks: [{"type":"text","text":"..."}]
//   - null / empty
func extractToolResultContent(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}

	// Try plain string first
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}

	// Try array of typed blocks
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		var parts []string
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				parts = append(parts, b.Text)
			}
		}
		return strings.Join(parts, "\n")
	}

	// Fallback: raw JSON as string (shouldn't normally happen)
	return string(raw)
}
