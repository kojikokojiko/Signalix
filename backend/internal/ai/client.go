package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
)

const (
	maxEmbedChars   = 8000
	maxSummaryChars = 4000
	maxTagChars     = 2000
	maxChatChars    = 6000
)

// ChatMessage represents a single turn in a conversation.
type ChatMessage struct {
	Role    string `json:"role"`    // "user" or "assistant"
	Content string `json:"content"`
}

// ExtractedTag holds a tag name and confidence score returned by the AI.
type ExtractedTag struct {
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
}

// Client wraps the OpenAI SDK with retry logic.
type Client struct {
	oai openai.Client
}

// NewClient creates an AIClient using the provided API key.
func NewClient(apiKey string) *Client {
	return &Client{
		oai: openai.NewClient(option.WithAPIKey(apiKey)),
	}
}

// CreateEmbedding generates a 1536-dim embedding for the given text.
func (c *Client) CreateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if len(text) > maxEmbedChars {
		text = text[:maxEmbedChars]
	}

	var result []float32
	err := retryWithBackoff(ctx, func() error {
		resp, err := c.oai.Embeddings.New(ctx, openai.EmbeddingNewParams{
			Input: openai.EmbeddingNewParamsInputUnion{
				OfString: openai.String(text),
			},
			Model: openai.EmbeddingModelTextEmbedding3Small,
		})
		if err != nil {
			return classifyError(err)
		}
		if len(resp.Data) == 0 {
			return permanent(fmt.Errorf("empty embedding response"))
		}
		raw := resp.Data[0].Embedding
		result = make([]float32, len(raw))
		for i, v := range raw {
			result[i] = float32(v)
		}
		return nil
	})
	return result, err
}

// CreateSummary generates a 2-4 sentence summary of the article.
func (c *Client) CreateSummary(ctx context.Context, title, cleanContent string) (string, int, error) {
	content := cleanContent
	if len(content) > maxSummaryChars {
		content = content[:maxSummaryChars]
	}

	var summary string
	var tokensUsed int
	err := retryWithBackoff(ctx, func() error {
		resp, err := c.oai.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model: openai.ChatModelGPT4oMini,
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage(summarySystemPrompt),
				openai.UserMessage(fmt.Sprintf("タイトル: %s\n\n記事本文（一部）:\n%s", title, content)),
			},
			Temperature: openai.Float(0.3),
			MaxTokens:   openai.Int(300),
		})
		if err != nil {
			return classifyError(err)
		}
		if len(resp.Choices) == 0 {
			return retryable(fmt.Errorf("empty completion response"))
		}
		summary = strings.TrimSpace(resp.Choices[0].Message.Content)
		tokensUsed = int(resp.Usage.TotalTokens)
		return nil
	})
	return summary, tokensUsed, err
}

// CreateTags extracts relevant tags from the article.
func (c *Client) CreateTags(ctx context.Context, title, cleanContent string, allowedTags []string) ([]ExtractedTag, int, error) {
	content := cleanContent
	if len(content) > maxTagChars {
		content = content[:maxTagChars]
	}
	tagsCSV := strings.Join(allowedTags, ", ")
	systemPrompt := fmt.Sprintf(tagSystemPromptTpl, tagsCSV)

	var tags []ExtractedTag
	var tokensUsed int
	err := retryWithBackoff(ctx, func() error {
		resp, err := c.oai.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model: openai.ChatModelGPT4oMini,
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage(systemPrompt),
				openai.UserMessage(fmt.Sprintf("タイトル: %s\n\n記事本文（一部）:\n%s", title, content)),
			},
			Temperature: openai.Float(0.1),
			MaxTokens:   openai.Int(200),
			ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
				OfJSONObject: &shared.ResponseFormatJSONObjectParam{},
			},
		})
		if err != nil {
			return classifyError(err)
		}
		if len(resp.Choices) == 0 {
			return retryable(fmt.Errorf("empty completion response"))
		}
		raw := resp.Choices[0].Message.Content
		tokensUsed = int(resp.Usage.TotalTokens)
		parsed, err := parseTagResponse(raw, allowedTags)
		if err != nil {
			return retryable(err)
		}
		tags = parsed
		return nil
	})
	return tags, tokensUsed, err
}

