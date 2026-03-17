# AI 統合仕様

## 使用モデルと用途

| 用途 | モデル | 呼び出し箇所 |
|------|-------|------------|
| テキスト埋め込み生成 | `text-embedding-3-small` | 記事処理ワーカー（ステージ2） |
| 記事要約生成 | `gpt-4o-mini` | 記事処理ワーカー（ステージ3） |
| タグ抽出 | `gpt-4o-mini` | 記事処理ワーカー（ステージ4） |

---

## OpenAI API クライアント設計

### クライアント構造

```go
type AIClient struct {
    client      *openai.Client
    rateLimiter *rate.Limiter  // トークンバケット
    metrics     MetricsRecorder
}

// 埋め込み生成
func (c *AIClient) CreateEmbedding(ctx context.Context, text string) ([]float32, error)

// テキスト補完（要約・タグ抽出）
func (c *AIClient) CreateCompletion(ctx context.Context, req CompletionRequest) (string, error)
```

### レートリミット設定

```
text-embedding-3-small: 3,000 RPM / 1,000,000 TPM
gpt-4o-mini: 500 RPM / 200,000 TPM

実装上の制限（余裕を持たせる）:
- 埋め込み: 50 RPM（ワーカー起動時に設定）
- 補完: 20 RPM
```

### リトライ戦略

```go
func retryWithBackoff(ctx context.Context, fn func() error) error {
    delays := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}

    for i, delay := range delays {
        err := fn()
        if err == nil {
            return nil
        }

        apiErr, ok := err.(*openai.APIError)
        if !ok {
            return err  // 非API エラーはリトライしない
        }

        switch apiErr.HTTPStatusCode {
        case 429:  // Rate Limit
            waitTime := extractRetryAfter(apiErr)
            if waitTime == 0 {
                waitTime = 60 * time.Second
            }
            time.Sleep(waitTime)
        case 500, 502, 503, 504:  // Server Error
            if i < len(delays)-1 {
                time.Sleep(delay)
            }
        default:
            return err  // 4xx はリトライしない
        }
    }

    return fmt.Errorf("max retries exceeded")
}
```

---

## プロンプト管理

### ディレクトリ構造

```
internal/ai/prompts/
├── summarize_v1.txt
├── summarize_v2.txt  （将来バージョン）
├── tag_extract_v1.txt
└── tag_extract_v2.txt
```

### バージョン管理ルール

- プロンプトファイルはバージョン番号をサフィックスに持つ。
- 現在使用中のバージョンは設定ファイル（環境変数）で管理する。
- 新しいプロンプトバージョンをリリースする前に、テスト用データセット（100 記事）で
  品質を手動確認する。
- `article_summaries.prompt_version` に使用バージョンを記録する。

---

## 要約プロンプト（v1.0）

```
System:
あなたは技術記事の要約専門家です。
以下のルールに厳格に従って要約を作成してください:

ルール:
1. 2〜4 文で収める（必ず守る）
2. 記事の主要な技術的ポイントを含める
3. 「なぜ重要か」または「どんな影響があるか」を最後の文で説明する
4. 箇条書きは絶対に使わない
5. 記事の言語（日本語または英語）に合わせて要約を書く
6. 主観的評価（「素晴らしい」「革命的な」等）は使わない
7. 要約のみを出力する（余計なテキスト不可）

User:
タイトル: {title}

記事本文（一部）:
{clean_content}
```

---

## タグ抽出プロンプト（v1.0）

```
System:
あなたは技術記事の分類専門家です。
以下の許可されたタグリストから最も適切なタグを選択してください。

許可されたタグ（カンマ区切り）:
{tags_list}

ルール:
1. 3〜7 個のタグを選ぶ（多すぎず、少なすぎず）
2. confidence は 0.0〜1.0 の浮動小数点数で示す
3. 記事の中心的なトピックを優先する
4. 必ず以下の JSON のみを出力する（余計なテキスト不可）

出力形式:
{"tags": [{"name": "タグ名", "confidence": 0.95}, ...]}

User:
タイトル: {title}

記事本文（一部）:
{clean_content}
```

---

## AI 出力の検証

### 要約の品質チェック

```go
func validateSummary(text string) error {
    if len(text) < 50 {
        return fmt.Errorf("summary too short: %d chars", len(text))
    }
    if len(text) > 1000 {
        return fmt.Errorf("summary too long: %d chars", len(text))
    }
    // 明らかな JSON/コードブロックの混入チェック
    if strings.Contains(text, "```") || strings.Contains(text, "{\"") {
        return fmt.Errorf("summary contains code or JSON")
    }
    return nil
}
```

### タグ抽出の検証

```go
func validateTagResponse(resp string, allowedTags map[string]bool) ([]ExtractedTag, error) {
    var result TagExtractionResult
    if err := json.Unmarshal([]byte(resp), &result); err != nil {
        return nil, fmt.Errorf("JSON parse error: %w", err)
    }

    var valid []ExtractedTag
    for _, tag := range result.Tags {
        if !allowedTags[tag.Name] {
            continue  // 許可リスト外のタグは無視
        }
        if tag.Confidence < 0.5 {
            continue  // 信頼度が低いタグは無視
        }
        valid = append(valid, tag)
    }

    if len(valid) == 0 {
        return nil, fmt.Errorf("no valid tags extracted")
    }

    return valid, nil
}
```

---

## コスト管理

### トークン使用量の見積もり（1 記事あたり）

| 処理 | 入力トークン | 出力トークン | コスト概算 |
|------|-----------|-----------|---------|
| 埋め込み生成 | ~2,000 | - | $0.00004 |
| 要約生成 | ~1,200 | ~150 | $0.000165 |
| タグ抽出 | ~600 | ~80 | $0.000082 |
| **合計** | | | **~$0.00029** |

1 日 1,000 記事処理の場合: 約 **$0.29/日**

### アラート設定

- 1 日のトークン使用量が予算の 80% を超えたら CloudWatch アラートを発行。
- 異常なトークン使用量（単一記事で 5,000 トークン超）はログに警告を記録。
