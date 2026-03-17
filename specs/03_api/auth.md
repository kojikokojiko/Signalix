# API 仕様: 認証 (Auth)

## エンドポイント一覧

| メソッド | パス | 認証必須 | 説明 |
|---------|------|---------|------|
| POST | `/api/v1/auth/register` | 不要 | 新規ユーザー登録 |
| POST | `/api/v1/auth/login` | 不要 | ログイン |
| POST | `/api/v1/auth/logout` | 必要 | ログアウト |
| POST | `/api/v1/auth/refresh` | 不要（Cookie） | アクセストークン再発行 |

---

## POST /api/v1/auth/register

新規ユーザーを登録し、ログイン済み状態のトークンを返す。

### リクエストボディ

```json
{
  "email": "user@example.com",
  "password": "Secure1234",
  "display_name": "Koji Iwase"
}
```

| フィールド | 型 | 必須 | バリデーション |
|-----------|-----|------|------------|
| `email` | string | ✓ | RFC 5321 準拠の形式。255 文字以内 |
| `password` | string | ✓ | 8文字以上。英字と数字を含む |
| `display_name` | string | ✓ | 1〜50 文字 |

### レスポンス: 201 Created

```json
{
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "token_type": "Bearer",
    "expires_in": 3600,
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "user@example.com",
      "display_name": "Koji Iwase",
      "is_admin": false,
      "created_at": "2024-03-15T10:30:00Z"
    }
  }
}
```

Refresh Token は `Set-Cookie` ヘッダーで設定される:
```
Set-Cookie: refresh_token=<token>; HttpOnly; Secure; SameSite=Strict; Max-Age=604800; Path=/api/v1/auth/refresh
```

### エラー

| 条件 | ステータス | コード |
|------|---------|------|
| メールアドレス重複 | 409 | `email_already_exists` |
| バリデーションエラー | 400 | `validation_error` |

---

## POST /api/v1/auth/login

メールアドレスとパスワードで認証し、トークンを返す。

### リクエストボディ

```json
{
  "email": "user@example.com",
  "password": "Secure1234"
}
```

| フィールド | 型 | 必須 |
|-----------|-----|------|
| `email` | string | ✓ |
| `password` | string | ✓ |

### レスポンス: 200 OK

```json
{
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "token_type": "Bearer",
    "expires_in": 3600,
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "user@example.com",
      "display_name": "Koji Iwase",
      "is_admin": false,
      "created_at": "2024-03-15T10:30:00Z"
    }
  }
}
```

Refresh Token は `Set-Cookie` で設定される（register と同様）。

### エラー

| 条件 | ステータス | コード |
|------|---------|------|
| 認証失敗 | 401 | `invalid_credentials` |
| アカウントロック | 403 | `account_locked` |
| アカウント無効 | 403 | `account_disabled` |

**ロックポリシー:** 5 回連続失敗で 15 分ロック。失敗カウントは Redis で管理。

---

## POST /api/v1/auth/logout

現在のセッションを終了し、Refresh Token を無効化する。

### リクエスト

ボディ不要。`Authorization` ヘッダーのアクセストークンを使用。

### レスポンス: 204 No Content

Refresh Token Cookie をクリアする:
```
Set-Cookie: refresh_token=; HttpOnly; Secure; Max-Age=0; Path=/api/v1/auth/refresh
```

---

## POST /api/v1/auth/refresh

Cookie の Refresh Token を使ってアクセストークンを再発行する。

### リクエスト

ボディ不要。Cookie の `refresh_token` を使用。

### レスポンス: 200 OK

```json
{
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "token_type": "Bearer",
    "expires_in": 3600
  }
}
```

### エラー

| 条件 | ステータス | コード |
|------|---------|------|
| Refresh Token なし | 401 | `unauthorized` |
| Refresh Token 期限切れ | 401 | `token_expired` |
| Refresh Token 無効（ログアウト済み） | 401 | `token_invalid` |

---

## JWT ペイロード仕様

```json
{
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "is_admin": false,
  "iat": 1710000000,
  "exp": 1710003600,
  "jti": "unique-token-id"
}
```

| クレーム | 説明 |
|---------|------|
| `sub` | ユーザー ID（UUID） |
| `email` | メールアドレス |
| `is_admin` | 管理者フラグ |
| `iat` | 発行日時（Unix タイムスタンプ） |
| `exp` | 有効期限（Unix タイムスタンプ） |
| `jti` | JWT ID（トークンの一意識別子） |

署名アルゴリズム: **HS256**（MVP）→ RS256 に移行予定
