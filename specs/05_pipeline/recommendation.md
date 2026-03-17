# パイプライン仕様: レコメンドエンジン

## 概要

レコメンドエンジンは、各ユーザーの興味プロフィールと記事の特徴を組み合わせて
パーソナライズスコアを算出し、`recommendation_logs` に書き込む。

---

## トリガー

| トリガー | 詳細 |
|---------|------|
| スケジュール実行 | 毎時 0 分に全アクティブユーザーを対象に実行 |
| 新規記事バッチ完了 | `stream:recommendation_refresh` を消費 |
| ユーザーフィードバック | フィードバック API が非同期でトリガー（対象ユーザーのみ） |
| 手動リフレッシュ | `POST /api/v1/recommendations/refresh` API（レートリミット付き） |

---

## スコアリングアルゴリズム（Phase 1: コンテンツベース）

### 最終スコア計算式

```
total_score = (
    w_relevance    * relevance_score      +
    w_freshness    * freshness_score      +
    w_trend        * trend_score          +
    w_source       * source_quality_score +
    w_personal     * personalization_boost
)

重み係数（合計 1.0）:
w_relevance = 0.35
w_freshness = 0.20
w_trend     = 0.20
w_source    = 0.10
w_personal  = 0.15
```

---

### relevance_score（関連性スコア）

ユーザーの興味プロフィールベクトルと記事埋め込みベクトルのコサイン類似度。

```go
// ユーザー興味プロフィールのベクトルを生成
// user_interests の各タグに紐づく記事埋め込みの加重平均
func buildUserProfileVector(userID uuid.UUID) ([]float32, error) {
    interests := getTopUserInterests(userID, limit=20)

    var profileVector []float32
    totalWeight := 0.0

    for _, interest := range interests {
        // そのタグの直近50記事の埋め込みの平均ベクトルを取得
        tagVector := getAverageEmbeddingForTag(interest.TagID, limit=50)
        profileVector = addWeighted(profileVector, tagVector, interest.Weight)
        totalWeight += interest.Weight
    }

    return normalize(profileVector, totalWeight), nil
}

// コサイン類似度
relevance_score = cosineSimilarity(userProfileVector, articleEmbedding)
// 範囲: -1.0 〜 1.0 → 0.0 〜 1.0 に正規化
relevance_score = (relevance_score + 1.0) / 2.0
```

---

### freshness_score（新鮮さスコア）

```go
func freshnessScore(publishedAt time.Time) float64 {
    age := time.Since(publishedAt)
    switch {
    case age < 6*time.Hour:
        return 1.0
    case age < 12*time.Hour:
        return 0.85
    case age < 24*time.Hour:
        return 0.70
    case age < 3*24*time.Hour:
        return 0.50
    case age < 7*24*time.Hour:
        return 0.30
    case age < 30*24*time.Hour:
        return 0.10
    default:
        return 0.05
    }
}
```

---

### trend_score（トレンドスコア）

記事処理時に計算済みの `articles.trend_score` をそのまま使用。
値は 0.0〜1.0 の範囲。

---

### source_quality_score（ソース品質スコア）

`sources.quality_score` をそのまま使用。
値は 0.0〜1.0 の範囲（デフォルト 0.7）。
管理者が手動で調整可能。

---

### personalization_boost（個人化ブースト）

ユーザーが過去に positive フィードバック（like/click/save）を与えた記事のタグと、
当該記事のタグの一致度に基づくブースト値。

```go
func personalizationBoost(userID uuid.UUID, articleTags []Tag) float64 {
    // 過去 30 日間の positive フィードバック記事のタグを集計
    positiveTags := getPositiveFeedbackTags(userID, days=30)

    boost := 0.0
    matchCount := 0
    for _, tag := range articleTags {
        if freq, ok := positiveTags[tag.ID]; ok {
            boost += freq * tag.Confidence
            matchCount++
        }
    }

    if matchCount == 0 {
        return 0.0
    }

    // 0.0〜1.0 に正規化
    return math.Min(boost/float64(matchCount), 1.0)
}
```

---

## 候補記事の取得

