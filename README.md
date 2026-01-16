# SKEW OpenAI (Go)

Drop-in replacement for the OpenAI Go client with automatic cost optimization and telemetry.

## Installation

```bash
go get github.com/skew-ai/openai-go
```

## Usage

Change one line of code:

```go
// Before
import "github.com/sashabaranov/go-openai"

// After
import openai "github.com/skew-ai/openai-go"

client := openai.NewClient(apiKey)

// Everything works exactly the same!
resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
    Model: "gpt-4o",
    Messages: []openai.ChatCompletionMessage{
        {Role: "user", Content: "Hello!"},
    },
})
```

That's it. No configuration needed.

## What It Does

### Telemetry (Always On)
- Tracks token usage, cost, and latency
- Privacy-preserving (no prompts sent by default)
- Zero performance impact (async)
- Never breaks your app (fail-safe)

### Cost Optimization (Opt-In)
When you enable policies in the SKEW dashboard:
- Automatic model downgrading for simple queries
- Retry storm suppression
- Semantic caching
- Token optimization

## Configuration

### Required
```bash
export SKEW_API_KEY=sk_live_...  # Get from dashboard.skew.ai
```

### Optional
```bash
export SKEW_PROXY_ENABLED=true  # Enable when policies require routing
export SKEW_BASE_URL=https://api.skew.ai/v1/openai  # Custom proxy URL
```

## Migration Path

1. **Install** - `go get github.com/skew-ai/openai-go`
2. **Replace import** - Change package name
3. **Set API key** - `export SKEW_API_KEY=sk_live_...`
4. **See savings** - Visit dashboard.skew.ai
5. **Enable policies** - When ready, `export SKEW_PROXY_ENABLED=true`

## Guarantees

✅ Drop-in replacement - works identically  
✅ No behavior changes without opt-in  
✅ Fail-safe - errors don't break your app  
✅ Reversible - remove anytime  
✅ Privacy-first - no prompts sent by default  

## License

MIT
