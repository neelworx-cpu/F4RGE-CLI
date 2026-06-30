package gateway

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/google"
	"github.com/google/uuid"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/controlplane"
	f4rgesession "github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/session"
)

type Provider struct{}

func NewProvider() Provider {
	return Provider{}
}

func (Provider) Name() string {
	return "f4rge-gateway"
}

func (Provider) LanguageModel(_ context.Context, modelID string) (fantasy.LanguageModel, error) {
	return languageModel{modelID: modelID}, nil
}

type languageModel struct {
	modelID string
}

type gatewayEvent struct {
	Type             string          `json:"type"`
	Delta            string          `json:"delta"`
	ToolCallID       string          `json:"toolCallId"`
	Name             string          `json:"name"`
	ArgumentsJSON    string          `json:"argumentsJson"`
	ProviderMetadata json.RawMessage `json:"providerMetadata"`
	Code             string          `json:"code"`
	Message          string          `json:"message"`
	Retryable        bool            `json:"retryable"`
	InputTokens      int64           `json:"inputTokens"`
	OutputTokens     int64           `json:"outputTokens"`
	FinishReason     string          `json:"finishReason"`
}

func (m languageModel) Provider() string {
	return "f4rge-gateway"
}

func (m languageModel) Model() string {
	return m.modelID
}

func (m languageModel) Generate(ctx context.Context, call fantasy.Call) (*fantasy.Response, error) {
	stream, err := m.Stream(ctx, call)
	if err != nil {
		return nil, err
	}
	var text strings.Builder
	var usage fantasy.Usage
	for part := range stream {
		switch part.Type {
		case fantasy.StreamPartTypeTextDelta:
			text.WriteString(part.Delta)
		case fantasy.StreamPartTypeFinish:
			usage = part.Usage
		case fantasy.StreamPartTypeError:
			if part.Error != nil {
				return nil, part.Error
			}
		}
	}
	return &fantasy.Response{
		Content: fantasy.ResponseContent{fantasy.TextContent{Text: text.String()}},
		Usage:   usage,
	}, nil
}

func (m languageModel) Stream(ctx context.Context, call fantasy.Call) (fantasy.StreamResponse, error) {
	session, err := f4rgesession.Load()
	if err != nil {
		return nil, err
	}
	if !f4rgesession.IsUsable(session) {
		return nil, fmt.Errorf("F4RGE sign-in is required. Open 4RGED and use the F4RGE sign-in dialog")
	}
	request := controlplane.InferenceRequest{
		RequestID:      uuid.NewString(),
		Surface:        "cli",
		OrganizationID: session.OrganizationID,
		SessionID:      session.RuntimeSessionID,
		ModelID:        m.modelID,
		PromptMode:     "agent",
		Messages:       promptToGatewayMessages(call.Prompt),
		Tools:          toolsToGatewayTools(call.Tools),
		ToolResults:    promptToGatewayToolResults(call.Prompt),
	}
	body, err := controlplane.New().StreamInference(ctx, session, request)
	if err != nil {
		return nil, err
	}
	return func(yield func(fantasy.StreamPart) bool) {
		defer body.Close()
		emittedContent := false
		sawToolResult := len(promptToGatewayToolResults(call.Prompt)) > 0
		var usage fantasy.Usage
		scanner := bufio.NewScanner(body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			var event gatewayEvent
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &event); err != nil {
				if !yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeError, Error: err}) {
					return
				}
				continue
			}
			if event.Type == "usage" {
				usage.InputTokens = event.InputTokens
				usage.OutputTokens = event.OutputTokens
				usage.TotalTokens = event.InputTokens + event.OutputTokens
				continue
			}
			part, ok := eventToStreamPart(event, usage)
			if ok && !yield(part) {
				return
			}
			if ok && (part.Type == fantasy.StreamPartTypeTextDelta || part.Type == fantasy.StreamPartTypeToolCall) {
				emittedContent = true
			}
		}
		if err := scanner.Err(); err != nil && err != io.EOF {
			yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeError, Error: err})
			return
		}
		if !emittedContent {
			if sawToolResult {
				yield(fantasy.StreamPart{
					Type:         fantasy.StreamPartTypeFinish,
					FinishReason: fantasy.FinishReasonStop,
					Usage:        usage,
				})
				return
			}
			yield(fantasy.StreamPart{
				Type:  fantasy.StreamPartTypeError,
				Error: fmt.Errorf("F4RGE Gateway completed without text or tool calls"),
			})
		}
	}, nil
}

func (m languageModel) GenerateObject(context.Context, fantasy.ObjectCall) (*fantasy.ObjectResponse, error) {
	return nil, fmt.Errorf("F4RGE Gateway object generation is not implemented")
}

func (m languageModel) StreamObject(context.Context, fantasy.ObjectCall) (fantasy.ObjectStreamResponse, error) {
	return nil, fmt.Errorf("F4RGE Gateway object streaming is not implemented")
}

func toolsToGatewayTools(tools []fantasy.Tool) []controlplane.InferenceTool {
	result := make([]controlplane.InferenceTool, 0, len(tools))
	for _, tool := range tools {
		switch t := tool.(type) {
		case fantasy.FunctionTool:
			result = append(result, controlplane.InferenceTool{
				Name:        t.Name,
				Description: t.Description,
				InputSchema: t.InputSchema,
			})
		case *fantasy.FunctionTool:
			if t != nil {
				result = append(result, controlplane.InferenceTool{
					Name:        t.Name,
					Description: t.Description,
					InputSchema: t.InputSchema,
				})
			}
		default:
			if tool.GetName() != "" {
				result = append(result, controlplane.InferenceTool{Name: tool.GetName()})
			}
		}
	}
	return result
}