```sql
-- 直近 7 日以内の処理済み記事を候補として取得
-- 既にフィードバック済みの記事は除外
-- hide されている記事は除外
SELECT
    a.id,
    a.trend_score,
    a.published_at,
    ae.embedding,
    s.quality_score
FROM articles a
JOIN article_embeddings ae ON ae.article_id = a.id
JOIN sources s ON s.id = a.source_id
WHERE a.status = 'processed'
  AND a.published_at > NOW() - INTERVAL '7 days'
  AND a.language = $user_preferred_language  -- 言語フィルタ（設定ありの場合）
  AND a.id NOT IN (
      SELECT article_id FROM user_feedback
      WHERE user_id = $user_id AND feedback_type IN ('hide', 'dislike')
  )
LIMIT 1000;  -- 候補数の上限
```

---

## レコメンド結果の保存

```sql
-- スコア計算結果を UPSERT
INSERT INTO recommendation_logs (
    user_id, article_id,
    total_score, relevance_score, freshness_score,
    trend_score, source_quality_score, personalization_boost,
    explanation, generated_at, expires_at
)
VALUES (...)
ON CONFLICT (user_id, article_id)
DO UPDATE SET
    total_score             = EXCLUDED.total_score,
    relevance_score         = EXCLUDED.relevance_score,
    freshness_score         = EXCLUDED.freshness_score,
    trend_score             = EXCLUDED.trend_score,
    source_quality_score    = EXCLUDED.source_quality_score,
    personalization_boost   = EXCLUDED.personalization_boost,
    explanation             = EXCLUDED.explanation,
    generated_at            = EXCLUDED.generated_at,
    expires_at              = EXCLUDED.expires_at;
```

---

## 推薦理由テキストの生成

テンプレートベース（LLM 呼び出しなし）で生成する:

```go
func generateExplanation(scores ScoreBreakdown, tags []Tag, source Source) string {
    dominant := dominantScoreComponent(scores)

    switch dominant {
    case "relevance":
        topTag := topMatchingUserInterestTag(tags)
        return fmt.Sprintf("あなたがよく読む %s の記事に類似しています", topTag.Name)

    case "trend":
        if scores.FreshnessScore > 0.8 {
            return fmt.Sprintf("%s のトレンドに急上昇しています", formatPeriod(scores))
        }
        return "今週のトレンド上位の記事です"

    case "personalization":
        topTag := topPositiveFeedbackTag(tags)
        return fmt.Sprintf("「%s」に興味があるあなたへのおすすめです", topTag.Name)

    case "freshness":
        return fmt.Sprintf("%s の最新記事です", source.Name)

    default:
        return "あなたの興味とトレンドの両方に一致しています"
    }
}
```

---

## パフォーマンス考慮事項

### バッチ処理

- 全ユーザーを一度に処理するのではなく、100 ユーザーずつのバッチで処理する。
- バッチ間でレートリミット調整（ゴルーチン数: 10）。

### ベクトル検索の最適化

```sql
-- pgvector でユーザープロフィールベクトルに最近傍の記事を直接検索
-- 候補取得クエリと組み合わせることで、全件スキャンを回避
SELECT article_id, embedding <=> $user_vector AS distance
FROM article_embeddings
ORDER BY embedding <=> $user_vector
LIMIT 100;
```

### キャッシュ無効化

レコメンドログ更新完了後:
```
REDIS DEL user_feed:{user_id}:page:*
```

---

## Phase 2 以降の拡張計画

### 行動シグナルの追加（Phase 2）

- クリックスルー率（CTR）を計算し、過去レコメンドのスコア補正に使用。
- 滞在時間（記事を開いた後どのくらい時間をかけたか）をクライアントから収集。
- `user_interests.weight` の減衰関数を導入（古い行動より新しい行動を重視）。

### 協調フィルタリング（Phase 3）

- 類似ユーザープロフィールを検出（ユーザー埋め込みの類似検索）。
- 類似ユーザーが評価した記事をレコメンドに含める。
- 探索と活用のバランス（ε-greedy または UCB）を導入。
