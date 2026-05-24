package gateway

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"charm.land/fantasy"
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
	Type          string `json:"type"`
	Delta         string `json:"delta"`
	ToolCallID    string `json:"toolCallId"`
	Name          string `json:"name"`
	ArgumentsJSON string `json:"argumentsJson"`
	Code          string `json:"code"`
	Message       string `json:"message"`
	Retryable     bool   `json:"retryable"`
	InputTokens   int64  `json:"inputTokens"`
	OutputTokens  int64  `json:"outputTokens"`
	FinishReason  string `json:"finishReason"`
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
		return nil, fmt.Errorf("F4RGE managed runtime session required; run `4rged login`")
	}
	request := controlplane.InferenceRequest{
		RequestID:      uuid.NewString(),
		Surface:        "cli",
		OrganizationID: session.OrganizationID,
		SessionID:      session.RuntimeSessionID,
		ModelID:        m.modelID,
		PromptMode:     "agent",
		Messages:       promptToGatewayMessages(call.Prompt),
	}
	body, err := controlplane.New().StreamInference(ctx, session, request)
	if err != nil {
		return nil, err
	}
	return func(yield func(fantasy.StreamPart) bool) {
		defer body.Close()
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
		}
		if err := scanner.Err(); err != nil && err != io.EOF {
			yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeError, Error: err})
		}
	}, nil
}

func (m languageModel) GenerateObject(context.Context, fantasy.ObjectCall) (*fantasy.ObjectResponse, error) {
	return nil, fmt.Errorf("F4RGE Gateway object generation is not implemented")
}

func (m languageModel) StreamObject(context.Context, fantasy.ObjectCall) (fantasy.ObjectStreamResponse, error) {
	return nil, fmt.Errorf("F4RGE Gateway object streaming is not implemented")
}

func promptToGatewayMessages(prompt fantasy.Prompt) []controlplane.InferenceMessage {
	messages := make([]controlplane.InferenceMessage, 0, len(prompt))
	for _, msg := range prompt {
		var text strings.Builder
		for _, part := range msg.Content {
			if content, ok := fantasy.AsMessagePart[fantasy.TextPart](part); ok {
				text.WriteString(content.Text)
			}
		}
		if text.Len() == 0 {
			continue
		}
		messages = append(messages, controlplane.InferenceMessage{
			Role:    string(msg.Role),
			Content: text.String(),
		})
	}
	return messages
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
			Type:          fantasy.StreamPartTypeToolCall,
			ID:            event.ToolCallID,
			ToolCallName:  event.Name,
			ToolCallInput: event.ArgumentsJSON,
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

var _ fantasy.Provider = Provider{}
var _ fantasy.LanguageModel = languageModel{}
