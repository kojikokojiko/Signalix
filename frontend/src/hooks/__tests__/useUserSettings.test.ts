import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { useUserInterests, useUpdateUser, useUpdateInterests } from '../useUserSettings';
import { apiClient } from '@/lib/api-client';
import type { ApiResponse, UserInterest, User } from '@/types/api';

jest.mock('@/lib/api-client', () => ({
  apiClient: {
    users: {
      interests: jest.fn(),
      update: jest.fn(),
      updateInterests: jest.fn(),
    },
  },
}));

const mockInterests = jest.mocked(apiClient.users.interests);
const mockUpdate = jest.mocked(apiClient.users.update);
const mockUpdateInterests = jest.mocked(apiClient.users.updateInterests);

function createWrapper() {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return React.createElement(QueryClientProvider, { client: qc }, children);
  };
}

const mockInterestData: UserInterest[] = [
  { tag_name: 'AI', weight: 0.8, positive_count: 5, negative_count: 1, created_at: '2025-01-01T00:00:00Z' },
  { tag_name: 'Go', weight: 0.6, positive_count: 3, negative_count: 0, created_at: '2025-01-01T00:00:00Z' },
];

const mockUser: User = {
  id: 'u1', email: 'user@example.com', display_name: 'Test User',
  is_admin: false, preferred_language: 'ja', created_at: '2025-01-01T00:00:00Z',
};

// ─── useUserInterests ─────────────────────────────────────────────────────────

describe('useUserInterests', () => {
  beforeEach(() => jest.clearAllMocks());

  it('ユーザーの興味タグ一覧を取得する', async () => {
    const response: ApiResponse<UserInterest[]> = { data: mockInterestData };
    mockInterests.mockResolvedValueOnce(response);

    const { result } = renderHook(() => useUserInterests(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data?.data).toHaveLength(2);
    expect(result.current.data?.data[0].tag_name).toBe('AI');
  });

  it('enabled=false のとき API を呼ばない', () => {
    renderHook(() => useUserInterests(false), { wrapper: createWrapper() });
    expect(mockInterests).not.toHaveBeenCalled();
  });

  it('API エラー時に isError=true になる', async () => {
    mockInterests.mockRejectedValueOnce(new Error('Unauthorized'));

    const { result } = renderHook(() => useUserInterests(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

// ─── useUpdateUser ────────────────────────────────────────────────────────────

describe('useUpdateUser', () => {
  beforeEach(() => jest.clearAllMocks());

  it('ユーザー情報を更新する', async () => {
    const response: ApiResponse<User> = { data: { ...mockUser, display_name: 'Updated Name' } };
    mockUpdate.mockResolvedValueOnce(response);

    const { result } = renderHook(() => useUpdateUser(), { wrapper: createWrapper() });

    result.current.mutate({ display_name: 'Updated Name' });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockUpdate).toHaveBeenCalledWith({ display_name: 'Updated Name' });
  });

  it('preferred_language を更新できる', async () => {
    const response: ApiResponse<User> = { data: { ...mockUser, preferred_language: 'en' } };
    mockUpdate.mockResolvedValueOnce(response);

    const { result } = renderHook(() => useUpdateUser(), { wrapper: createWrapper() });

    result.current.mutate({ preferred_language: 'en' });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockUpdate).toHaveBeenCalledWith({ preferred_language: 'en' });
  });

  it('API エラー時に isError=true になる', async () => {
    mockUpdate.mockRejectedValueOnce(new Error('Validation error'));

    const { result } = renderHook(() => useUpdateUser(), { wrapper: createWrapper() });

    result.current.mutate({ display_name: '' });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

// ─── useUpdateInterests ───────────────────────────────────────────────────────

describe('useUpdateInterests', () => {
  beforeEach(() => jest.clearAllMocks());

  it('興味タグを更新する', async () => {
    const response: ApiResponse<UserInterest[]> = { data: mockInterestData };
    mockUpdateInterests.mockResolvedValueOnce(response);

    const { result } = renderHook(() => useUpdateInterests(), { wrapper: createWrapper() });

    const interests = [{ tag_name: 'AI', weight: 1.0 }, { tag_name: 'Go', weight: 0.8 }];
    result.current.mutate({ interests });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockUpdateInterests).toHaveBeenCalledWith({ interests });
  });

  it('API エラー時に isError=true になる', async () => {
    mockUpdateInterests.mockRejectedValueOnce(new Error('Server error'));

    const { result } = renderHook(() => useUpdateInterests(), { wrapper: createWrapper() });

    result.current.mutate({ interests: [] });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});
