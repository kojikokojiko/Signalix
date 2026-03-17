# バックエンドテスト仕様

## ディレクトリ構造

```
backend/
├── internal/
│   ├── handler/
│   │   ├── auth_handler_test.go
│   │   ├── article_handler_test.go
│   │   └── ...
│   ├── usecase/
│   │   ├── auth_usecase_test.go
│   │   ├── recommendation_usecase_test.go
│   │   └── ...
│   ├── repository/
│   │   ├── article_repository_test.go   # 統合テスト
│   │   └── ...
│   └── worker/
│       ├── ingestion_worker_test.go
│       ├── processing_worker_test.go
│       └── ...
└── internal/testutil/
    ├── fixtures.go
    ├── db.go
    └── helpers.go
```

---

## レイヤー別テスト方針

### Handler テスト

- `httptest.NewRecorder` と `httptest.NewRequest` を使用。
- Usecase はモックインターフェースで差し替え。
- HTTP ステータスコード・レスポンスボディ・ヘッダーを検証。

```go
// handler/auth_handler_test.go
func TestAuthHandler_Register_Success(t *testing.T) {
    mockUsecase := mocks.NewAuthUsecase(t)
    mockUsecase.On("Register", mock.Anything, mock.MatchedBy(func(req RegisterRequest) bool {
        return req.Email == "test@example.com"
    })).Return(&AuthResult{
        AccessToken: "token",
        User:        testUser(),
    }, nil)

    handler := NewAuthHandler(mockUsecase)
    body := `{"email":"test@example.com","password":"Secure1234","display_name":"Test"}`
    req := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    handler.Register(w, req)

    assert.Equal(t, http.StatusCreated, w.Code)

    var resp AuthResponse
    json.Unmarshal(w.Body.Bytes(), &resp)
    assert.Equal(t, "token", resp.Data.AccessToken)
    assert.Equal(t, "test@example.com", resp.Data.User.Email)
}

func TestAuthHandler_Register_DuplicateEmail(t *testing.T) {
    mockUsecase := mocks.NewAuthUsecase(t)
    mockUsecase.On("Register", mock.Anything, mock.Anything).
        Return(nil, ErrEmailAlreadyExists)

    // ... 409 Conflict を検証
}
```

---

### Usecase テスト（ユニットテスト）

- Repository はモックで差し替え。
- ビジネスロジックの正確性を検証。

```go
// usecase/recommendation_usecase_test.go
func TestRecommendationUsecase_BuildScore_CorrectWeights(t *testing.T) {
    // スコア計算式の重みが正しいことを確認
    scores := ScoreBreakdown{
        Relevance:        0.8,
        Freshness:        0.9,
        Trend:            0.7,
        SourceQuality:    0.8,
        Personalization:  0.6,
    }

    totalScore := calculateTotalScore(scores)

    expected := 0.35*0.8 + 0.20*0.9 + 0.20*0.7 + 0.10*0.8 + 0.15*0.6
    assert.InDelta(t, expected, totalScore, 0.001)
}

func TestRecommendationUsecase_FreshnessScore(t *testing.T) {
    tests := []struct {
        name     string
        age      time.Duration
        expected float64
    }{
        {"within 6 hours", 3 * time.Hour, 1.0},
        {"within 12 hours", 8 * time.Hour, 0.85},
        {"within 24 hours", 15 * time.Hour, 0.70},
        {"within 3 days", 2 * 24 * time.Hour, 0.50},
        {"within 7 days", 5 * 24 * time.Hour, 0.30},
        {"within 30 days", 20 * 24 * time.Hour, 0.10},
        {"over 30 days", 60 * 24 * time.Hour, 0.05},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            publishedAt := time.Now().Add(-tt.age)
            score := freshnessScore(publishedAt)
            assert.Equal(t, tt.expected, score)
        })
    }
}
```

---

### Repository テスト（統合テスト）

