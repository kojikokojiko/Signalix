# フロントエンドテスト仕様

## テストスタック

| ツール | 用途 |
|-------|------|
| Vitest | ユニットテスト・コンポーネントテスト のランナー |
| React Testing Library | コンポーネントテスト |
| MSW (Mock Service Worker) | API モッキング |
| Playwright | E2E テスト |
| @testing-library/user-event | ユーザー操作シミュレーション |

---

## ディレクトリ構造

```
frontend/
├── __tests__/
│   └── e2e/
│       ├── auth.spec.ts
│       ├── feed.spec.ts
│       └── article.spec.ts
├── components/
│   ├── article/
│   │   ├── ArticleCard.tsx
│   │   └── ArticleCard.test.tsx  # コンポーネントと並置
│   └── ...
├── hooks/
│   ├── useRecommendations.ts
│   └── useRecommendations.test.ts
└── lib/
    ├── api-client.ts
    └── api-client.test.ts
```

---

## コンポーネントテスト

### ArticleCard テスト

```typescript
// components/article/ArticleCard.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ArticleCard } from './ArticleCard';
import { mockArticle, mockRecommendation } from '@/testutil/fixtures';

describe('ArticleCard', () => {
  it('タイトルと要約を表示する', () => {
    render(
      <ArticleCard
        article={mockArticle}
        isBookmarked={false}
        userFeedback={null}
        onFeedback={vi.fn()}
        onBookmark={vi.fn()}
      />
    );

    expect(screen.getByText(mockArticle.title)).toBeInTheDocument();
    expect(screen.getByText(mockArticle.summary)).toBeInTheDocument();
    expect(screen.getByText(mockArticle.source.name)).toBeInTheDocument();
  });

  it('レコメンドがある場合、推薦理由を表示する', () => {
    render(
      <ArticleCard
        article={mockArticle}
        recommendation={mockRecommendation}
        isBookmarked={false}
        userFeedback={null}
        onFeedback={vi.fn()}
        onBookmark={vi.fn()}
      />
    );

    expect(screen.getByText(mockRecommendation.explanation)).toBeInTheDocument();
  });

  it('いいねボタンをクリックすると onFeedback("like") が呼ばれる', async () => {
    const user = userEvent.setup();
    const onFeedback = vi.fn();

    render(
      <ArticleCard
        article={mockArticle}
        isBookmarked={false}
        userFeedback={null}
        onFeedback={onFeedback}
        onBookmark={vi.fn()}
      />
    );

    await user.click(screen.getByRole('button', { name: 'いいね' }));

    expect(onFeedback).toHaveBeenCalledWith(mockArticle.id, 'like');
  });

  it('ブックマーク済みの場合、ブックマークボタンがアクティブ状態になる', () => {
    render(
      <ArticleCard
        article={mockArticle}
        isBookmarked={true}
        userFeedback={null}
        onFeedback={vi.fn()}
        onBookmark={vi.fn()}
      />
    );

    const bookmarkButton = screen.getByRole('button', { name: '保存済み' });
    expect(bookmarkButton).toHaveClass('active');
  });

  it('非表示フィードバック後にカードが非表示になる', async () => {
    const user = userEvent.setup();
    const onFeedback = vi.fn();

    const { container } = render(
      <ArticleCard
        article={mockArticle}
        isBookmarked={false}
        userFeedback={null}
        onFeedback={onFeedback}
        onBookmark={vi.fn()}
      />
    );

    await user.click(screen.getByRole('button', { name: '非表示にする' }));

    expect(onFeedback).toHaveBeenCalledWith(mockArticle.id, 'hide');
    // アニメーション後に非表示になることを確認
    await waitFor(() => {
      expect(container.firstChild).toHaveStyle({ display: 'none' });
    });
  });
});
```

---

### カスタムフックテスト

