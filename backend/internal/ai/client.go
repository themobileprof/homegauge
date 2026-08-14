package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var ErrNoProvider = errors.New("no AI provider configured")

type ProviderName string

const (
	ProviderAnthropic ProviderName = "anthropic"
	ProviderGemini   ProviderName = "gemini"
	ProviderDeepSeek ProviderName = "deepseek"
)

type Message struct {
	Role    string `json:"role"` // user | system
	Content string `json:"content"`
}

type CompletionRequest struct {
	System string
	User   string
	JSON   bool // ask for JSON-only reply when possible
}

type Completion struct {
	Provider ProviderName `json:"provider"`
	Model    string       `json:"model"`
	Text     string       `json:"text"`
}

type Provider interface {
	Name() ProviderName
	Model() string
	Configured() bool
	Complete(ctx context.Context, req CompletionRequest) (*Completion, error)
}

type Client struct {
	providers []Provider
	http      *http.Client
}

func NewClient(anthropicKey, anthropicModel, geminiKey, geminiModel, deepseekKey, deepseekModel string) *Client {
	httpClient := &http.Client{Timeout: 90 * time.Second}
	c := &Client{http: httpClient}
	c.providers = []Provider{
		&anthropicProvider{apiKey: anthropicKey, model: anthropicModel, http: httpClient},
		&geminiProvider{apiKey: geminiKey, model: geminiModel, http: httpClient},
		&deepseekProvider{apiKey: deepseekKey, model: deepseekModel, http: httpClient},
	}
	return c
}

func (c *Client) ConfiguredProviders() []ProviderName {
	var out []ProviderName
	for _, p := range c.providers {
		if p.Configured() {
			out = append(out, p.Name())
		}
	}
	return out
}

func (c *Client) Primary() (Provider, error) {
	for _, p := range c.providers {
		if p.Configured() {
			return p, nil
		}
	}
	return nil, ErrNoProvider
}

func (c *Client) Complete(ctx context.Context, req CompletionRequest) (*Completion, error) {
	p, err := c.Primary()
	if err != nil {
		return nil, err
	}
	return p.Complete(ctx, req)
}

// CompleteAll runs every configured provider (best-effort, in parallel) for multi-model review.
func (c *Client) CompleteAll(ctx context.Context, req CompletionRequest) []Completion {
	type result struct {
		idx  int
		comp *Completion
	}
	configured := make([]Provider, 0, len(c.providers))
	for _, p := range c.providers {
		if p.Configured() {
			configured = append(configured, p)
		}
	}
	if len(configured) == 0 {
		return nil
	}
	ch := make(chan result, len(configured))
	for i, p := range configured {
		i, p := i, p
		go func() {
			res, err := p.Complete(ctx, req)
			if err != nil || res == nil {
				ch <- result{idx: i}
				return
			}
			ch <- result{idx: i, comp: res}
		}()
	}
	tmp := make([]*Completion, len(configured))
	for range configured {
		r := <-ch
		if r.comp != nil {
			tmp[r.idx] = r.comp
		}
	}
	var out []Completion
	for _, c := range tmp {
		if c != nil {
			out = append(out, *c)
		}
	}
	return out
}

func postJSON(ctx context.Context, httpClient *http.Client, url string, headers map[string]string, body any) ([]byte, int, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return nil, res.StatusCode, err
	}
	return raw, res.StatusCode, nil
}

func extractJSONObject(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```JSON")
		s = strings.TrimPrefix(s, "```")
		if i := strings.LastIndex(s, "```"); i >= 0 {
			s = s[:i]
		}
		s = strings.TrimSpace(s)
	}
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}

// --- Anthropic ---

type anthropicProvider struct {
	apiKey string
	model  string
	http   *http.Client
}

func (p *anthropicProvider) Name() ProviderName { return ProviderAnthropic }
func (p *anthropicProvider) Model() string      { return p.model }
func (p *anthropicProvider) Configured() bool   { return strings.TrimSpace(p.apiKey) != "" }