// CreateChat generates a reply to a user's question about a specific article.
func (c *Client) CreateChat(ctx context.Context, articleTitle, articleContent string, history []ChatMessage, userMessage string) (string, error) {
	content := articleContent
	if len(content) > maxChatChars {
		content = content[:maxChatChars]
	}

	systemPrompt := fmt.Sprintf(chatSystemPromptTpl, articleTitle, content)

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPrompt),
	}
	for _, m := range history {
		switch m.Role {
		case "user":
			messages = append(messages, openai.UserMessage(m.Content))
		case "assistant":
			messages = append(messages, openai.AssistantMessage(m.Content))
		}
	}
	messages = append(messages, openai.UserMessage(userMessage))

	var reply string
	err := retryWithBackoff(ctx, func() error {
		resp, err := c.oai.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model:       openai.ChatModelGPT4oMini,
			Messages:    messages,
			Temperature: openai.Float(0.7),
			MaxTokens:   openai.Int(800),
		})
		if err != nil {
			return classifyError(err)
		}
		if len(resp.Choices) == 0 {
			return retryable(fmt.Errorf("empty completion response"))
		}
		reply = strings.TrimSpace(resp.Choices[0].Message.Content)
		return nil
	})
	return reply, err
}

// --- retry ---

type permanentError struct{ err error }
type retryableError struct{ err error }

func permanent(err error) error { return &permanentError{err} }
func retryable(err error) error { return &retryableError{err} }

func (e *permanentError) Error() string { return e.err.Error() }
func (e *retryableError) Error() string { return e.err.Error() }

func retryWithBackoff(ctx context.Context, fn func() error) error {
	delays := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}
	var lastErr error
	for i, delay := range delays {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		var perm *permanentError
		if errors.As(lastErr, &perm) {
			return perm.err
		}
		if i < len(delays)-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}
	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// classifyError wraps an OpenAI API error for retry decisions.
func classifyError(err error) error {
	var apiErr *openai.Error
	if !errors.As(err, &apiErr) {
		return permanent(err)
	}
	switch apiErr.StatusCode {
	case 429, 500, 502, 503, 504:
		return retryable(err)
	default:
		return permanent(err)
	}
}

// parseTagResponse validates and filters the raw JSON tag response.
func parseTagResponse(raw string, allowedTags []string) ([]ExtractedTag, error) {
	allowed := make(map[string]bool, len(allowedTags))
	for _, t := range allowedTags {
		allowed[t] = true
	}

	var result struct {
		Tags []ExtractedTag `json:"tags"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w", err)
	}

	var valid []ExtractedTag
	for _, tag := range result.Tags {
		if !allowed[tag.Name] || tag.Confidence < 0.5 {
			continue
		}
		valid = append(valid, tag)
	}
	if len(valid) == 0 {
		return nil, fmt.Errorf("no valid tags extracted")
	}
	return valid, nil
}

// --- prompts ---

const chatSystemPromptTpl = `あなたは技術記事専門のAIアシスタントです。以下の記事の内容をもとに、ユーザーの質問に日本語で丁寧に回答してください。

ルール:
- 記事の内容に基づいて回答する
- 記事に明記されていない情報は「記事には記載がありませんが、一般的には〜」と前置きして補足する
- 簡潔かつわかりやすく回答する
- コードの例が必要な場合はコードブロックを使う

---
記事タイトル: %s

記事本文:
%s
---`

const summarySystemPrompt = `あなたは技術記事の要約専門家です。
以下のルールに厳格に従って要約を作成してください:

ルール:
1. 2〜4 文で収める（必ず守る）
2. 記事の主要な技術的ポイントを含める
3. 「なぜ重要か」または「どんな影響があるか」を最後の文で説明する
4. 箇条書きは絶対に使わない
5. 記事の言語（日本語または英語）に合わせて要約を書く
6. 主観的評価（「素晴らしい」「革命的な」等）は使わない
7. 要約のみを出力する（余計なテキスト不可）`

const tagSystemPromptTpl = `あなたは技術記事の分類専門家です。
以下の許可されたタグリストから最も適切なタグを選択してください。

許可されたタグ（カンマ区切り）:
%s

ルール:
1. 3〜7 個のタグを選ぶ（多すぎず、少なすぎず）
2. confidence は 0.0〜1.0 の浮動小数点数で示す
3. 記事の中心的なトピックを優先する
4. 必ず以下の JSON のみを出力する（余計なテキスト不可）

出力形式:
{"tags": [{"name": "タグ名", "confidence": 0.95}, ...]}`
