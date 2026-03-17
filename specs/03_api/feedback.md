# API 仕様: フィードバック (Feedback)

## エンドポイント一覧

| メソッド | パス | 認証必須 | 説明 |
|---------|------|---------|------|
| POST | `/api/v1/feedback` | 必要 | フィードバック送信 |
| DELETE | `/api/v1/feedback/:article_id` | 必要 | フィードバック削除（取り消し） |

---

## フィードバック種別

| 種別 | 値 | 意味 | UI での表示 |
|------|-----|------|------------|
| いいね | `like` | 明示的な肯定評価 | 👍 ボタン |
| 非表示 | `dislike` | 明示的な否定評価 | 👎 ボタン |
| 保存 | `save` | ブックマーク保存時に自動記録 | - |
| クリック | `click` | 記事を開いた時に自動記録 | - |
| 非表示にする | `hide` | このフィードから非表示 | 🚫 ボタン |

**レコメンドへの影響:**
- `like`, `save`, `click`: 該当タグの `user_interests.weight` を 0.05 加算（上限 1.0）。
- `dislike`, `hide`: 該当タグの `user_interests.weight` を 0.1 減算（下限 0.0）。
- `hide` は追加で当該記事を当該ユーザーのレコメンドから永久除外する。

---

## POST /api/v1/feedback

記事に対するフィードバックを送信する。

### リクエストボディ

```json
{
  "article_id": "article-uuid-001",
  "feedback_type": "like"
}
```

| フィールド | 型 | 必須 | バリデーション |
|-----------|-----|------|------------|
| `article_id` | string (UUID) | ✓ | 存在する記事の ID |
| `feedback_type` | string | ✓ | `like`, `dislike`, `save`, `click`, `hide` のいずれか |

### レスポンス: 201 Created

```json
{
  "data": {
    "id": "feedback-uuid-001",
    "article_id": "article-uuid-001",
    "feedback_type": "like",
    "created_at": "2024-03-15T10:00:00Z"
  }
}
```

### 上書きルール

同じ `(user_id, article_id)` に対して異なる `feedback_type` を送信した場合:

| 既存 | 新規 | 処理 |
|------|------|------|
| `like` | `dislike` | `like` を削除して `dislike` を挿入 |
| `dislike` | `like` | `dislike` を削除して `like` を挿入 |
| `like` | `like` | 冪等。変更なし |
| 任意 | `click` | 常に挿入（複数記録可） |
| 任意 | `save` | 冪等。変更なし |
| 任意 | `hide` | 既存を削除して `hide` を挿入 |

### エラー

| 条件 | ステータス | コード |
|------|---------|------|
| 記事が存在しない | 404 | `article_not_found` |
| 不正な feedback_type | 400 | `validation_error` |

**冪等性:** `Idempotency-Key` ヘッダー対応。

---

## DELETE /api/v1/feedback/:article_id

記事に対するフィードバックを取り消す。（`click` は取り消し不可）

### パスパラメータ

| パラメータ | 型 | 説明 |
|-----------|-----|------|
| `article_id` | string (UUID) | 記事 ID |

### レスポンス: 204 No Content

フィードバックが削除され、次回のレコメンド計算から反映される。

### エラー

| 条件 | ステータス | コード |
|------|---------|------|
| フィードバックが存在しない | 404 | `feedback_not_found` |
