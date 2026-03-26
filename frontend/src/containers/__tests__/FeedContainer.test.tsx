import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { FeedContainer } from '../FeedContainer';
import type { PaginatedResponse, RecommendationItem } from '@/types/api';

// ─── Mocks ────────────────────────────────────────────────────────────────────

const mockPush = jest.fn();

jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: mockPush }),
}));

const mockRefreshMutate = jest.fn();
const mockAddMutate = jest.fn();
const mockRemoveMutate = jest.fn();
const mockFeedbackMutate = jest.fn();

jest.mock('@/hooks/useBookmarks', () => ({
  useAddBookmark: () => ({ mutate: mockAddMutate, isPending: false }),
  useRemoveBookmark: () => ({ mutate: mockRemoveMutate, isPending: false }),
}));

jest.mock('@/hooks/useFeedback', () => ({
  useSubmitFeedback: () => ({ mutate: mockFeedbackMutate }),
}));

jest.mock('@/components/ArticleCard', () => ({
  ArticleCard: ({
    article,
    callbacks,
  }: {
    article: { id: string; title: string };
    callbacks?: {
      onBookmark?: (id: string, bookmarked: boolean) => void;
      onFeedback?: (id: string, type: string) => void;
    };
  }) => (
    <div data-testid="article-card">
      {article.title}
      <button onClick={() => callbacks?.onBookmark?.(article.id, false)}>bookmark</button>
      <button onClick={() => callbacks?.onFeedback?.(article.id, 'like')}>like</button>
    </div>
  ),
  ArticleCardSkeleton: () => <div data-testid="skeleton" />,
}));

jest.mock('@/components/ui/Button', () => ({
  Button: ({
    children,
    onClick,
    loading,
  }: {
    children: React.ReactNode;
    onClick?: () => void;
    loading?: boolean;
  }) => (
    <button onClick={onClick} disabled={loading} data-testid="button">
      {children}
    </button>
  ),
}));

const mockItem: RecommendationItem = {
  article: {
    id: 'a1', title: 'Recommended Article', url: 'https://example.com',
    published_at: null, language: 'en', trend_score: 0.8, status: 'processed',
    source: null, summary: null, tags: [],
  },
  score: 0.9,
  reason: 'Relevant to your interests',
  score_breakdown: { relevance: 0.9, freshness: 0.8, trend: 0.7, source_quality: 0.85, personalization: 0.9 },
  user_feedback: null,
  is_bookmarked: false,
};

const mockPage: PaginatedResponse<RecommendationItem> = {
  data: [mockItem],
  pagination: { page: 1, per_page: 20, total: 1, total_pages: 1, has_next: false, has_prev: false },
};

type AuthStatus = 'loading' | 'authenticated' | 'unauthenticated';
let mockAuthStatus: AuthStatus = 'authenticated';
let mockUser = { id: 'u1', email: 'alice@example.com', display_name: 'Alice', is_admin: false, preferred_language: 'ja' as const, created_at: '2025-01-01T00:00:00Z' };

jest.mock('@/contexts/AuthContext', () => ({
  useAuth: () => ({ user: mockUser, status: mockAuthStatus }),
}));

let mockRecommendations = {
  data: { pages: [mockPage] } as { pages: PaginatedResponse<RecommendationItem>[] } | undefined,
  fetchNextPage: jest.fn(),
  hasNextPage: false,
  isFetchingNextPage: false,
  isLoading: false,
  isError: false,
};

jest.mock('@/hooks/useRecommendations', () => ({
  useRecommendations: () => mockRecommendations,
  useRequestRecommendationRefresh: () => ({ mutate: mockRefreshMutate, isPending: false }),
}));

// ─── Tests ────────────────────────────────────────────────────────────────────

describe('FeedContainer', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockAuthStatus = 'authenticated';
    mockRecommendations = {
      data: { pages: [mockPage] },
      fetchNextPage: jest.fn(),
      hasNextPage: false,
      isFetchingNextPage: false,
      isLoading: false,
      isError: false,
    };
  });

  it('認証済みのとき記事一覧を表示する', () => {
    render(<FeedContainer />);

    expect(screen.getByTestId('article-card')).toBeInTheDocument();
    expect(screen.getByText('Recommended Article')).toBeInTheDocument();
  });

  it('未認証のとき null を返す（/login へリダイレクト待ち）', () => {
    mockAuthStatus = 'unauthenticated';

    const { container } = render(<FeedContainer />);

    expect(container.firstChild).toBeNull();
    expect(mockPush).toHaveBeenCalledWith('/login');
  });

  it('loading 中は null を返す', () => {
    mockAuthStatus = 'loading';

    const { container } = render(<FeedContainer />);

    expect(container.firstChild).toBeNull();
  });

  it('isLoading=true のときスケルトンを表示する', () => {
    mockRecommendations = { ...mockRecommendations, isLoading: true, data: undefined };

    render(<FeedContainer />);

    expect(screen.getAllByTestId('skeleton').length).toBeGreaterThan(0);
  });

  it('isError=true のときエラーメッセージを表示する', () => {
    mockRecommendations = { ...mockRecommendations, isError: true, data: undefined };

    render(<FeedContainer />);

    expect(screen.getByText('フィードの読み込みに失敗しました')).toBeInTheDocument();
  });

  it('記事が0件のとき空メッセージを表示する', () => {
    mockRecommendations = {
      ...mockRecommendations,
      data: { pages: [{ ...mockPage, data: [] }] },
    };

    render(<FeedContainer />);

    expect(screen.getByText('まだレコメンドがありません')).toBeInTheDocument();
  });

  it('更新ボタンを押すと refreshMutation.mutate を呼ぶ', () => {
    render(<FeedContainer />);

    const buttons = screen.getAllByTestId('button');
    const refreshBtn = buttons.find((b) => b.textContent?.includes('更新'));
    expect(refreshBtn).toBeDefined();
    fireEvent.click(refreshBtn!);

    expect(mockRefreshMutate).toHaveBeenCalledTimes(1);
  });

  it('ブックマークボタンを押すと addBookmark.mutate を呼ぶ', () => {
    render(<FeedContainer />);

    fireEvent.click(screen.getByText('bookmark'));

    expect(mockAddMutate).toHaveBeenCalledWith('a1');
  });

  it('like ボタンを押すと feedbackMutation.mutate を呼ぶ', () => {
    render(<FeedContainer />);

    fireEvent.click(screen.getByText('like'));

    expect(mockFeedbackMutate).toHaveBeenCalledWith({ articleId: 'a1', feedbackType: 'like' });
  });

  it('hasNextPage=true のとき「もっと見る」ボタンを表示する', () => {
    mockRecommendations = { ...mockRecommendations, hasNextPage: true };

    render(<FeedContainer />);

    expect(screen.getByText('もっと見る')).toBeInTheDocument();
  });

  it('hasNextPage=false のとき「もっと見る」ボタンを表示しない', () => {
    render(<FeedContainer />);

    expect(screen.queryByText('もっと見る')).toBeNull();
  });
});
