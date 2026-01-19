// Package langmesh provides a drop-in replacement for OpenAI Go client
//
// Usage:
//
//	// Before
//	import "github.com/sashabaranov/go-openai"
//
//	// After
//	import openai "github.com/langmesh-ai/openai-go"
//
//	client := openai.NewClient(apiKey)
//	// Works exactly the same!
package langmesh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	openai "github.com/sashabaranov/go-openai"
)

var (
	langmeshAPIKey          = os.Getenv("langmesh_API_KEY")
	langmeshTelemetryURL    = getEnv("langmesh_TELEMETRY_ENDPOINT", "https://api.langmesh.ai/v1/telemetry")
	langmeshProxyEnabled    = os.Getenv("langmesh_PROXY_ENABLED") == "true"
	langmeshBaseURL         = getEnv("langmesh_BASE_URL", "https://api.langmesh.ai/v1/openai")
)

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Client is a langmesh-wrapped OpenAI client
type Client struct {
	*openai.Client
	telemetryEnabled bool
	telemetryBuffer  []TelemetryEvent
	mu               sync.Mutex
	httpClient       *http.Client
}

// NewClient creates a new langmesh-wrapped OpenAI client
func NewClient(authToken string) *Client {
	config := openai.DefaultConfig(authToken)

	// If proxy is enabled, route through langmesh
	if langmeshProxyEnabled && langmeshAPIKey != "" {
		config.BaseURL = langmeshBaseURL
		config.HTTPClient = &http.Client{
			Transport: &langmeshTransport{
				base:      http.DefaultTransport,
				langmeshKey:   langmeshAPIKey,
				originalKey: authToken,
			},
		}
	}

	client := &Client{
		Client:           openai.NewClientWithConfig(config),
		telemetryEnabled: langmeshAPIKey != "",
		telemetryBuffer:  make([]TelemetryEvent, 0, 10),
		httpClient:       &http.Client{Timeout: 5 * time.Second},
	}

	if client.telemetryEnabled {
		client.startTelemetry()
	}

	return client
}

// CreateChatCompletion wraps the original method with telemetry
func (c *Client) CreateChatCompletion(
	ctx context.Context,
	request openai.ChatCompletionRequest,
) (openai.ChatCompletionResponse, error) {
	startTime := time.Now()
	requestID := fmt.Sprintf("req_%d_%s", time.Now().UnixMilli(), uuid.New().String()[:8])

	resp, err := c.Client.CreateChatCompletion(ctx, request)
	endTime := time.Now()

	if c.telemetryEnabled {
		event := TelemetryEvent{
			RequestID:      requestID,
			TimestampStart: startTime.Format(time.RFC3339),
			TimestampEnd:   endTime.Format(time.RFC3339),
			Model:          request.Model,
			Endpoint:       "chat.completions",
			LatencyMs:      endTime.Sub(startTime).Milliseconds(),
			Status:         "success",
		}

		if err != nil {
			event.Status = "error"
			event.ErrorClass = "Error"
			event.ErrorMessage = err.Error()
		} else {
			event.TokenUsage = TokenUsage{
				PromptTokens:     resp.Usage.PromptTokens,
				CompletionTokens: resp.Usage.CompletionTokens,
				TotalTokens:      resp.Usage.TotalTokens,
			}
			event.CostEstimateUSD = estimateCost(request.Model, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
		}

		c.recordTelemetry(event)
	}

	return resp, err
}

func (c *Client) recordTelemetry(event TelemetryEvent) {
	c.mu.Lock()
	c.telemetryBuffer = append(c.telemetryBuffer, event)
	shouldFlush := len(c.telemetryBuffer) >= 10
	c.mu.Unlock()

	if shouldFlush {
		c.flushTelemetry()
	}
}

func (c *Client) flushTelemetry() {
	c.mu.Lock()
	if len(c.telemetryBuffer) == 0 {
		c.mu.Unlock()
		return
	}
	batch := make([]TelemetryEvent, len(c.telemetryBuffer))
	copy(batch, c.telemetryBuffer)
	c.telemetryBuffer = c.telemetryBuffer[:0]
	c.mu.Unlock()

	go func() {
		payload := map[string]interface{}{"events": batch}
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return
		}

		req, err := http.NewRequest("POST", langmeshTelemetryURL, bytes.NewBuffer(jsonData))
		if err != nil {
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+langmeshAPIKey)

		_, _ = c.httpClient.Do(req)
		// Silent drop - telemetry must never break user's app
	}()
}

func (c *Client) startTelemetry() {
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for range ticker.C {
			c.flushTelemetry()
		}
	}()
}

func estimateCost(model string, promptTokens, completionTokens int) float64 {
	pricing := map[string]map[string]float64{
		"gpt-4o":        {"input": 2.5, "output": 10.0},
		"gpt-4o-mini":   {"input": 0.15, "output": 0.6},
		"gpt-4-turbo":   {"input": 10.0, "output": 30.0},
		"gpt-4":         {"input": 30.0, "output": 60.0},
		"gpt-3.5-turbo": {"input": 0.5, "output": 1.5},
	}

	modelPricing, ok := pricing[model]
	if !ok {
		modelPricing = map[string]float64{"input": 0.01, "output": 0.01}
	}

	return (float64(promptTokens)/1_000_000)*modelPricing["input"] +
		(float64(completionTokens)/1_000_000)*modelPricing["output"]
}

// langmeshTransport adds langmesh headers to requests
type langmeshTransport struct {
	base        http.RoundTripper
	langmeshKey     string
	originalKey string
}

func (t *langmeshTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("X-langmesh-API-Key", t.langmeshKey)
	req.Header.Set("X-langmesh-Original-API-Key", t.originalKey)
	return t.base.RoundTrip(req)
}

// TelemetryEvent represents a telemetry event
type TelemetryEvent struct {
	RequestID       string      `json:"request_id"`
	TimestampStart  string      `json:"timestamp_start"`
	TimestampEnd    string      `json:"timestamp_end"`
	Model           string      `json:"model"`
	Endpoint        string      `json:"endpoint"`
	LatencyMs       int64       `json:"latency_ms"`
	TokenUsage      TokenUsage  `json:"token_usage"`
	CostEstimateUSD float64     `json:"cost_estimate_usd"`
	Status          string      `json:"status"`
	ErrorClass      string      `json:"error_class,omitempty"`
	ErrorMessage    string      `json:"error_message,omitempty"`
}

// TokenUsage represents token usage
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
