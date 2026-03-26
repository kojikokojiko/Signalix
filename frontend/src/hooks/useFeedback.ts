import { useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/lib/api-client';
import { queryKeys } from '@/lib/query-keys';

export function useSubmitFeedback() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (params: { articleId: string; feedbackType: 'like' | 'dislike' }) =>
      apiClient.feedback.submit({
        article_id: params.articleId,
        feedback_type: params.feedbackType,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.recommendations.all });
    },
  });
}

export function useRemoveFeedback() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (articleId: string) => apiClient.feedback.remove(articleId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.recommendations.all });
    },
  });
}
