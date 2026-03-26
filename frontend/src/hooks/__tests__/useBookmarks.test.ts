import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { useBookmarkList, useAddBookmark, useRemoveBookmark } from '../useBookmarks';
import { apiClient } from '@/lib/api-client';
import type { BookmarkItem, PaginatedResponse } from '@/types/api';

// ─── Mock ─────────────────────────────────────────────────────────────────────

jest.mock('@/lib/api-client', () => ({
  apiClient: {
    bookmarks: {
      list: jest.fn(),
      add: jest.fn(),
      remove: jest.fn(),
    },
    recommendations: { getFeed: jest.fn() },
  },
}));

const mockList = jest.mocked(apiClient.bookmarks.list);
const mockAdd = jest.mocked(apiClient.bookmarks.add);
const mockRemove = jest.mocked(apiClient.bookmarks.remove);

// ─── Test wrapper ─────────────────────────────────────────────────────────────

function createWrapper() {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return React.createElement(QueryClientProvider, { client: qc }, children);
  };
}

// ─── Fixtures ─────────────────────────────────────────────────────────────────

const mockBookmark: BookmarkItem = {
  id: 'bm-1',
  article_id: 'article-1',
  created_at: '2025-01-01T00:00:00Z',
  article: {
    id: 'article-1',
    title: 'Test Article',
    url: 'https://example.com',
    published_at: null,
    language: 'en',
    trend_score: 0.5,
    status: 'processed',
    source: null,
    summary: null,
    tags: [],
  },
};

const mockPaginatedResponse: PaginatedResponse<BookmarkItem> = {
  data: [mockBookmark],
  pagination: {
    page: 1, per_page: 20, total: 1, total_pages: 1, has_next: false, has_prev: false,
  },
};

// ─── useBookmarkList ──────────────────────────────────────────────────────────

describe('useBookmarkList', () => {
  beforeEach(() => jest.clearAllMocks());

  it('ブックマーク一覧を正常に取得する', async () => {
    mockList.mockResolvedValueOnce(mockPaginatedResponse);

    const { result } = renderHook(() => useBookmarkList(1), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data?.data).toHaveLength(1);
    expect(result.current.data?.data[0].id).toBe('bm-1');
  });

  it('enabled=false のとき API を呼ばない', () => {
    renderHook(() => useBookmarkList(1, false), { wrapper: createWrapper() });
    expect(mockList).not.toHaveBeenCalled();
  });

  it('API エラーのとき isError=true になる', async () => {
    mockList.mockRejectedValueOnce(new Error('Network error'));

    const { result } = renderHook(() => useBookmarkList(1), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });

  it('ページ番号をパラメータとして渡す', async () => {
    mockList.mockResolvedValueOnce(mockPaginatedResponse);

    renderHook(() => useBookmarkList(3), { wrapper: createWrapper() });

    await waitFor(() => expect(mockList).toHaveBeenCalledWith({ page: 3, per_page: 20 }));
  });
});

// ─── useAddBookmark ───────────────────────────────────────────────────────────

describe('useAddBookmark', () => {
  beforeEach(() => jest.clearAllMocks());

  it('ブックマーク追加ミューテーションが呼ばれる', async () => {
    mockAdd.mockResolvedValueOnce({ data: mockBookmark });

    const { result } = renderHook(() => useAddBookmark(), { wrapper: createWrapper() });

    result.current.mutate('article-1');

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockAdd).toHaveBeenCalledWith('article-1');
  });

  it('ミューテーションエラー時に isError=true になる', async () => {
    mockAdd.mockRejectedValueOnce(new Error('Already bookmarked'));

    const { result } = renderHook(() => useAddBookmark(), { wrapper: createWrapper() });

    result.current.mutate('article-1');

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

// ─── useRemoveBookmark ────────────────────────────────────────────────────────

describe('useRemoveBookmark', () => {
  beforeEach(() => jest.clearAllMocks());

  it('ブックマーク削除ミューテーションが呼ばれる', async () => {
    mockRemove.mockResolvedValueOnce(undefined);

    const { result } = renderHook(() => useRemoveBookmark(), { wrapper: createWrapper() });

    result.current.mutate('article-1');

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockRemove).toHaveBeenCalledWith('article-1');
  });
});
