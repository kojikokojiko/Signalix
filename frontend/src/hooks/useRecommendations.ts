import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/lib/api-client';
import { queryKeys } from '@/lib/query-keys';
import type { PaginatedResponse, RecommendationItem } from '@/types/api';

export function useRecommendations(enabled: boolean = true) {
  return useInfiniteQuery<PaginatedResponse<RecommendationItem>>({
    queryKey: queryKeys.recommendations.all,
    queryFn: ({ pageParam }) =>
      apiClient.recommendations.getFeed({ page: (pageParam as number) ?? 1 }),
    getNextPageParam: (lastPage) =>
      lastPage.pagination.has_next ? lastPage.pagination.page + 1 : undefined,
    initialPageParam: 1,
    enabled,
  });
}

export function useRequestRecommendationRefresh() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => apiClient.recommendations.requestRefresh(),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.recommendations.all });
    },
  });
}
