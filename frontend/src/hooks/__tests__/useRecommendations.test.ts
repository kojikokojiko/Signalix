import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { useRecommendations, useRequestRecommendationRefresh } from '../useRecommendations';
import { apiClient } from '@/lib/api-client';
import type { PaginatedResponse, RecommendationItem } from '@/types/api';

jest.mock('@/lib/api-client', () => ({
  apiClient: {
    recommendations: {
      getFeed: jest.fn(),
      requestRefresh: jest.fn(),
    },
  },
}));

const mockGetFeed = jest.mocked(apiClient.recommendations.getFeed);
const mockRequestRefresh = jest.mocked(apiClient.recommendations.requestRefresh);

function createWrapper() {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return React.createElement(QueryClientProvider, { client: qc }, children);
  };
}

const mockRecommendation: RecommendationItem = {
  article: {
    id: 'a1', title: 'AI Article', url: 'https://example.com',
    published_at: null, language: 'en', trend_score: 0.9, status: 'processed',
    source: null, summary: 'Summary here', tags: [],
  },
  score: 0.88,
  reason: 'Matches your interests',
  score_breakdown: {
    relevance: 0.9, freshness: 0.8, trend: 0.9, source_quality: 0.85, personalization: 0.88,
  },
  user_feedback: null,
  is_bookmarked: false,
};

const mockPage1: PaginatedResponse<RecommendationItem> = {
  data: [mockRecommendation],
  pagination: { page: 1, per_page: 20, total: 2, total_pages: 2, has_next: true, has_prev: false },
};

const mockPage2: PaginatedResponse<RecommendationItem> = {
  data: [{ ...mockRecommendation, score: 0.75 }],
  pagination: { page: 2, per_page: 20, total: 2, total_pages: 2, has_next: false, has_prev: true },
};

// ─── useRecommendations ───────────────────────────────────────────────────────

describe('useRecommendations', () => {
  beforeEach(() => jest.clearAllMocks());

  it('初回ページのレコメンドを取得する', async () => {
    mockGetFeed.mockResolvedValueOnce(mockPage1);

    const { result } = renderHook(() => useRecommendations(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mockGetFeed).toHaveBeenCalledWith({ page: 1 });
    expect(result.current.data?.pages[0].data[0].score).toBe(0.88);
  });

  it('enabled=false のとき API を呼ばない', () => {
    renderHook(() => useRecommendations(false), { wrapper: createWrapper() });
    expect(mockGetFeed).not.toHaveBeenCalled();
  });

  it('has_next=true のとき getNextPageParam がページ番号を返す', async () => {
    mockGetFeed.mockResolvedValueOnce(mockPage1);

    const { result } = renderHook(() => useRecommendations(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.hasNextPage).toBe(true);
  });

  it('has_next=false のとき getNextPageParam が undefined を返す', async () => {
    mockGetFeed.mockResolvedValueOnce(mockPage2);

    const { result } = renderHook(() => useRecommendations(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.hasNextPage).toBe(false);
  });

  it('fetchNextPage で次のページを取得する', async () => {
    mockGetFeed.mockResolvedValueOnce(mockPage1).mockResolvedValueOnce(mockPage2);

    const { result } = renderHook(() => useRecommendations(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    result.current.fetchNextPage();

    await waitFor(() => expect(result.current.data?.pages).toHaveLength(2));
    expect(mockGetFeed).toHaveBeenCalledTimes(2);
    expect(mockGetFeed).toHaveBeenLastCalledWith({ page: 2 });
  });

  it('API エラー時に isError=true になる', async () => {
    mockGetFeed.mockRejectedValueOnce(new Error('Unauthorized'));

    const { result } = renderHook(() => useRecommendations(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

// ─── useRequestRecommendationRefresh ─────────────────────────────────────────

describe('useRequestRecommendationRefresh', () => {
  beforeEach(() => jest.clearAllMocks());

  it('リフレッシュリクエストを送信する', async () => {
    mockRequestRefresh.mockResolvedValueOnce(undefined);

    const { result } = renderHook(() => useRequestRecommendationRefresh(), {
      wrapper: createWrapper(),
    });

    result.current.mutate();

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockRequestRefresh).toHaveBeenCalledTimes(1);
  });

  it('API エラー時に isError=true になる', async () => {
    mockRequestRefresh.mockRejectedValueOnce(new Error('Rate limited'));

    const { result } = renderHook(() => useRequestRecommendationRefresh(), {
      wrapper: createWrapper(),
    });

    result.current.mutate();

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});
