import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import {
  useAdminStats,
  useAdminSources,
  useCreateSource,
  useUpdateSource,
  useDeleteSource,
  useTriggerFetch,
  useAdminJobs,
} from '../useAdminStats';
import { apiClient } from '@/lib/api-client';
import type { ApiResponse, PaginatedResponse, AdminStats, Source, IngestionJob } from '@/types/api';

jest.mock('@/lib/api-client', () => ({
  apiClient: {
    admin: {
      getStats: jest.fn(),
      listSources: jest.fn(),
      createSource: jest.fn(),
      updateSource: jest.fn(),
      deleteSource: jest.fn(),
      triggerFetch: jest.fn(),
      listJobs: jest.fn(),
    },
  },
}));

const mockGetStats = jest.mocked(apiClient.admin.getStats);
const mockListSources = jest.mocked(apiClient.admin.listSources);
const mockCreateSource = jest.mocked(apiClient.admin.createSource);
const mockUpdateSource = jest.mocked(apiClient.admin.updateSource);
const mockDeleteSource = jest.mocked(apiClient.admin.deleteSource);
const mockTriggerFetch = jest.mocked(apiClient.admin.triggerFetch);
const mockListJobs = jest.mocked(apiClient.admin.listJobs);

function createWrapper() {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false, refetchInterval: false }, mutations: { retry: false } },
  });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return React.createElement(QueryClientProvider, { client: qc }, children);
  };
}

const mockStats: AdminStats = {
  sources: { total: 10, active: 8, degraded: 1, disabled: 1 },
  articles: { total: 500, processed: 480, pending: 15, failed: 5 },
  ingestion_jobs: { last_24h_completed: 20, last_24h_failed: 2 },
  users: { total: 100, active_last_7d: 30 },
};

const mockSource: Source = {
  id: 's1', name: 'TechCrunch', category: 'news', language: 'en',
  site_url: 'https://techcrunch.com', quality_score: 0.9, status: 'active',
  feed_url: 'https://techcrunch.com/feed', description: null,
  fetch_interval_minutes: 60, consecutive_failures: 0, last_fetched_at: null,
};

const mockJob: IngestionJob = {
  id: 'j1', source_id: 's1', source_name: 'TechCrunch', status: 'completed',
  articles_found: 10, articles_new: 3, articles_skipped: 7,
  error_message: null, started_at: '2025-01-01T00:00:00Z', completed_at: '2025-01-01T00:01:00Z',
};

// ─── useAdminStats ────────────────────────────────────────────────────────────

