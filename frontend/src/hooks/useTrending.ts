import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api-client';
import { queryKeys } from '@/lib/query-keys';

export type TrendingPeriod = '24h' | '7d';

export function useTrending(period: TrendingPeriod) {
  return useQuery({
    queryKey: queryKeys.articles.trending({ period }),
    queryFn: () => apiClient.articles.trending({ period, page: 1, per_page: 20 }),
    staleTime: 10 * 60 * 1000,
  });
}
