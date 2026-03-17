# テスト戦略

## 基本方針

**SPEC 駆動 + TDD** の組み合わせで開発する。

1. **仕様を先に定義する** → 本リポジトリのスペックファイルが実装の契約になる。
2. **テストを先に書く** → 実装ロジックの前にテストを書く。
3. **テストがドキュメントになる** → テストケースを読めば仕様がわかる状態を維持する。

---

## テストピラミッド

```
        ┌──────┐
        │ E2E  │  少数・重要フロー (5〜10 シナリオ)
        └──────┘
       ┌────────┐
       │  API   │  中程度・全エンドポイント網羅
       │ Contract│
       └────────┘
     ┌──────────┐
     │Integration│  DB・Redis・ワーカー間の連携
     └──────────┘
   ┌────────────┐
   │    Unit    │  最多・ビジネスロジック単体
   └────────────┘
```

---

## カバレッジ目標

| レイヤー | カバレッジ目標 | 計測方法 |
|---------|-------------|---------|
| Go バックエンド（ユニット） | 80% 以上 | `go test -coverprofile` |
| Go バックエンド（統合） | 主要フロー 100% | 結合テストスイート |
| API コントラクト | 全エンドポイント 100% | httptest |
| フロントエンドコンポーネント | 70% 以上 | vitest/istanbul |
| E2E | クリティカルパス 100% | Playwright |

CI でカバレッジが閾値を下回った場合はビルド失敗とする。

---

## テスト環境

### ローカル開発（docker-compose）

```yaml
# docker-compose.test.yml
services:
  postgres:
    image: pgvector/pgvector:pg16
    environment:
      POSTGRES_DB: signalix_test
      POSTGRES_USER: signalix
      POSTGRES_PASSWORD: test
    ports: ["5432:5432"]

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]
```

### CI 環境

- GitHub Actions のサービスコンテナとして PostgreSQL + Redis を起動。
- テスト実行前に `migrate up` を自動実行。
- テスト間でのデータ汚染防止: 各テストケースはトランザクション内で実行し、終了後にロールバック。

---

## テストデータ戦略

### フィクスチャ設計

```go
// internal/testutil/fixtures.go

type Fixtures struct {
    Users    UserFixtures
    Sources  SourceFixtures
    Articles ArticleFixtures
    Tags     TagFixtures
}

// デフォルトのテストデータセットを作成
func NewFixtures(db *pgxpool.Pool) *Fixtures {
    return &Fixtures{
        Users:    newUserFixtures(db),
        Sources:  newSourceFixtures(db),
        Articles: newArticleFixtures(db),
        Tags:     newTagFixtures(db),
    }
}
```

### テスト用ファクトリ関数

```go
// 最小限の有効なオブジェクトを返し、オプションで上書き可能
func NewTestUser(opts ...func(*User)) User {
    u := User{
        ID:          uuid.New(),
        Email:       "test@example.com",
        DisplayName: "Test User",
        IsActive:    true,
    }
    for _, opt := range opts {
        opt(&u)
    }
    return u
}
```

---

## モック戦略

| 対象 | MVP のアプローチ |
|------|--------------|
| DB（ユニットテスト） | Repository インターフェースをモック化（`mockery` 自動生成） |
| DB（統合テスト） | 実際の PostgreSQL を使用（docker-compose） |
| OpenAI API | レスポンスを返すスタブサーバー（`httptest.NewServer`） |
| Redis | `miniredis` ライブラリ |
| 時刻（time.Now） | `clock` インターフェースを注入してモック可能にする |

**方針:** DB のモックは統合テストでは使わない。実際の DB を使ってスキーマとクエリの正確性を検証する。

---

## テストタグ

```go
// ユニットテスト: タグなし（デフォルトで実行）
func TestSomething(t *testing.T) { ... }

// 統合テスト: タグ付き（CI で分離実行）
//go:build integration
func TestSomethingWithDB(t *testing.T) { ... }
```

```makefile
make test           # ユニットテストのみ
make test-integration  # 統合テストのみ（DB 起動が必要）
make test-all       # 全テスト
```

---

## テスト命名規則

### Go テスト

```go
// ユニットテスト
func TestFunctionName_Condition_ExpectedBehavior(t *testing.T)

// 例:
func TestArticleService_GetFeed_ReturnsEmpty_WhenUserHasNoInterests(t *testing.T)
func TestFreshnessScore_Returns1_WhenPublishedWithin6Hours(t *testing.T)
func TestFreshnessScore_Returns005_WhenPublishedOver30DaysAgo(t *testing.T)
```

### TypeScript テスト

```typescript
// コンポーネントテスト
describe('ArticleCard', () => {
  it('renders article title and summary', () => { ... });
  it('calls onFeedback with "like" when like button clicked', () => { ... });
  it('hides card when "hide" feedback is submitted', () => { ... });
});
```

---

## CI での並列実行

```yaml
# GitHub Actions での並列テスト実行
jobs:
  test:
    strategy:
      matrix:
        suite: [unit, integration, api-contract, frontend]
    steps:
      - run: make test-${{ matrix.suite }}
```
