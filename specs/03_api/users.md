# API 仕様: ユーザー (Users)

## エンドポイント一覧

| メソッド | パス | 認証必須 | 説明 |
|---------|------|---------|------|
| GET | `/api/v1/users/me` | 必要 | 自分のプロフィール取得 |
| PATCH | `/api/v1/users/me` | 必要 | プロフィール更新 |
| GET | `/api/v1/users/me/interests` | 必要 | 興味タグ一覧取得 |
| PUT | `/api/v1/users/me/interests` | 必要 | 興味タグ一括更新 |
| DELETE | `/api/v1/users/me` | 必要 | アカウント削除 |

---

## GET /api/v1/users/me

ログイン中ユーザーのプロフィールを取得する。

### レスポンス: 200 OK

```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "display_name": "Koji Iwase",
    "avatar_url": null,
    "preferred_language": "ja",
    "is_admin": false,
    "last_login_at": "2024-03-15T09:00:00Z",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-03-15T09:00:00Z"
  }
}
```

---

## PATCH /api/v1/users/me

プロフィール情報を部分更新する。送信したフィールドのみ更新される。

### リクエストボディ

```json
{
  "display_name": "Koji Iwase",
  "avatar_url": "https://example.com/avatar.jpg",
  "preferred_language": "ja"
}
```

| フィールド | 型 | バリデーション |
|-----------|-----|------------|
| `display_name` | string | 1〜50 文字 |
| `avatar_url` | string or null | 有効な URL または null |
| `preferred_language` | string | ISO 639-1 言語コード（例: "ja", "en"） |

### レスポンス: 200 OK

更新後のプロフィールを返す（GET /users/me と同じ形式）。

---

## GET /api/v1/users/me/interests

ユーザーの興味タグ一覧を重み降順で取得する。

### レスポンス: 200 OK

```json
{
  "data": [
    {
      "tag": {
        "id": "tag-uuid-001",
        "name": "go",
        "category": "language"
      },
      "weight": 0.9,
      "source": "manual",
      "updated_at": "2024-03-15T10:00:00Z"
    },
    {
      "tag": {
        "id": "tag-uuid-002",
        "name": "kubernetes",
        "category": "infrastructure"
      },
      "weight": 0.7,
      "source": "inferred",
      "updated_at": "2024-03-14T08:00:00Z"
    }
  ]
}
```

---

## PUT /api/v1/users/me/interests

興味タグを一括更新する。送信したタグリストで完全に置き換える（削除を含む）。

### リクエストボディ

```json
{
  "interests": [
    { "tag_id": "tag-uuid-001", "weight": 0.9 },
    { "tag_id": "tag-uuid-002", "weight": 0.7 },
    { "tag_id": "tag-uuid-003", "weight": 0.5 }
  ]
}
```

| フィールド | 型 | バリデーション |
|-----------|-----|------------|
| `interests` | array | 1〜20 件 |
| `interests[].tag_id` | string (UUID) | 存在するタグ ID |
| `interests[].weight` | number | 0.1〜1.0 |

### レスポンス: 200 OK

更新後の興味リスト全体を返す（GET /interests と同じ形式）。

### エラー

| 条件 | ステータス | コード |
|------|---------|------|
| 存在しない tag_id | 422 | `tag_not_found` |
| 20件超 | 400 | `validation_error` |

---

## DELETE /api/v1/users/me

アカウントを削除する。この操作は取り消せない。

### リクエストボディ

```json
{
  "password": "Secure1234"
}
```

パスワードを確認として要求する。

### レスポンス: 204 No Content

**削除される内容:**
- ユーザーアカウント
- 興味プロフィール
- ブックマーク
- フィードバック
- レコメンドログ

**保持される内容:**
- ユーザーが閲覧したことで影響を受けたシステムのアグリゲートデータ（匿名化されたトレンドスコア等）

### エラー

| 条件 | ステータス | コード |
|------|---------|------|
| パスワード不一致 | 401 | `invalid_credentials` |
