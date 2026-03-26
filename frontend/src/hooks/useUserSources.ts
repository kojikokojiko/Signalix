'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/lib/api-client';

export const userSourcesKeys = {
  all: ['user-sources'] as const,
};

export function useUserSources(enabled: boolean = true) {
  return useQuery({
    queryKey: userSourcesKeys.all,
    queryFn: () => apiClient.users.listSources(),
    enabled,
  });
}

export function useSubscribeSource() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (sourceId: string) => apiClient.users.subscribeSource(sourceId),
    onSuccess: () => qc.invalidateQueries({ queryKey: userSourcesKeys.all }),
  });
}

export function useUnsubscribeSource() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (sourceId: string) => apiClient.users.unsubscribeSource(sourceId),
    onSuccess: () => qc.invalidateQueries({ queryKey: userSourcesKeys.all }),
  });
}
