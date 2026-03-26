import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { useTrending } from '../useTrending';
import { apiClient } from '@/lib/api-client';
import type { ArticleSummary, PaginatedResponse } from '@/types/api';

jest.mock('@/lib/api-client', () => ({
  apiClient: { articles: { trending: jest.fn() } },
}));

const mockTrending = jest.mocked(apiClient.articles.trending);

function createWrapper() {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return React.createElement(QueryClientProvider, { client: qc }, children);
  };
}

const mockArticle: ArticleSummary = {
  id: 'a1', title: 'Trending Article', url: 'https://example.com',
  published_at: null, language: 'en', trend_score: 0.95, status: 'processed',
  source: null, summary: null, tags: [],
};

const mockResponse: PaginatedResponse<ArticleSummary> = {
  data: [mockArticle],
  pagination: { page: 1, per_page: 20, total: 1, total_pages: 1, has_next: false, has_prev: false },
};

describe('useTrending', () => {
  beforeEach(() => jest.clearAllMocks());

  it('24h トレンド記事を取得する', async () => {
    mockTrending.mockResolvedValueOnce(mockResponse);

    const { result } = renderHook(() => useTrending('24h'), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mockTrending).toHaveBeenCalledWith({ period: '24h', page: 1, per_page: 20 });
    expect(result.current.data?.data[0].title).toBe('Trending Article');
  });

  it('7d トレンド記事を取得する', async () => {
    mockTrending.mockResolvedValueOnce(mockResponse);

    const { result } = renderHook(() => useTrending('7d'), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mockTrending).toHaveBeenCalledWith({ period: '7d', page: 1, per_page: 20 });
  });

  it('period が変わるとクエリキーが変わり再フェッチする', async () => {
    mockTrending.mockResolvedValue(mockResponse);

    const { result, rerender } = renderHook(
      ({ period }: { period: '24h' | '7d' }) => useTrending(period),
      { wrapper: createWrapper(), initialProps: { period: '24h' } }
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockTrending).toHaveBeenCalledTimes(1);

    rerender({ period: '7d' });
    await waitFor(() => expect(mockTrending).toHaveBeenCalledTimes(2));
    expect(mockTrending).toHaveBeenLastCalledWith({ period: '7d', page: 1, per_page: 20 });
  });

  it('API エラー時に isError=true になる', async () => {
    mockTrending.mockRejectedValueOnce(new Error('Server error'));

    const { result } = renderHook(() => useTrending('24h'), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});
