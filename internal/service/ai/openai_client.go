package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type OpenAIClient struct {
	apiKey string
	model  string
	http   *http.Client
}

func NewOpenAIClient(apiKey, model string) *OpenAIClient {
	return &OpenAIClient{
		apiKey: apiKey,
		model:  model,
		http: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

type chatCompletionsReq struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionsResp struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

func (c *OpenAIClient) Analyze(ctx context.Context, prompt string) (string, error) {
	payload := chatCompletionsReq{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: "You are an AI trading assistant for crypto markets. Be concise, mention uncertainty, avoid guaranteed claims, and include risk-management notes."},
			{Role: "user", Content: prompt},
		},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal openai request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(b))
	if err != nil {
		return "", fmt.Errorf("create openai request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("openai request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("openai status: %d", resp.StatusCode)
	}

	var out chatCompletionsResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("decode openai response: %w", err)
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("empty openai choices")
	}
	return out.Choices[0].Message.Content, nil
}