```typescript
// hooks/useRecommendations.test.ts
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClientWrapper } from '@/testutil/wrappers';
import { server } from '@/testutil/msw-server';
import { http, HttpResponse } from 'msw';
import { useRecommendations } from './useRecommendations';

describe('useRecommendations', () => {
  it('正常にレコメンドを取得する', async () => {
    server.use(
      http.get('/api/v1/recommendations', () => {
        return HttpResponse.json({
          data: [mockRecommendationItem],
          pagination: mockPagination,
          meta: { last_refreshed_at: '2024-03-15T10:00:00Z', has_interest_profile: true },
        });
      })
    );

    const { result } = renderHook(() => useRecommendations(), {
      wrapper: QueryClientWrapper,
    });

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(result.current.data?.pages[0].data).toHaveLength(1);
  });

  it('興味プロフィール未設定の場合、has_interest_profile が false になる', async () => {
    server.use(
      http.get('/api/v1/recommendations', () => {
        return HttpResponse.json({
          data: [],
          pagination: emptyPagination,
          meta: { last_refreshed_at: null, has_interest_profile: false },
        });
      })
    );

    const { result } = renderHook(() => useRecommendations(), {
      wrapper: QueryClientWrapper,
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.pages[0].meta.has_interest_profile).toBe(false);
  });
});
```

---

### フォームバリデーションテスト

```typescript
// components/auth/RegisterForm.test.tsx
describe('RegisterForm', () => {
  it('空フォームの送信でバリデーションエラーが表示される', async () => {
    const user = userEvent.setup();
    render(<RegisterForm onSuccess={vi.fn()} />);

    await user.click(screen.getByRole('button', { name: '登録する' }));

    expect(screen.getByText('表示名を入力してください')).toBeInTheDocument();
    expect(screen.getByText('有効なメールアドレスを入力してください')).toBeInTheDocument();
    expect(screen.getByText('パスワードは8文字以上必要です')).toBeInTheDocument();
  });

  it('パスワードに数字が含まれない場合にエラーが表示される', async () => {
    const user = userEvent.setup();
    render(<RegisterForm onSuccess={vi.fn()} />);

    await user.type(screen.getByLabelText('パスワード'), 'NoNumbers!!');
    await user.click(screen.getByRole('button', { name: '登録する' }));

    expect(screen.getByText('数字を含めてください')).toBeInTheDocument();
  });
});
```

---

## MSW ハンドラー設計

```typescript
// testutil/msw-handlers.ts
import { http, HttpResponse } from 'msw';

export const handlers = [
  // デフォルトのハンドラー（各テストでオーバーライド可能）
  http.get('/api/v1/recommendations', () => {
    return HttpResponse.json(mockRecommendationsResponse);
  }),

  http.post('/api/v1/feedback', async ({ request }) => {
    const body = await request.json();
    return HttpResponse.json({
      data: {
        id: 'feedback-id',
        article_id: body.article_id,
        feedback_type: body.feedback_type,
        created_at: new Date().toISOString(),
      },
    }, { status: 201 });
  }),

  http.post('/api/v1/auth/login', async ({ request }) => {
    const body = await request.json();
    if (body.email === 'test@example.com' && body.password === 'Secure1234') {
      return HttpResponse.json(mockAuthResponse);
    }
    return HttpResponse.json(
      { error: { code: 'invalid_credentials', message: '認証に失敗しました' } },
      { status: 401 }
    );
  }),
];
```

---

## テストフィクスチャ

```typescript
// testutil/fixtures.ts
export const mockArticle: ArticleSummary = {
  id: 'article-uuid-001',
  title: 'Go 1.23 のジェネリクス改善について',
  url: 'https://example.com/go-1.23',
  source: {
    id: 'source-uuid-001',
    name: 'Go Blog',
    site_url: 'https://blog.golang.org',
  },
  author: 'Go Team',
  language: 'en',
  published_at: '2024-03-15T08:00:00Z',
  summary: 'Go 1.23 ではジェネリクスの型推論が大幅に改善され...',
  tags: [
    { id: 'tag-uuid-001', name: 'go', category: 'language' },
  ],
  trend_score: 0.87,
};

export const mockRecommendation = {
  total_score: 0.84,
  explanation: 'あなたがよく読む Go バックエンド記事に類似しています',
  score_breakdown: {
    relevance: 0.78,
    freshness: 0.90,
    trend: 0.87,
    source_quality: 0.80,
    personalization: 0.75,
  },
  generated_at: '2024-03-15T09:30:00Z',
};
```
