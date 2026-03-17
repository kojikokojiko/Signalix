# フロントエンド コンポーネント設計

## 設計原則

- **単一責任**: 1 コンポーネント = 1 つの明確な役割。
- **関心の分離**: データフェッチは page/container、表示は presentational コンポーネントが担当。
- **型安全**: すべての props は TypeScript で明示的に型定義する。
- **テスト可能**: 副作用のない純粋なコンポーネントを優先。

---

## コンポーネント一覧

### レイアウト系

| コンポーネント | パス | 説明 |
|-------------|------|------|
| `RootLayout` | `app/layout.tsx` | 全ページ共通レイアウト |
| `Navbar` | `components/layout/Navbar.tsx` | ナビゲーションバー |
| `Footer` | `components/layout/Footer.tsx` | フッター |
| `AdminLayout` | `components/layout/AdminLayout.tsx` | 管理画面レイアウト |

---

### 記事系

| コンポーネント | パス | Props |
|-------------|------|-------|
| `ArticleCard` | `components/article/ArticleCard.tsx` | `article`, `recommendation?`, `onFeedback`, `onBookmark` |
| `ArticleCardSkeleton` | `components/article/ArticleCardSkeleton.tsx` | - |
| `ArticleList` | `components/article/ArticleList.tsx` | `articles`, `loading`, `onLoadMore` |
| `ArticleSummaryBadge` | `components/article/ArticleSummaryBadge.tsx` | `summary` |
| `RecommendationReason` | `components/article/RecommendationReason.tsx` | `explanation` |
| `TagBadge` | `components/article/TagBadge.tsx` | `tag`, `onClick?` |
| `TagBadgeList` | `components/article/TagBadgeList.tsx` | `tags`, `maxVisible` |
| `TrendScoreBar` | `components/article/TrendScoreBar.tsx` | `score` |

---

### フィードバック系

| コンポーネント | パス | Props |
|-------------|------|-------|
| `FeedbackButtons` | `components/feedback/FeedbackButtons.tsx` | `articleId`, `currentFeedback`, `onFeedback` |
| `BookmarkButton` | `components/feedback/BookmarkButton.tsx` | `articleId`, `isBookmarked`, `onToggle` |

---

### フォーム系

| コンポーネント | パス | Props |
|-------------|------|-------|
| `LoginForm` | `components/auth/LoginForm.tsx` | `onSuccess` |
| `RegisterForm` | `components/auth/RegisterForm.tsx` | `onSuccess` |
| `InterestSelector` | `components/settings/InterestSelector.tsx` | `availableTags`, `selectedTags`, `onChange` |
| `SourceForm` | `components/admin/SourceForm.tsx` | `source?`, `onSubmit` |

---

### 共通 UI 系

| コンポーネント | パス | Props |
|-------------|------|-------|
| `Button` | `components/ui/Button.tsx` | `variant`, `size`, `loading`, `disabled`, `onClick`, `children` |
| `Input` | `components/ui/Input.tsx` | `label`, `error`, `...HTMLInputProps` |
| `Badge` | `components/ui/Badge.tsx` | `variant`, `children` |
| `Spinner` | `components/ui/Spinner.tsx` | `size` |
| `Modal` | `components/ui/Modal.tsx` | `open`, `onClose`, `title`, `children` |
| `Pagination` | `components/ui/Pagination.tsx` | `page`, `totalPages`, `onChange` |
| `Alert` | `components/ui/Alert.tsx` | `variant`, `message`, `onClose?` |
| `ConfirmDialog` | `components/ui/ConfirmDialog.tsx` | `open`, `onConfirm`, `onCancel`, `message` |
| `EmptyState` | `components/ui/EmptyState.tsx` | `icon`, `title`, `description`, `action?` |

---

## 重要コンポーネントの詳細仕様

### ArticleCard

フィードの核となるコンポーネント。

```typescript
interface ArticleCardProps {
  article: ArticleSummary;
  recommendation?: {
    total_score: number;
    explanation: string;
  };
  isBookmarked: boolean;
  userFeedback: FeedbackType | null;
  onFeedback: (articleId: string, type: FeedbackType) => void;
  onBookmark: (articleId: string, bookmarked: boolean) => void;
}
```

**表示仕様:**
- カードクリック → 記事詳細ページへ遷移（`/articles/:id`）。
- 外部リンクアイコン → 元記事を新しいタブで開く。
- 推薦理由は `recommendation` が存在する場合のみ表示。
- フィードバックボタンは楽観的更新（即時 UI 反映 → バックグラウンドで API 呼び出し）。
- `onFeedback('hide')` 後はカードをアニメーションで消す（`opacity: 0` → 高さ 0）。

---

### FeedbackButtons

```typescript
type FeedbackType = 'like' | 'dislike' | 'save' | 'click' | 'hide';

interface FeedbackButtonsProps {
  articleId: string;
  currentFeedback: FeedbackType | null;
  onFeedback: (type: FeedbackType) => void;
}
```

**ボタン一覧:**

| アイコン | ラベル | type | 選択済み状態 |
|---------|-------|------|------------|
| 👍 | いいね | `like` | 青塗りつぶし |
| 👎 | 非表示 | `dislike` | 赤塗りつぶし |
| 🚫 | 非表示にする | `hide` | - |
| 🔖 | 保存 | `save` | 黄塗りつぶし |

---

### InterestSelector（オンボーディング・設定）

```typescript
interface Tag {
  id: string;
  name: string;
  category: string;
}

interface InterestSelectorProps {
  availableTags: Tag[];
  selectedTagIds: string[];
  onChange: (selectedIds: string[]) => void;
  maxSelectable?: number;  // デフォルト: 20
}
```

**表示仕様:**
- タグをカテゴリでグループ化して表示。
- 選択済みはチェックマーク付きでハイライト。
- `maxSelectable` に達したら未選択タグをグレーアウト（選択不可）。

---

## 型定義（共有型）

```typescript
// types/api.ts

export interface ArticleSummary {
  id: string;
  title: string;
  url: string;
  source: SourceSummary;
  author: string | null;
  language: string;
  published_at: string;
  summary: string;
  tags: Tag[];
  trend_score: number;
}

export interface ArticleDetail extends ArticleSummary {
  summary: {
    text: string;
    model_name: string;
    model_version: string;
  };
  is_bookmarked: boolean | null;
  user_feedback: FeedbackType | null;
  created_at: string;
}

export interface RecommendationItem {
  article: ArticleSummary & {
    is_bookmarked: boolean;
    user_feedback: FeedbackType | null;
  };
  recommendation: {
    total_score: number;
    explanation: string;
    score_breakdown: ScoreBreakdown;
    generated_at: string;
  };
}

export interface Tag {
  id: string;
  name: string;
  category: string;
}

export interface SourceSummary {
  id: string;
  name: string;
  site_url: string;
}

export type FeedbackType = 'like' | 'dislike' | 'save' | 'click' | 'hide';

export interface PaginationMeta {
  page: number;
  per_page: number;
  total: number;
  total_pages: number;
  has_next: boolean;
  has_prev: boolean;
}

export interface ApiResponse<T> {
  data: T;
}

export interface PaginatedResponse<T> {
  data: T[];
  pagination: PaginationMeta;
}

export interface ApiError {
  error: {
    code: string;
    message: string;
    details?: Array<{ field: string; message: string }>;
  };
}
```
