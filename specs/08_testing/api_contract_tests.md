# API コントラクト・統合テスト仕様

## 方針

API コントラクトテストは、仕様書（`03_api/` 以下）に定義されたすべてのエンドポイントを
実際の HTTP リクエストで検証する。
テストはリアルな DB・Redis に対して実行する（モックは使わない）。

---

## テストスコープ

| エンドポイント | 正常系 | 異常系 | 認証テスト |
|-------------|-------|-------|---------|
| POST /auth/register | ✓ | メール重複, バリデーション | - |
| POST /auth/login | ✓ | 認証失敗, アカウントロック | - |
| POST /auth/logout | ✓ | - | 未認証 |
| POST /auth/refresh | ✓ | 期限切れトークン | - |
| GET /users/me | ✓ | - | 未認証 |
| PATCH /users/me | ✓ | バリデーション | 未認証 |
| GET /users/me/interests | ✓ | - | 未認証 |
| PUT /users/me/interests | ✓ | 存在しないタグ | 未認証 |
| GET /articles | ✓ | - | - |
| GET /articles/:id | ✓ | 存在しない, 未処理 | - |
| GET /articles/trending | ✓ | - | - |
| GET /recommendations | ✓ | 未設定プロフィール | 未認証 |
| POST /recommendations/refresh | ✓ | レートリミット | 未認証 |
| POST /bookmarks | ✓ | 重複, 記事なし | 未認証 |
| DELETE /bookmarks/:id | ✓ | 存在しない | 未認証 |
| GET /bookmarks | ✓ | - | 未認証 |
| POST /feedback | ✓ | 不正種別, 記事なし | 未認証 |
| DELETE /feedback/:id | ✓ | 存在しない | 未認証 |
| GET /admin/sources | ✓ | - | 管理者以外 |
| POST /admin/sources | ✓ | URL重複, バリデーション | 管理者以外 |

---

## テストケース詳細

### 認証フロー統合テスト

```go
//go:build integration

func TestAuthFlow_RegisterLoginRefreshLogout(t *testing.T) {
    ts := setupTestServer(t)

    // 1. 新規登録
    registerResp := POST(t, ts, "/api/v1/auth/register", map[string]any{
        "email":        "integration@example.com",
        "password":     "Secure1234",
        "display_name": "Integration Test User",
    })
    assert.Equal(t, 201, registerResp.StatusCode)

    var registerBody AuthResponse
    decodeJSON(t, registerResp, &registerBody)
    assert.NotEmpty(t, registerBody.Data.AccessToken)
    accessToken := registerBody.Data.AccessToken

    // Refresh Token が Cookie にセットされていることを確認
    cookies := registerResp.Cookies()
    refreshTokenCookie := findCookie(cookies, "refresh_token")
    require.NotNil(t, refreshTokenCookie)
    assert.True(t, refreshTokenCookie.HttpOnly)

    // 2. 認証が必要なエンドポイントにアクセス
    meResp := GET(t, ts, "/api/v1/users/me", accessToken)
    assert.Equal(t, 200, meResp.StatusCode)

    // 3. Refresh Token でアクセストークン再発行
    refreshResp := POSTWithCookie(t, ts, "/api/v1/auth/refresh", refreshTokenCookie)
    assert.Equal(t, 200, refreshResp.StatusCode)
    var refreshBody TokenResponse
    decodeJSON(t, refreshResp, &refreshBody)
    assert.NotEmpty(t, refreshBody.Data.AccessToken)
    newAccessToken := refreshBody.Data.AccessToken
    assert.NotEqual(t, accessToken, newAccessToken)

    // 4. ログアウト
    logoutResp := POSTWithAuth(t, ts, "/api/v1/auth/logout", newAccessToken, nil)
    assert.Equal(t, 204, logoutResp.StatusCode)

    // 5. ログアウト後は古い Refresh Token が無効
    invalidRefreshResp := POSTWithCookie(t, ts, "/api/v1/auth/refresh", refreshTokenCookie)
    assert.Equal(t, 401, invalidRefreshResp.StatusCode)
}
```

---

### 記事フィード統合テスト

```go
func TestRecommendationFeed_WithInterestProfile(t *testing.T) {
    ts := setupTestServer(t)
    db := testutil.GetDB(ts)

    // テストユーザーと興味プロフィールをセットアップ
    user := testutil.CreateUser(t, db)
    goTag := testutil.FindTag(t, db, "go")
    testutil.CreateUserInterest(t, db, user.ID, goTag.ID, 0.9)

    // Go に関連する処理済み記事を作成
    source := testutil.CreateSource(t, db)
    goArticle := testutil.CreateProcessedArticle(t, db, source.ID, []Tag{goTag})
    otherArticle := testutil.CreateProcessedArticle(t, db, source.ID, nil)

    // レコメンドスコアをセットアップ
    testutil.CreateRecommendationLog(t, db, user.ID, goArticle.ID, 0.9,
        "あなたの Go 興味に一致")
    testutil.CreateRecommendationLog(t, db, user.ID, otherArticle.ID, 0.3,
        "トレンド上位の記事")

    token := testutil.GetAuthToken(t, ts, user.Email, "password")

    resp := GET(t, ts, "/api/v1/recommendations?per_page=20", token)

    assert.Equal(t, 200, resp.StatusCode)
    var body RecommendationsResponse
    decodeJSON(t, resp, &body)

    // スコア降順に並んでいることを確認
    require.Len(t, body.Data, 2)
    assert.Equal(t, goArticle.ID, body.Data[0].Article.ID)
    assert.Equal(t, "あなたの Go 興味に一致", body.Data[0].Recommendation.Explanation)
    assert.Equal(t, otherArticle.ID, body.Data[1].Article.ID)

    // メタ情報を確認
    assert.True(t, body.Meta.HasInterestProfile)
    assert.NotEmpty(t, body.Meta.LastRefreshedAt)
}
```

