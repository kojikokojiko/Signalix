# API 仕様: レコメンデーション (Recommendations)

## エンドポイント一覧

| メソッド | パス | 認証必須 | 説明 |
|---------|------|---------|------|
| GET | `/api/v1/recommendations` | 必要 | パーソナライズフィード取得 |
| POST | `/api/v1/recommendations/refresh` | 必要 | フィード再計算をリクエスト |

---

## レコメンドアイテムオブジェクト

```json
{
  "article": {
    "id": "article-uuid-001",
    "title": "Go 1.23 のジェネリクス改善について",
    "url": "https://example.com/go-1.23-generics",
    "source": {
      "id": "source-uuid-001",
      "name": "Go Blog",
      "site_url": "https://blog.golang.org"
    },
    "author": "Go Team",
    "language": "en",
    "published_at": "2024-03-15T08:00:00Z",
    "summary": "Go 1.23 ではジェネリクスの型推論が大幅に改善され...",
    "tags": [
      { "id": "tag-uuid-001", "name": "go", "category": "language" }
    ],
    "trend_score": 0.87,
    "is_bookmarked": false,
    "user_feedback": null
  },
  "recommendation": {
    "total_score": 0.84,
    "explanation": "あなたがよく読む Go バックエンド記事に類似しています",
    "score_breakdown": {
      "relevance": 0.78,
      "freshness": 0.90,
      "trend": 0.87,
      "source_quality": 0.80,
      "personalization": 0.75
    },
    "generated_at": "2024-03-15T09:30:00Z"
  }
}
```

### explanation の生成ルール

スコアコンポーネントの最大値に基づいて説明テキストを選択する:

| 主因スコアコンポーネント | 説明テキストパターン |
|----------------------|---------------------|
| relevance が最大（≥ 0.7） | "あなたが興味を持つ {タグ名} の記事に類似しています" |
| trend が最大（≥ 0.8） | "{期間} のトレンドに上がっています" |
| freshness が最大（公開 3 時間以内） | "{ソース名} の最新記事です" |
| personalization が最大 | "よく読む {タグ名} 記事に類似しています" |
| 複合（差異が小さい） | "あなたの興味とトレンドの両方に一致しています" |

---

## GET /api/v1/recommendations

ログイン中ユーザーのパーソナライズフィードを取得する。

### クエリパラメータ

| パラメータ | 型 | デフォルト | 説明 |
|-----------|-----|---------|------|
| `page` | integer | 1 | ページ番号 |
| `per_page` | integer | 20 | 1ページあたりの件数（最大 50） |
| `language` | string | - | 言語フィルタ |

### レスポンス: 200 OK

```json
{
  "data": [
    { /* レコメンドアイテムオブジェクト */ },
    { /* レコメンドアイテムオブジェクト */ }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 85,
    "total_pages": 5,
    "has_next": true,
    "has_prev": false
  },
  "meta": {
    "last_refreshed_at": "2024-03-15T09:30:00Z",
    "has_interest_profile": true
  }
}
```

### `meta` フィールドの説明

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `last_refreshed_at` | string (ISO 8601) | フィードが最後に再計算された日時 |
| `has_interest_profile` | boolean | ユーザーが興味設定済みかどうか |

### 興味プロフィール未設定時の挙動

`has_interest_profile: false` の場合、パーソナライズなしのトレンドフィードを返す。
`explanation` は `"トレンド上位の記事です"` となる。

### キャッシュ戦略

- Redis キー: `user_feed:{user_id}:page:{page}`
- TTL: **5 分**
- キャッシュ無効化トリガー: レコメンドワーカー完了時・フィードバック送信時

---

## POST /api/v1/recommendations/refresh

ユーザーのフィード再計算を非同期でリクエストする。

### リクエスト

ボディ不要。

### レスポンス: 202 Accepted

```json
{
  "data": {
    "message": "フィードの再計算をリクエストしました",
    "estimated_wait_seconds": 30
  }
}
```

**レートリミット:** 1 ユーザーあたり **5 分に 1 回** まで。

### エラー

| 条件 | ステータス | コード |
|------|---------|------|
| 再計算リクエストが多すぎる | 429 | `rate_limit_exceeded` |
