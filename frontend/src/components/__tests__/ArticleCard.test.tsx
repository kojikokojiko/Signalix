import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { ArticleCard, ArticleCardSkeleton } from '../ArticleCard';
import type { ArticleSummary, RecommendationItem } from '@/types/api';

// ─── Fixtures ─────────────────────────────────────────────────────────────────

const baseArticle: ArticleSummary = {
  id: 'article-1',
  title: 'Go 1.22 リリースノート',
  url: 'https://go.dev/blog/go1.22',
  published_at: new Date(Date.now() - 60 * 60 * 1000).toISOString(), // 1 hour ago
  language: 'ja',
  trend_score: 0.8,
  status: 'processed',
  source: { id: 'src-1', name: 'Go Blog', category: 'language', language: 'en', site_url: 'https://go.dev/blog', quality_score: 0.9, status: 'active', feed_url: '', description: null, fetch_interval_minutes: 60, consecutive_failures: 0, last_fetched_at: null },
  summary: 'Go 1.22 では range over integers が追加されました。',
  tags: [
    { id: 't1', name: 'go' },
    { id: 't2', name: 'backend' },
  ],
};

const recommendation: RecommendationItem = {
  article: baseArticle,
  recommendation: {
    total_score: 0.85,
    explanation: 'あなたの Go 興味に一致',
    score_breakdown: { relevance: 0.9, freshness: 0.7, trend: 0.8, source_quality: 0.9, personalization: 0.85 },
    generated_at: '2025-01-01T00:00:00Z',
  },
};

// ─── ArticleCard ─────────────────────────────────────────────────────────────

describe('ArticleCard', () => {
  it('タイトルとソース名を表示する', () => {
    render(<ArticleCard article={baseArticle} />);
    expect(screen.getByText('Go 1.22 リリースノート')).toBeInTheDocument();
    expect(screen.getByText('Go Blog')).toBeInTheDocument();
  });

  it('要約テキストを表示する', () => {
    render(<ArticleCard article={baseArticle} />);
    expect(screen.getByText('Go 1.22 では range over integers が追加されました。')).toBeInTheDocument();
  });

  it('タグバッジを表示する (最大3件)', () => {
    const articleWithManyTags: ArticleSummary = {
      ...baseArticle,
      tags: [
        { id: 't1', name: 'go' },
        { id: 't2', name: 'backend' },
        { id: 't3', name: 'performance' },
        { id: 't4', name: 'memory' },
      ],
    };
    render(<ArticleCard article={articleWithManyTags} />);
    expect(screen.getByText('go')).toBeInTheDocument();
    expect(screen.getByText('backend')).toBeInTheDocument();
    expect(screen.getByText('performance')).toBeInTheDocument();
    expect(screen.getByText('+1')).toBeInTheDocument(); // 4th tag is hidden
    expect(screen.queryByText('memory')).not.toBeInTheDocument();
  });

  it('認証済みユーザーにのみアクションボタンを表示する', () => {
    const { rerender } = render(
      <ArticleCard article={baseArticle} recommendation={recommendation} isAuthenticated={false} />
    );
    expect(screen.queryByLabelText('いいね')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('非表示')).not.toBeInTheDocument();

    rerender(
      <ArticleCard article={baseArticle} recommendation={recommendation} isAuthenticated={true} />
    );
    expect(screen.getByLabelText('いいね')).toBeInTheDocument();
    expect(screen.getByLabelText('非表示')).toBeInTheDocument();
  });

  it('ブックマークボタンを showBookmark=false で非表示にする', () => {
    render(
      <ArticleCard article={baseArticle} isAuthenticated={true} showBookmark={false} />
    );
    expect(screen.queryByLabelText('ブックマーク')).not.toBeInTheDocument();
  });

  it('推薦理由を recommendation.reason がある場合のみ表示する', () => {
    const { rerender } = render(<ArticleCard article={baseArticle} />);
    expect(screen.queryByText(/あなたの Go 興味に一致/)).not.toBeInTheDocument();

    rerender(<ArticleCard article={baseArticle} recommendation={recommendation} />);
    expect(screen.getByText('💡 あなたの Go 興味に一致')).toBeInTheDocument();
  });

  it('is_bookmarked=true のとき onBookmark コールバックが isBookmarked=true で呼ばれる', () => {
    const onBookmark = jest.fn();
    const bookmarkedRec = { ...recommendation, is_bookmarked: true };

    render(
      <ArticleCard
        article={baseArticle}
        recommendation={bookmarkedRec}
        isAuthenticated={true}
        callbacks={{ onBookmark }}
      />
    );

    fireEvent.click(screen.getByLabelText('ブックマーク解除'));
    expect(onBookmark).toHaveBeenCalledWith('article-1', true);
  });

  it('いいねボタンクリックで onFeedback("like") が呼ばれる', () => {
    const onFeedback = jest.fn();
    render(
      <ArticleCard
        article={baseArticle}
        recommendation={recommendation}
        isAuthenticated={true}
        callbacks={{ onFeedback }}
      />
    );

    fireEvent.click(screen.getByLabelText('いいね'));
    expect(onFeedback).toHaveBeenCalledWith('article-1', 'like');
  });

  it('非表示ボタンクリックで onFeedback("dislike") が呼ばれる', () => {
    const onFeedback = jest.fn();
    render(
      <ArticleCard
        article={baseArticle}
        recommendation={recommendation}
        isAuthenticated={true}
        callbacks={{ onFeedback }}
      />
    );

    fireEvent.click(screen.getByLabelText('非表示'));
    expect(onFeedback).toHaveBeenCalledWith('article-1', 'dislike');
  });

  it('isBookmarkPending=true のとき ブックマークボタンが disabled', () => {
    render(
      <ArticleCard
        article={baseArticle}
        recommendation={recommendation}
        isAuthenticated={true}
        isBookmarkPending={true}
      />
    );
    expect(screen.getByLabelText('ブックマーク')).toBeDisabled();
  });

  it('外部リンクが正しい url を持つ', () => {
    render(<ArticleCard article={baseArticle} />);
    const link = screen.getByRole('link', { name: '↗' });
    expect(link).toHaveAttribute('href', 'https://go.dev/blog/go1.22');
    expect(link).toHaveAttribute('target', '_blank');
    expect(link).toHaveAttribute('rel', 'noopener noreferrer');
  });

  it('summary が null の場合は要約セクションを表示しない', () => {
    const noSummary: ArticleSummary = { ...baseArticle, summary: null };
    render(<ArticleCard article={noSummary} />);
    expect(screen.queryByText(/Go 1.22 では/)).not.toBeInTheDocument();
  });
});

// ─── ArticleCardSkeleton ──────────────────────────────────────────────────────

describe('ArticleCardSkeleton', () => {
  it('renders without crashing', () => {
    const { container } = render(<ArticleCardSkeleton />);
    expect(container.firstChild).toBeInTheDocument();
  });

  it('animate-pulse クラスを持つ', () => {
    const { container } = render(<ArticleCardSkeleton />);
    expect(container.firstChild).toHaveClass('animate-pulse');
  });
});