---

### ブックマーク統合テスト

```go
func TestBookmark_AddAndList(t *testing.T) {
    ts := setupTestServer(t)
    db := testutil.GetDB(ts)

    user := testutil.CreateUser(t, db)
    source := testutil.CreateSource(t, db)
    article := testutil.CreateProcessedArticle(t, db, source.ID, nil)
    token := testutil.GetAuthToken(t, ts, user.Email, "password")

    // 追加
    addResp := POST(t, ts, "/api/v1/bookmarks",
        map[string]any{"article_id": article.ID},
        WithAuth(token),
    )
    assert.Equal(t, 201, addResp.StatusCode)

    // 一覧確認
    listResp := GET(t, ts, "/api/v1/bookmarks", token)
    var listBody BookmarksResponse
    decodeJSON(t, listResp, &listBody)
    require.Len(t, listBody.Data, 1)
    assert.Equal(t, article.ID, listBody.Data[0].Article.ID)

    // 二重登録は 409
    dupResp := POST(t, ts, "/api/v1/bookmarks",
        map[string]any{"article_id": article.ID},
        WithAuth(token),
    )
    assert.Equal(t, 409, dupResp.StatusCode)
    assertErrorCode(t, dupResp, "already_bookmarked")

    // 削除
    deleteResp := DELETE(t, ts, fmt.Sprintf("/api/v1/bookmarks/%s", article.ID), token)
    assert.Equal(t, 204, deleteResp.StatusCode)

    // 削除後は空リスト
    listResp2 := GET(t, ts, "/api/v1/bookmarks", token)
    var listBody2 BookmarksResponse
    decodeJSON(t, listResp2, &listBody2)
    assert.Len(t, listBody2.Data, 0)
}
```

---

### レートリミットテスト

```go
func TestRateLimit_AuthEndpoint(t *testing.T) {
    ts := setupTestServer(t)

    // 認証エンドポイントは 1 分間に 10 リクエストまで
    for i := 0; i < 10; i++ {
        resp := POST(t, ts, "/api/v1/auth/login", map[string]any{
            "email": "test@example.com", "password": "wrong",
        })
        assert.NotEqual(t, 429, resp.StatusCode)
    }

    // 11 回目は 429
    resp := POST(t, ts, "/api/v1/auth/login", map[string]any{
        "email": "test@example.com", "password": "wrong",
    })
    assert.Equal(t, 429, resp.StatusCode)
    assert.NotEmpty(t, resp.Header.Get("Retry-After"))
}
```

---

## レスポンス形式の汎用検証ヘルパー

```go
// testutil/assertions.go

func assertPaginatedResponse(t *testing.T, resp *http.Response, expectedCount int) {
    t.Helper()
    var body PaginatedResponseRaw
    require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
    assert.Len(t, body.Data, expectedCount)
    assert.Positive(t, body.Pagination.Total)
    assert.Positive(t, body.Pagination.TotalPages)
}

func assertErrorCode(t *testing.T, resp *http.Response, expectedCode string) {
    t.Helper()
    var body ErrorResponse
    require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
    assert.Equal(t, expectedCode, body.Error.Code)
}

func assertValidationError(t *testing.T, resp *http.Response, field string) {
    t.Helper()
    assert.Equal(t, 400, resp.StatusCode)
    var body ErrorResponse
    require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
    assert.Equal(t, "validation_error", body.Error.Code)
    fieldErrors := body.Error.Details
    found := false
    for _, fe := range fieldErrors {
        if fe.Field == field {
            found = true
            break
        }
    }
    assert.True(t, found, "expected validation error for field: %s", field)
}
```

---

## E2E テスト（Playwright）

```typescript
// __tests__/e2e/feed.spec.ts
import { test, expect } from '@playwright/test';

test.describe('パーソナライズフィード', () => {
  test('ログイン後にフィードが表示される', async ({ page }) => {
    await page.goto('/login');
    await page.fill('[name="email"]', 'test@example.com');
    await page.fill('[name="password"]', 'Secure1234');
    await page.click('button[type="submit"]');

    await page.waitForURL('/feed');
    await expect(page.locator('[data-testid="article-card"]').first())
      .toBeVisible({ timeout: 10000 });
  });

  test('いいねボタンをクリックするとフィードバックが送信される', async ({ page }) => {
    // ログイン済み状態でテスト
    await page.goto('/feed');

    const firstCard = page.locator('[data-testid="article-card"]').first();
    const likeButton = firstCard.locator('[data-testid="feedback-like"]');

    // API リクエストをインターセプト
    const feedbackRequest = page.waitForRequest(req =>
      req.url().includes('/api/v1/feedback') && req.method() === 'POST'
    );

    await likeButton.click();
    const req = await feedbackRequest;
    const body = req.postDataJSON();
    expect(body.feedback_type).toBe('like');

    // ボタンがアクティブ状態になることを確認
    await expect(likeButton).toHaveClass(/active/);
  });

  test('非表示にするとカードが消える', async ({ page }) => {
    await page.goto('/feed');

    const cards = page.locator('[data-testid="article-card"]');
    const initialCount = await cards.count();

    await cards.first().locator('[data-testid="feedback-hide"]').click();

    await expect(cards).toHaveCount(initialCount - 1);
  });
});
```
