import type { ArticleListParams, TrendingParams, SearchParams } from '@/types/api';

export const queryKeys = {
  recommendations: {
    all: ['recommendations'] as const,
    paginated: (page: number) => ['recommendations', page] as const,
  },
  trending: {
    all: ['trending'] as const,
    byPeriod: (period: '24h' | '7d') => ['trending', period] as const,
  },
  articles: {
    all: ['articles'] as const,
    detail: (id: string) => ['articles', id] as const,
    list: (params: ArticleListParams) => ['articles', 'list', params] as const,
    search: (params: SearchParams) => ['articles', 'search', params] as const,
    trending: (params: TrendingParams) => ['articles', 'trending', params] as const,
  },
  bookmarks: {
    all: ['bookmarks'] as const,
    paginated: (page: number) => ['bookmarks', page] as const,
  },
  user: {
    me: ['user', 'me'] as const,
    interests: ['user', 'me', 'interests'] as const,
  },
  admin: {
    stats: ['admin', 'stats'] as const,
    sources: (page?: number) => ['admin', 'sources', page] as const,
    jobs: (page?: number) => ['admin', 'jobs', page] as const,
  },
};