describe('useAdminStats', () => {
  beforeEach(() => jest.clearAllMocks());

  it('管理者統計を取得する', async () => {
    const response: ApiResponse<AdminStats> = { data: mockStats };
    mockGetStats.mockResolvedValueOnce(response);

    const { result } = renderHook(() => useAdminStats(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data?.data.sources.total).toBe(10);
    expect(result.current.data?.data.users.total).toBe(100);
  });

  it('enabled=false のとき API を呼ばない', () => {
    renderHook(() => useAdminStats(false), { wrapper: createWrapper() });
    expect(mockGetStats).not.toHaveBeenCalled();
  });

  it('API エラー時に isError=true になる', async () => {
    mockGetStats.mockRejectedValueOnce(new Error('Forbidden'));

    const { result } = renderHook(() => useAdminStats(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

// ─── useAdminSources ──────────────────────────────────────────────────────────

describe('useAdminSources', () => {
  beforeEach(() => jest.clearAllMocks());

  it('ソース一覧を取得する', async () => {
    const response: PaginatedResponse<Source> = {
      data: [mockSource],
      pagination: { page: 1, per_page: 50, total: 1, total_pages: 1, has_next: false, has_prev: false },
    };
    mockListSources.mockResolvedValueOnce(response);

    const { result } = renderHook(() => useAdminSources(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data?.data[0].name).toBe('TechCrunch');
  });

  it('enabled=false のとき API を呼ばない', () => {
    renderHook(() => useAdminSources(1, false), { wrapper: createWrapper() });
    expect(mockListSources).not.toHaveBeenCalled();
  });
});

// ─── useCreateSource ─────────────────────────────────────────────────────────

describe('useCreateSource', () => {
  beforeEach(() => jest.clearAllMocks());

  it('ソースを作成する', async () => {
    const response: ApiResponse<Source> = { data: mockSource };
    mockCreateSource.mockResolvedValueOnce(response);

    const { result } = renderHook(() => useCreateSource(), { wrapper: createWrapper() });

    result.current.mutate({ name: 'TechCrunch', feed_url: 'https://techcrunch.com/feed' });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockCreateSource).toHaveBeenCalledWith({
      name: 'TechCrunch',
      feed_url: 'https://techcrunch.com/feed',
    });
  });

  it('API エラー時に isError=true になる', async () => {
    mockCreateSource.mockRejectedValueOnce(new Error('Conflict'));

    const { result } = renderHook(() => useCreateSource(), { wrapper: createWrapper() });

    result.current.mutate({ feed_url: 'https://duplicate.com/feed' });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

// ─── useUpdateSource ─────────────────────────────────────────────────────────

describe('useUpdateSource', () => {
  beforeEach(() => jest.clearAllMocks());

  it('ソースを更新する', async () => {
    const response: ApiResponse<Source> = { data: { ...mockSource, status: 'disabled' } };
    mockUpdateSource.mockResolvedValueOnce(response);

    const { result } = renderHook(() => useUpdateSource(), { wrapper: createWrapper() });

    result.current.mutate({ id: 's1', data: { status: 'disabled' } });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockUpdateSource).toHaveBeenCalledWith('s1', { status: 'disabled' });
  });
});

// ─── useDeleteSource ─────────────────────────────────────────────────────────

describe('useDeleteSource', () => {
  beforeEach(() => jest.clearAllMocks());

  it('ソースを削除する', async () => {
    mockDeleteSource.mockResolvedValueOnce(undefined);

    const { result } = renderHook(() => useDeleteSource(), { wrapper: createWrapper() });

    result.current.mutate('s1');

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockDeleteSource).toHaveBeenCalledWith('s1');
  });
});

// ─── useTriggerFetch ─────────────────────────────────────────────────────────

describe('useTriggerFetch', () => {
  beforeEach(() => jest.clearAllMocks());

  it('手動フェッチをトリガーする', async () => {
    const response: ApiResponse<{ job_id: string }> = { data: { job_id: 'j1' } };
    mockTriggerFetch.mockResolvedValueOnce(response);

    const { result } = renderHook(() => useTriggerFetch(), { wrapper: createWrapper() });

    result.current.mutate('s1');

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockTriggerFetch).toHaveBeenCalledWith('s1');
  });

  it('API エラー時に isError=true になる', async () => {
    mockTriggerFetch.mockRejectedValueOnce(new Error('Source not found'));

    const { result } = renderHook(() => useTriggerFetch(), { wrapper: createWrapper() });

    result.current.mutate('nonexistent');

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

// ─── useAdminJobs ─────────────────────────────────────────────────────────────

describe('useAdminJobs', () => {
  beforeEach(() => jest.clearAllMocks());

  it('ジョブ一覧を取得する', async () => {
    const response: PaginatedResponse<IngestionJob> = {
      data: [mockJob],
      pagination: { page: 1, per_page: 30, total: 1, total_pages: 1, has_next: false, has_prev: false },
    };
    mockListJobs.mockResolvedValueOnce(response);

    const { result } = renderHook(() => useAdminJobs(), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data?.data[0].status).toBe('completed');
    expect(result.current.data?.data[0].articles_new).toBe(3);
  });

  it('ページ番号をパラメータとして渡す', async () => {
    const response: PaginatedResponse<IngestionJob> = {
      data: [], pagination: { page: 2, per_page: 30, total: 0, total_pages: 0, has_next: false, has_prev: true },
    };
    mockListJobs.mockResolvedValueOnce(response);

    renderHook(() => useAdminJobs(2), { wrapper: createWrapper() });

    await waitFor(() => expect(mockListJobs).toHaveBeenCalledWith({ page: 2, per_page: 30 }));
  });

  it('enabled=false のとき API を呼ばない', () => {
    renderHook(() => useAdminJobs(1, false), { wrapper: createWrapper() });
    expect(mockListJobs).not.toHaveBeenCalled();
  });
});
