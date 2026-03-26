import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/lib/api-client';
import { queryKeys } from '@/lib/query-keys';
import type { Source } from '@/types/api';

export function useAdminStats(enabled: boolean = true) {
  return useQuery({
    queryKey: queryKeys.admin.stats,
    queryFn: () => apiClient.admin.getStats(),
    enabled,
    refetchInterval: 30000,
  });
}

export function useAdminSources(page: number = 1, enabled: boolean = true) {
  return useQuery({
    queryKey: queryKeys.admin.sources(page),
    queryFn: () => apiClient.admin.listSources({ per_page: 50 }),
    enabled,
  });
}

export function useCreateSource() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Partial<Source>) => apiClient.admin.createSource(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.admin.sources() });
    },
  });
}

export function useUpdateSource() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Source> }) =>
      apiClient.admin.updateSource(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.admin.sources() });
    },
  });
}

export function useDeleteSource() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiClient.admin.deleteSource(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.admin.sources() });
    },
  });
}

export function useTriggerFetch() {
  return useMutation({
    mutationFn: (id: string) => apiClient.admin.triggerFetch(id),
  });
}

export function useAdminJobs(page: number = 1, enabled: boolean = true) {
  return useQuery({
    queryKey: queryKeys.admin.jobs(page),
    queryFn: () => apiClient.admin.listJobs({ page, per_page: 30 }),
    enabled,
    refetchInterval: 10000,
  });
}
