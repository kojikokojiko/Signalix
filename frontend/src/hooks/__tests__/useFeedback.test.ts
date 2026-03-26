import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { useSubmitFeedback, useRemoveFeedback } from '../useFeedback';
import { apiClient } from '@/lib/api-client';

// ─── Mock ─────────────────────────────────────────────────────────────────────

jest.mock('@/lib/api-client', () => ({
  apiClient: {
    feedback: {
      submit: jest.fn(),
      remove: jest.fn(),
    },
    recommendations: { getFeed: jest.fn() },
  },
}));

const mockSubmit = jest.mocked(apiClient.feedback.submit);
const mockRemove = jest.mocked(apiClient.feedback.remove);

function createWrapper() {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return React.createElement(QueryClientProvider, { client: qc }, children);
  };
}

// ─── useSubmitFeedback ────────────────────────────────────────────────────────

describe('useSubmitFeedback', () => {
  beforeEach(() => jest.clearAllMocks());

  it('like フィードバックを送信する', async () => {
    mockSubmit.mockResolvedValueOnce({ data: { article_id: 'a1', feedback_type: 'like' } });

    const { result } = renderHook(() => useSubmitFeedback(), { wrapper: createWrapper() });

    result.current.mutate({ articleId: 'a1', feedbackType: 'like' });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockSubmit).toHaveBeenCalledWith({ article_id: 'a1', feedback_type: 'like' });
  });

  it('dislike フィードバックを送信する', async () => {
    mockSubmit.mockResolvedValueOnce({ data: { article_id: 'a1', feedback_type: 'dislike' } });

    const { result } = renderHook(() => useSubmitFeedback(), { wrapper: createWrapper() });

    result.current.mutate({ articleId: 'a1', feedbackType: 'dislike' });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockSubmit).toHaveBeenCalledWith({ article_id: 'a1', feedback_type: 'dislike' });
  });

  it('API エラー時に isError=true になる', async () => {
    mockSubmit.mockRejectedValueOnce(new Error('Server error'));

    const { result } = renderHook(() => useSubmitFeedback(), { wrapper: createWrapper() });

    result.current.mutate({ articleId: 'a1', feedbackType: 'like' });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

// ─── useRemoveFeedback ────────────────────────────────────────────────────────

describe('useRemoveFeedback', () => {
  beforeEach(() => jest.clearAllMocks());

  it('フィードバックを削除する', async () => {
    mockRemove.mockResolvedValueOnce(undefined);

    const { result } = renderHook(() => useRemoveFeedback(), { wrapper: createWrapper() });

    result.current.mutate('article-1');

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockRemove).toHaveBeenCalledWith('article-1');
  });
});