func (p *anthropicProvider) Complete(ctx context.Context, req CompletionRequest) (*Completion, error) {
	system := req.System
	if req.JSON && system != "" {
		system += "\nRespond with a single JSON object only. No markdown."
	}
	body := map[string]any{
		"model":      p.model,
		"max_tokens": 2048,
		"messages": []map[string]string{
			{"role": "user", "content": req.User},
		},
	}
	if system != "" {
		body["system"] = system
	}
	raw, status, err := postJSON(ctx, p.http, "https://api.anthropic.com/v1/messages", map[string]string{
		"x-api-key":         p.apiKey,
		"anthropic-version": "2023-06-01",
	}, body)
	if err != nil {
		return nil, err
	}
	if status >= 300 {
		return nil, fmt.Errorf("anthropic: HTTP %d: %s", status, truncate(string(raw), 300))
	}
	var parsed struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	var text strings.Builder
	for _, c := range parsed.Content {
		if c.Type == "text" {
			text.WriteString(c.Text)
		}
	}
	out := strings.TrimSpace(text.String())
	if out == "" {
		return nil, errors.New("anthropic: empty response")
	}
	return &Completion{Provider: ProviderAnthropic, Model: p.model, Text: out}, nil
}

// --- Gemini ---

type geminiProvider struct {
	apiKey string
	model  string
	http   *http.Client
}

func (p *geminiProvider) Name() ProviderName { return ProviderGemini }
func (p *geminiProvider) Model() string      { return p.model }
func (p *geminiProvider) Configured() bool   { return strings.TrimSpace(p.apiKey) != "" }

func (p *geminiProvider) Complete(ctx context.Context, req CompletionRequest) (*Completion, error) {
	prompt := req.User
	if req.System != "" {
		prompt = req.System + "\n\n" + req.User
	}
	if req.JSON {
		prompt += "\n\nRespond with a single JSON object only. No markdown."
	}
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		p.model, p.apiKey,
	)
	body := map[string]any{
		"contents": []map[string]any{
			{"role": "user", "parts": []map[string]string{{"text": prompt}}},
		},
	}
	raw, status, err := postJSON(ctx, p.http, url, nil, body)
	if err != nil {
		return nil, err
	}
	if status >= 300 {
		return nil, fmt.Errorf("gemini: HTTP %d: %s", status, truncate(string(raw), 300))
	}
	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return nil, errors.New("gemini: empty response")
	}
	text := strings.TrimSpace(parsed.Candidates[0].Content.Parts[0].Text)
	return &Completion{Provider: ProviderGemini, Model: p.model, Text: text}, nil
}

// --- DeepSeek (OpenAI-compatible) ---

type deepseekProvider struct {
	apiKey string
	model  string
	http   *http.Client
}

func (p *deepseekProvider) Name() ProviderName { return ProviderDeepSeek }
func (p *deepseekProvider) Model() string      { return p.model }
func (p *deepseekProvider) Configured() bool   { return strings.TrimSpace(p.apiKey) != "" }

func (p *deepseekProvider) Complete(ctx context.Context, req CompletionRequest) (*Completion, error) {
	msgs := []map[string]string{}
	system := req.System
	if req.JSON {
		if system != "" {
			system += "\nRespond with a single JSON object only. No markdown."
		} else {
			system = "Respond with a single JSON object only. No markdown."
		}
	}
	if system != "" {
		msgs = append(msgs, map[string]string{"role": "system", "content": system})
	}
	msgs = append(msgs, map[string]string{"role": "user", "content": req.User})
	body := map[string]any{
		"model":    p.model,
		"messages": msgs,
	}
	raw, status, err := postJSON(ctx, p.http, "https://api.deepseek.com/chat/completions", map[string]string{
		"Authorization": "Bearer " + p.apiKey,
	}, body)
	if err != nil {
		return nil, err
	}
	if status >= 300 {
		return nil, fmt.Errorf("deepseek: HTTP %d: %s", status, truncate(string(raw), 300))
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Choices) == 0 {
		return nil, errors.New("deepseek: empty response")
	}
	text := strings.TrimSpace(parsed.Choices[0].Message.Content)
	return &Completion{Provider: ProviderDeepSeek, Model: p.model, Text: text}, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
