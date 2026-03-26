import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/lib/api-client';
import { queryKeys } from '@/lib/query-keys';

export function useBookmarkList(page: number, enabled: boolean = true) {
  return useQuery({
    queryKey: queryKeys.bookmarks.paginated(page),
    queryFn: () => apiClient.bookmarks.list({ page, per_page: 20 }),
    enabled,
  });
}

export function useAddBookmark() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (articleId: string) => apiClient.bookmarks.add(articleId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.bookmarks.all });
      qc.invalidateQueries({ queryKey: queryKeys.recommendations.all });
    },
  });
}

export function useRemoveBookmark() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (articleId: string) => apiClient.bookmarks.remove(articleId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.bookmarks.all });
      qc.invalidateQueries({ queryKey: queryKeys.recommendations.all });
    },
  });
}