func promptToGatewayMessages(prompt fantasy.Prompt) []controlplane.InferenceMessage {
	messages := make([]controlplane.InferenceMessage, 0, len(prompt))
	for _, msg := range prompt {
		var text strings.Builder
		var toolCalls []controlplane.InferenceToolCall
		for _, part := range msg.Content {
			if content, ok := fantasy.AsMessagePart[fantasy.TextPart](part); ok {
				text.WriteString(content.Text)
			}
			if call, ok := fantasy.AsMessagePart[fantasy.ToolCallPart](part); ok {
				toolCalls = append(toolCalls, controlplane.InferenceToolCall{
					ToolCallID:       call.ToolCallID,
					Name:             call.ToolName,
					ArgumentsJSON:    call.Input,
					ProviderMetadata: gatewayProviderMetadataJSON(call.ProviderOptions),
				})
			}
		}
		if text.Len() == 0 && len(toolCalls) == 0 {
			continue
		}
		messages = append(messages, controlplane.InferenceMessage{
			Role:      string(msg.Role),
			Content:   text.String(),
			ToolCalls: toolCalls,
		})
	}
	return messages
}

func promptToGatewayToolResults(prompt fantasy.Prompt) []controlplane.InferenceToolResult {
	var results []controlplane.InferenceToolResult
	for _, msg := range prompt {
		for _, part := range msg.Content {
			result, ok := fantasy.AsMessagePart[fantasy.ToolResultPart](part)
			if !ok {
				continue
			}
			results = append(results, controlplane.InferenceToolResult{
				ToolCallID: result.ToolCallID,
				Name:       toolNameForResult(prompt, result.ToolCallID),
				Content:    toolResultText(result.Output),
			})
		}
	}
	return results
}

func toolNameForResult(prompt fantasy.Prompt, toolCallID string) string {
	for _, msg := range prompt {
		for _, part := range msg.Content {
			call, ok := fantasy.AsMessagePart[fantasy.ToolCallPart](part)
			if ok && call.ToolCallID == toolCallID {
				return call.ToolName
			}
		}
	}
	return ""
}

func toolResultText(result fantasy.ToolResultOutputContent) string {
	switch value := result.(type) {
	case fantasy.ToolResultOutputContentText:
		return value.Text
	case *fantasy.ToolResultOutputContentText:
		if value != nil {
			return value.Text
		}
	case fantasy.ToolResultOutputContentError:
		return value.Error.Error()
	case *fantasy.ToolResultOutputContentError:
		if value != nil && value.Error != nil {
			return value.Error.Error()
		}
	case fantasy.ToolResultOutputContentMedia:
		return value.Text
	case *fantasy.ToolResultOutputContentMedia:
		if value != nil {
			return value.Text
		}
	}
	return ""
}

func gatewayProviderMetadataJSON(metadata fantasy.ProviderOptions) json.RawMessage {
	if len(metadata) == 0 {
		return nil
	}
	if googleData, ok := metadata[google.Name]; ok {
		data, err := json.Marshal(googleData)
		if err == nil {
			var meta google.ReasoningMetadata
			if err := json.Unmarshal(data, &meta); err == nil && meta.Signature != "" {
				normalized, err := json.Marshal(map[string]any{
					google.Name: map[string]any{
						"signature":        meta.Signature,
						"thoughtSignature": meta.Signature,
					},
				})
				if err == nil {
					return normalized
				}
			}
		}
	}
	data, err := json.Marshal(metadata)
	if err != nil {
		return nil
	}
	return data
}

func gatewayProviderMetadata(data json.RawMessage) fantasy.ProviderMetadata {
	if len(data) == 0 {
		return nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	result := fantasy.ProviderMetadata{}
	if googleData, ok := raw[google.Name]; ok {
		var payload struct {
			Signature        string `json:"signature"`
			ThoughtSignature string `json:"thoughtSignature"`
			ToolID           string `json:"tool_id"`
		}
		if err := json.Unmarshal(googleData, &payload); err == nil {
			signature := payload.Signature
			if signature == "" {
				signature = payload.ThoughtSignature
			}
			if signature != "" || payload.ToolID != "" {
				result[google.Name] = &google.ReasoningMetadata{
					Signature: signature,
					ToolID:    payload.ToolID,
				}
			}
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func eventToStreamPart(event gatewayEvent, usage fantasy.Usage) (fantasy.StreamPart, bool) {
	switch event.Type {
	case "text.delta":
		return fantasy.StreamPart{
			Type:  fantasy.StreamPartTypeTextDelta,
			Delta: event.Delta,
		}, true
	case "thinking.delta":
		return fantasy.StreamPart{
			Type:  fantasy.StreamPartTypeReasoningDelta,
			Delta: event.Delta,
		}, true
	case "tool.call":
		return fantasy.StreamPart{
			Type:             fantasy.StreamPartTypeToolCall,
			ID:               event.ToolCallID,
			ToolCallName:     event.Name,
			ToolCallInput:    event.ArgumentsJSON,
			ProviderMetadata: gatewayProviderMetadata(event.ProviderMetadata),
		}, true
	case "run.completed":
		return fantasy.StreamPart{
			Type:         fantasy.StreamPartTypeFinish,
			FinishReason: fantasy.FinishReason(event.FinishReason),
			Usage:        usage,
		}, true
	case "run.failed":
		return fantasy.StreamPart{
			Type:  fantasy.StreamPartTypeError,
			Error: fmt.Errorf("%s: %s", event.Code, event.Message),
		}, true
	default:
		return fantasy.StreamPart{}, false
	}
}

var (
	_ fantasy.Provider      = Provider{}
	_ fantasy.LanguageModel = languageModel{}
)