```go
//go:build integration

// repository/article_repository_test.go
func TestArticleRepository_Create_Success(t *testing.T) {
    db := testutil.SetupDB(t)  // テスト用 DB に接続
    defer testutil.TeardownDB(t, db)

    repo := NewArticleRepository(db)
    ctx := context.Background()

    source := testutil.CreateTestSource(t, db)
    article := Article{
        SourceID:   source.ID,
        URL:        "https://example.com/test-article",
        URLHash:    sha256Hash("https://example.com/test-article"),
        Title:      "テスト記事",
        RawContent: "<p>Content</p>",
        Status:     "pending",
    }

    created, err := repo.Create(ctx, article)

    require.NoError(t, err)
    assert.NotEmpty(t, created.ID)
    assert.Equal(t, "pending", created.Status)
}

func TestArticleRepository_Create_DuplicateURLHash_ReturnsConflict(t *testing.T) {
    db := testutil.SetupDB(t)
    defer testutil.TeardownDB(t, db)

    repo := NewArticleRepository(db)
    source := testutil.CreateTestSource(t, db)

    article := Article{
        SourceID: source.ID,
        URL:      "https://example.com/same",
        URLHash:  sha256Hash("https://example.com/same"),
        Title:    "記事",
        Status:   "pending",
    }
    _, _ = repo.Create(context.Background(), article)

    // 同じ URLHash で再挿入
    _, err := repo.Create(context.Background(), article)

    assert.ErrorIs(t, err, ErrDuplicateURLHash)
}
```

---

### ワーカーテスト

```go
// worker/ingestion_worker_test.go
func TestIngestionWorker_ProcessSource_NewArticles(t *testing.T) {
    // RSS フィードをモックサーバーで返す
    rssServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(testRSSFeed))
    }))
    defer rssServer.Close()

    // DB は minipg、Redis は miniredis
    db := testutil.SetupDB(t)
    redisClient := testutil.SetupRedis(t)

    source := testutil.CreateTestSource(t, db, func(s *Source) {
        s.FeedURL = rssServer.URL
    })

    worker := NewIngestionWorker(db, redisClient)
    result, err := worker.ProcessSource(context.Background(), source.ID)

    require.NoError(t, err)
    assert.Equal(t, 3, result.ArticlesFound)
    assert.Equal(t, 3, result.ArticlesNew)
    assert.Equal(t, 0, result.ArticlesSkipped)

    // DB に記事が挿入されていることを確認
    count := countArticles(t, db, source.ID)
    assert.Equal(t, 3, count)
}

// テスト用 RSS フィード
const testRSSFeed = `<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <item>
      <title>Article 1</title>
      <link>https://example.com/article-1</link>
      <description>Content 1</description>
      <pubDate>Mon, 15 Mar 2024 10:00:00 +0000</pubDate>
    </item>
    <item>
      <title>Article 2</title>
      <link>https://example.com/article-2</link>
      <description>Content 2</description>
      <pubDate>Mon, 15 Mar 2024 09:00:00 +0000</pubDate>
    </item>
    <item>
      <title>Article 3</title>
      <link>https://example.com/article-3</link>
      <description>Content 3</description>
      <pubDate>Mon, 15 Mar 2024 08:00:00 +0000</pubDate>
    </item>
  </channel>
</rss>`
```

---

## バリデーション関数のテスト

```go
// usecase/auth_usecase_test.go
func TestValidatePassword(t *testing.T) {
    tests := []struct {
        password string
        valid    bool
    }{
        {"Secure1234", true},
        {"short1", false},          // 8文字未満
        {"nonnumbers", false},      // 数字なし
        {"12345678", false},        // 英字なし
        {"VeryLongSecure123", true},
    }
    for _, tt := range tests {
        t.Run(tt.password, func(t *testing.T) {
            err := validatePassword(tt.password)
            if tt.valid {
                assert.NoError(t, err)
            } else {
                assert.Error(t, err)
            }
        })
    }
}
```

---

## JWT テスト

```go
func TestJWTService_GenerateAndValidate(t *testing.T) {
    svc := NewJWTService("test-secret")
    user := testUser()

    token, err := svc.Generate(user)
    require.NoError(t, err)
    assert.NotEmpty(t, token)

    claims, err := svc.Validate(token)
    require.NoError(t, err)
    assert.Equal(t, user.ID.String(), claims.Subject)
    assert.Equal(t, user.Email, claims.Email)
}

func TestJWTService_Validate_ExpiredToken(t *testing.T) {
    svc := NewJWTService("test-secret")
    // 有効期限を過去に設定してトークン生成
    token := generateExpiredToken("test-secret")

    _, err := svc.Validate(token)
    assert.ErrorIs(t, err, ErrTokenExpired)
}
```

---

## golangci-lint 設定

```yaml
# .golangci.yml
linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
    - gofmt
    - goimports
    - revive
    - gocritic
    - testifylint  # testify の正しい使用法を強制

linters-settings:
  revive:
    rules:
      - name: exported
        severity: warning
  testifylint:
    enable-all: true
```
