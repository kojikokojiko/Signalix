import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { TrendingContainer } from '../TrendingContainer';
import type { ArticleSummary, PaginatedResponse } from '@/types/api';

// ─── Mocks ────────────────────────────────────────────────────────────────────

const mockAddBookmark = { mutate: jest.fn(), isPending: false };
const mockRemoveBookmark = { mutate: jest.fn(), isPending: false };

jest.mock('@/hooks/useBookmarks', () => ({
  useAddBookmark: () => mockAddBookmark,
  useRemoveBookmark: () => mockRemoveBookmark,
}));

jest.mock('@/contexts/AuthContext', () => ({
  useAuth: () => ({ user: null }),
}));

const mockTrendingData: PaginatedResponse<ArticleSummary> = {
  data: [
    {
      id: 'a1', title: 'Hot Article 1', url: 'https://example.com/1',
      published_at: null, language: 'en', trend_score: 0.95, status: 'processed',
      source: null, summary: null, tags: [],
    },
    {
      id: 'a2', title: 'Hot Article 2', url: 'https://example.com/2',
      published_at: null, language: 'en', trend_score: 0.88, status: 'processed',
      source: null, summary: null, tags: [],
    },
  ],
  pagination: { page: 1, per_page: 20, total: 2, total_pages: 1, has_next: false, has_prev: false },
};

let mockUseTrendingReturn = {
  data: mockTrendingData as PaginatedResponse<ArticleSummary> | undefined,
  isLoading: false,
};

jest.mock('@/hooks/useTrending', () => ({
  useTrending: () => mockUseTrendingReturn,
}));

jest.mock('@/components/ArticleCard', () => ({
  ArticleCard: ({ article }: { article: { title: string } }) => (
    <div data-testid="article-card">{article.title}</div>
  ),
  ArticleCardSkeleton: () => <div data-testid="skeleton" />,
}));

// ─── Tests ────────────────────────────────────────────────────────────────────

describe('TrendingContainer', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseTrendingReturn = { data: mockTrendingData, isLoading: false };
  });

  it('記事一覧を表示する', () => {
    render(<TrendingContainer />);

    const cards = screen.getAllByTestId('article-card');
    expect(cards).toHaveLength(2);
    expect(cards[0].textContent).toBe('Hot Article 1');
    expect(cards[1].textContent).toBe('Hot Article 2');
  });

  it('ローディング中はスケルトンを表示する', () => {
    mockUseTrendingReturn = { data: undefined, isLoading: true };

    render(<TrendingContainer />);

    expect(screen.getAllByTestId('skeleton').length).toBeGreaterThan(0);
    expect(screen.queryByTestId('article-card')).toBeNull();
  });

  it('記事が0件のとき空メッセージを表示する', () => {
    mockUseTrendingReturn = {
      data: { ...mockTrendingData, data: [] },
      isLoading: false,
    };

    render(<TrendingContainer />);

    expect(screen.getByText('トレンド記事がありません')).toBeInTheDocument();
  });

  it('24h / 7d タブを表示する', () => {
    render(<TrendingContainer />);

    expect(screen.getByText('24時間')).toBeInTheDocument();
    expect(screen.getByText('7日間')).toBeInTheDocument();
  });

  it('7d タブをクリックすると期間が切り替わる', () => {
    render(<TrendingContainer />);

    const tab7d = screen.getByText('7日間');
    fireEvent.click(tab7d);

    // アクティブクラスが付くことを確認（bg-whiteがactiveタブに付く）
    expect(tab7d.className).toContain('bg-white');
  });

  it('24h タブがデフォルトでアクティブ', () => {
    render(<TrendingContainer />);

    const tab24h = screen.getByText('24時間');
    expect(tab24h.className).toContain('bg-white');
  });
});
