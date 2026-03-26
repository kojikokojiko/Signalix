import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { useArticle } from '../useArticle';
import { apiClient } from '@/lib/api-client';
import type { ApiResponse, ArticleDetail } from '@/types/api';

jest.mock('@/lib/api-client', () => ({
  apiClient: {
    articles: { detail: jest.fn() },
  },
}));

const mockDetail = jest.mocked(apiClient.articles.detail);

function createWrapper() {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return React.createElement(QueryClientProvider, { client: qc }, children);
  };
}

const mockArticle: ArticleDetail = {
  id: 'a1',
  title: 'Detail Article',
  url: 'https://example.com/article',
  published_at: '2025-01-01T00:00:00Z',
  language: 'en',
  trend_score: 0.7,
  status: 'processed',
  source: null,
  summary: 'A great summary',
  tags: [{ id: 't1', name: 'AI' }],
  clean_content: '<p>Full content here</p>',
};

const mockResponse: ApiResponse<ArticleDetail> = { data: mockArticle };

describe('useArticle', () => {
  beforeEach(() => jest.clearAllMocks());

  it('記事詳細を取得する', async () => {
    mockDetail.mockResolvedValueOnce(mockResponse);

    const { result } = renderHook(() => useArticle('a1'), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mockDetail).toHaveBeenCalledWith('a1');
    expect(result.current.data?.data.title).toBe('Detail Article');
    expect(result.current.data?.data.clean_content).toBe('<p>Full content here</p>');
  });

  it('id が空文字のとき enabled=false でクエリを実行しない', () => {
    renderHook(() => useArticle(''), { wrapper: createWrapper() });
    expect(mockDetail).not.toHaveBeenCalled();
  });

  it('id が変わると再フェッチする', async () => {
    mockDetail.mockResolvedValue(mockResponse);

    const { rerender } = renderHook(
      ({ id }: { id: string }) => useArticle(id),
      { wrapper: createWrapper(), initialProps: { id: 'a1' } }
    );

    await waitFor(() => expect(mockDetail).toHaveBeenCalledWith('a1'));

    rerender({ id: 'a2' });

    await waitFor(() => expect(mockDetail).toHaveBeenCalledWith('a2'));
  });

  it('API エラー時に isError=true になる', async () => {
    mockDetail.mockRejectedValueOnce(new Error('Not found'));

    const { result } = renderHook(() => useArticle('a1'), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});
