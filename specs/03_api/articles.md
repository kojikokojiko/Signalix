# API 仕様: 記事 (Articles)

## エンドポイント一覧

| メソッド | パス | 認証必須 | 説明 |
|---------|------|---------|------|
| GET | `/api/v1/articles` | 任意 | 記事一覧取得（検索・フィルタ対応） |
| GET | `/api/v1/articles/:id` | 任意 | 記事詳細取得 |
| GET | `/api/v1/articles/trending` | 不要 | トレンド記事一覧取得 |

---

## 記事オブジェクト

### ArticleSummary（一覧表示用）

```json
{
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
    { "id": "tag-uuid-001", "name": "go", "category": "language" },
    { "id": "tag-uuid-002", "name": "generics", "category": "topic" }
  ],
  "trend_score": 0.87
}
```

### ArticleDetail（詳細ページ用）

```json
{
  "id": "article-uuid-001",
  "title": "Go 1.23 のジェネリクス改善について",
  "url": "https://example.com/go-1.23-generics",
  "source": {
    "id": "source-uuid-001",
    "name": "Go Blog",
    "site_url": "https://blog.golang.org",
    "category": "tech"
  },
  "author": "Go Team",
  "language": "en",
  "published_at": "2024-03-15T08:00:00Z",
  "summary": {
    "text": "Go 1.23 ではジェネリクスの型推論が大幅に改善され...",
    "model_name": "gpt-4o-mini",
    "model_version": "2024-07-18"
  },
  "tags": [
    { "id": "tag-uuid-001", "name": "go", "category": "language" },
    { "id": "tag-uuid-002", "name": "generics", "category": "topic" }
  ],
  "trend_score": 0.87,
  "is_bookmarked": true,
  "user_feedback": "like",
  "created_at": "2024-03-15T09:00:00Z"
}
```

**注意:** `is_bookmarked` と `user_feedback` は認証済みユーザーのみに返す。未認証時は `null`。

---

## GET /api/v1/articles

記事一覧を取得する。キーワード・タグ・ソースでフィルタリング可能。

### クエリパラメータ

| パラメータ | 型 | デフォルト | 説明 |
|-----------|-----|---------|------|
| `q` | string | - | キーワード検索（タイトル・要約を対象） |
| `tag` | string[] | - | タグ名でフィルタ（複数指定可: `tag=go&tag=backend`） |
| `source_id` | string (UUID) | - | ソース ID でフィルタ |
| `language` | string | - | 言語コードでフィルタ |
| `sort` | string | `published_at` | ソートフィールド: `published_at` / `trend_score` |
| `order` | string | `desc` | ソート順: `asc` / `desc` |
| `page` | integer | 1 | ページ番号 |
| `per_page` | integer | 20 | 1ページあたりの件数（最大 100） |

### レスポンス: 200 OK

```json
{
  "data": [
    { /* ArticleSummary */ },
    { /* ArticleSummary */ }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 350,
    "total_pages": 18,
    "has_next": true,
    "has_prev": false
  }
}
```

---

## GET /api/v1/articles/:id

記事の詳細情報を取得する。

### パスパラメータ

| パラメータ | 型 | 説明 |
|-----------|-----|------|
| `id` | string (UUID) | 記事 ID |

### レスポンス: 200 OK

```json
{
  "data": { /* ArticleDetail */ }
}
```

### エラー

| 条件 | ステータス | コード |
|------|---------|------|
| 記事が存在しない | 404 | `article_not_found` |
| 記事が未処理（status=pending/processing） | 404 | `article_not_found` |

**注意:** `status=processed` の記事のみ公開する。

---

## GET /api/v1/articles/trending

トレンドスコアの高い記事を取得する。認証不要。

### クエリパラメータ

| パラメータ | 型 | デフォルト | 説明 |
|-----------|-----|---------|------|
| `period` | string | `24h` | 集計期間: `24h` / `7d` |
| `language` | string | - | 言語フィルタ |
| `page` | integer | 1 | ページ番号 |
| `per_page` | integer | 20 | 1ページあたりの件数（最大 50） |

### レスポンス: 200 OK

```json
{
  "data": [
    { /* ArticleSummary */ },
    { /* ArticleSummary */ }
  ],
  "pagination": { ... },
  "meta": {
    "period": "24h",
    "generated_at": "2024-03-15T10:00:00Z"
  }
}
```

**キャッシュ:** トレンドフィードは Redis にキャッシュ（TTL: 10 分）。
