# API 規約

## 基本情報

- ベース URL: `https://api.signalix.io/api/v1`
- プロトコル: HTTPS 必須（HTTP は 301 リダイレクト）
- データ形式: JSON（`Content-Type: application/json`）
- 文字コード: UTF-8

---

## バージョニング

- URL パスにバージョンを含める: `/api/v1/...`
- 破壊的変更時は `/api/v2/...` にバンプする。
- 旧バージョンは新バージョンリリースから 3 ヶ月間サポートする。

---

## 認証

### JWT ベース認証

```
Authorization: Bearer <access_token>
```

- アクセストークン有効期限: **1 時間**
- Refresh Token 有効期限: **7 日**（HttpOnly Cookie で管理）
- 認証不要なエンドポイントは仕様に明示する。

### トークンリフレッシュ

```
POST /api/v1/auth/refresh
```

Refresh Token（Cookie）を使ってアクセストークンを再発行する。

### 認証エラー

| 状況 | HTTP ステータス | エラーコード |
|------|--------------|------------|
| トークンなし | 401 | `unauthorized` |
| トークン期限切れ | 401 | `token_expired` |
| トークン不正 | 401 | `token_invalid` |
| 権限不足 | 403 | `forbidden` |

---

## リクエスト共通ヘッダー

| ヘッダー | 必須 | 説明 |
|---------|------|------|
| `Authorization` | 認証必須エンドポイントで必須 | `Bearer <token>` |
| `Content-Type` | POST/PUT/PATCH で必須 | `application/json` |
| `X-Request-ID` | 任意 | クライアントがリクエストを追跡する UUID |

---

## レスポンス形式

### 成功レスポンス

**単一リソース:**
```json
{
  "data": {
    "id": "...",
    ...
  }
}
```

**リスト（ページネーション付き）:**
```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 150,
    "total_pages": 8,
    "has_next": true,
    "has_prev": false
  }
}
```

**操作結果のみ:**
```json
{
  "message": "操作が完了しました"
}
```

### エラーレスポンス

```json
{
  "error": {
    "code": "validation_error",
    "message": "入力値にエラーがあります",
    "details": [
      {
        "field": "email",
        "message": "有効なメールアドレスを入力してください"
      }
    ]
  }
}
```

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `code` | string | マシンリーダブルなエラーコード |
| `message` | string | 人間向けエラーメッセージ |
| `details` | array | フィールド別バリデーションエラー（任意） |

---

## HTTP ステータスコード

| コード | 用途 |
|--------|------|
| 200 | GET・PUT・PATCH 成功 |
| 201 | POST によるリソース作成成功 |
| 204 | DELETE 成功（ボディなし） |
| 400 | バリデーションエラー・不正なリクエスト |
| 401 | 未認証 |
| 403 | 認証済みだが権限なし |
| 404 | リソース不存在 |
| 409 | 競合（例: 重複登録） |
| 422 | セマンティクスエラー（処理不可能なエンティティ） |
| 429 | レートリミット超過 |
| 500 | サーバー内部エラー |
| 503 | サービス利用不可（メンテナンス等） |

---

## ページネーション

### クエリパラメータ

| パラメータ | 型 | デフォルト | 説明 |
|-----------|-----|---------|------|
| `page` | integer | 1 | ページ番号（1始まり） |
| `per_page` | integer | 20 | 1ページあたりの件数（最大 100） |

### レスポンスの `pagination` フィールド

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `page` | integer | 現在のページ番号 |
| `per_page` | integer | 1ページあたりの件数 |
| `total` | integer | 総件数 |
| `total_pages` | integer | 総ページ数 |
| `has_next` | boolean | 次のページが存在するか |
| `has_prev` | boolean | 前のページが存在するか |

---

## ソートとフィルタリング

| パラメータ | 説明 | 例 |
|-----------|------|-----|
| `sort` | ソートフィールド | `sort=published_at` |
| `order` | ソート順 | `order=desc`（デフォルト: `desc`） |
| `q` | キーワード検索 | `q=kubernetes` |
| `tag` | タグフィルタ（複数可） | `tag=go&tag=backend` |
| `source_id` | ソースフィルタ | `source_id=uuid` |
| `language` | 言語フィルタ | `language=ja` |

---

## レートリミット

| エンドポイント種別 | 上限 | ウィンドウ |
|-----------------|------|---------|
| 認証エンドポイント | 10 リクエスト | 1 分 |
| 一般 API | 100 リクエスト | 1 分 |
| 検索 API | 30 リクエスト | 1 分 |

レートリミット超過時のレスポンスヘッダー:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1710000000
Retry-After: 60
```

---

## 共通フィールド型

| フィールド | 型 | 形式 |
|-----------|-----|------|
| ID | string (UUID) | `"550e8400-e29b-41d4-a716-446655440000"` |
| 日時 | string | ISO 8601 (`"2024-03-15T10:30:00Z"`) |
| スコア | number | 0.0 〜 1.0 の float |

---

## 冪等性

- `POST` リクエストは `Idempotency-Key` ヘッダーをサポートする。
- 同じキーで 24 時間以内に再送信されたリクエストは、同じレスポンスを返す。
- フィードバック送信・ブックマーク登録に特に有効。

```
Idempotency-Key: <client-generated-uuid>
```
