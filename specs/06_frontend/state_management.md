# フロントエンド 状態管理・データフェッチ仕様

## 方針

- **サーバーステート** (API データ): React Query (TanStack Query) で管理。
- **クライアントステート** (UI 状態・認証): React Context + useReducer で管理。
- グローバルな状態管理ライブラリ（Zustand, Redux 等）は MVP では使用しない。
- フォームは react-hook-form + Zod でバリデーション。

---

## React Query 設定

```typescript
// lib/query-client.ts
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000,   // 5 分: フィードは 5 分間キャッシュ
      gcTime: 10 * 60 * 1000,     // 10 分: キャッシュの GC
      retry: 2,
      refetchOnWindowFocus: false, // フォーカス時の自動再フェッチは無効
    },
    mutations: {
      retry: 1,
    },
  },
});
```

---

## クエリキー設計

```typescript
// lib/query-keys.ts
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
    search: (params: SearchParams) => ['articles', 'search', params] as const,
  },
  bookmarks: {
    all: ['bookmarks'] as const,
    paginated: (page: number) => ['bookmarks', page] as const,
  },
  user: {
    me: ['user', 'me'] as const,
    interests: ['user', 'me', 'interests'] as const,
  },
};
```

---

## 主要フック仕様

### useRecommendations

```typescript
// hooks/useRecommendations.ts
export function useRecommendations(page: number = 1) {
  return useInfiniteQuery({
    queryKey: queryKeys.recommendations.all,
    queryFn: ({ pageParam = 1 }) =>
      apiClient.recommendations.getFeed({ page: pageParam }),
    getNextPageParam: (lastPage) =>
      lastPage.pagination.has_next
        ? lastPage.pagination.page + 1
        : undefined,
    staleTime: 5 * 60 * 1000,
  });
}
```

### useFeedback

楽観的更新を実装する。

```typescript
// hooks/useFeedback.ts
export function useFeedback() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (params: FeedbackParams) =>
      apiClient.feedback.submit(params),

    // 楽観的更新
    onMutate: async ({ articleId, feedbackType }) => {
      await queryClient.cancelQueries({ queryKey: queryKeys.recommendations.all });

      const previousData = queryClient.getQueryData(queryKeys.recommendations.all);

      // キャッシュを即時更新
      queryClient.setQueryData(
        queryKeys.recommendations.all,
        (old: RecommendationFeedResponse) =>
          updateFeedbackInCache(old, articleId, feedbackType)
      );

      return { previousData };
    },

    // エラー時はロールバック
    onError: (err, variables, context) => {
      queryClient.setQueryData(
        queryKeys.recommendations.all,
        context?.previousData
      );
    },

    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.recommendations.all });
    },
  });
}
```

### useBookmark

```typescript
// hooks/useBookmark.ts
export function useBookmark() {
  const queryClient = useQueryClient();

  const addBookmark = useMutation({
    mutationFn: (articleId: string) => apiClient.bookmarks.add(articleId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.bookmarks.all });
    },
  });

  const removeBookmark = useMutation({
    mutationFn: (articleId: string) => apiClient.bookmarks.remove(articleId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.bookmarks.all });
    },
  });

  return { addBookmark, removeBookmark };
}
```

---

## 認証状態管理

```typescript
// contexts/AuthContext.tsx

interface AuthState {
  user: User | null;
  accessToken: string | null;
  status: 'loading' | 'authenticated' | 'unauthenticated';
}

type AuthAction =
  | { type: 'LOGIN'; payload: { user: User; accessToken: string } }
  | { type: 'LOGOUT' }
  | { type: 'UPDATE_TOKEN'; payload: { accessToken: string } }
  | { type: 'SET_LOADING' };

// アクセストークンはメモリのみに保持（XSS 対策）
// Refresh Token は HttpOnly Cookie（サーバーが管理）
```

### トークン自動リフレッシュ

```typescript
// lib/api-client.ts

// Axios インターセプター
axiosInstance.interceptors.response.use(
  (response) => response,
  async (error) => {
    if (error.response?.status === 401 &&
        error.response?.data?.error?.code === 'token_expired') {
      // Refresh Token で再発行
      const { data } = await axiosInstance.post('/auth/refresh');
      // 新しいアクセストークンをメモリに保存
      setAccessToken(data.access_token);
      // 元のリクエストをリトライ
      error.config.headers['Authorization'] = `Bearer ${data.access_token}`;
      return axiosInstance(error.config);
    }
    return Promise.reject(error);
  }
);
```

---

## API クライアント設計

```typescript
// lib/api-client.ts

export const apiClient = {
  auth: {
    register: (data: RegisterInput) =>
      post<AuthResponse>('/auth/register', data),
    login: (data: LoginInput) =>
      post<AuthResponse>('/auth/login', data),
    logout: () => post('/auth/logout'),
    refresh: () => post<TokenResponse>('/auth/refresh'),
  },

  recommendations: {
    getFeed: (params: PaginationParams) =>
      get<PaginatedResponse<RecommendationItem>>('/recommendations', params),
    requestRefresh: () =>
      post('/recommendations/refresh'),
  },

  articles: {
    list: (params: ArticleListParams) =>
      get<PaginatedResponse<ArticleSummary>>('/articles', params),
    detail: (id: string) =>
      get<ApiResponse<ArticleDetail>>(`/articles/${id}`),
    trending: (params: TrendingParams) =>
      get<PaginatedResponse<ArticleSummary>>('/articles/trending', params),
  },

  bookmarks: {
    list: (params: PaginationParams) =>
      get<PaginatedResponse<BookmarkItem>>('/bookmarks', params),
    add: (articleId: string) =>
      post<ApiResponse<BookmarkResponse>>('/bookmarks', { article_id: articleId }),
    remove: (articleId: string) =>
      del(`/bookmarks/${articleId}`),
  },

  feedback: {
    submit: (data: FeedbackInput) =>
      post<ApiResponse<FeedbackResponse>>('/feedback', data),
    remove: (articleId: string) =>
      del(`/feedback/${articleId}`),
  },

  users: {
    me: () =>
      get<ApiResponse<User>>('/users/me'),
    update: (data: UpdateUserInput) =>
      patch<ApiResponse<User>>('/users/me', data),
    interests: () =>
      get<ApiResponse<UserInterest[]>>('/users/me/interests'),
    updateInterests: (data: UpdateInterestsInput) =>
      put<ApiResponse<UserInterest[]>>('/users/me/interests', data),
  },
};
```

---

## エラーハンドリング

```typescript
// lib/error-handler.ts

export function handleApiError(error: unknown): string {
  if (axios.isAxiosError(error)) {
    const apiError = error.response?.data as ApiError;
    if (apiError?.error?.message) {
      return apiError.error.message;
    }
    if (error.response?.status === 429) {
      return 'リクエストが多すぎます。しばらく待ってから再試行してください。';
    }
    if (error.response?.status >= 500) {
      return 'サーバーエラーが発生しました。しばらく待ってから再試行してください。';
    }
  }
  return '予期しないエラーが発生しました。';
}
```

---

## フォームバリデーション（Zod スキーマ）

```typescript
// lib/validations.ts

export const loginSchema = z.object({
  email: z.string().email('有効なメールアドレスを入力してください'),
  password: z.string().min(1, 'パスワードを入力してください'),
});

export const registerSchema = z.object({
  email: z.string().email('有効なメールアドレスを入力してください'),
  password: z
    .string()
    .min(8, 'パスワードは8文字以上必要です')
    .regex(/[a-zA-Z]/, '英字を含めてください')
    .regex(/[0-9]/, '数字を含めてください'),
  display_name: z
    .string()
    .min(1, '表示名を入力してください')
    .max(50, '表示名は50文字以内にしてください'),
});

export const updateUserSchema = z.object({
  display_name: z.string().min(1).max(50).optional(),
  avatar_url: z.string().url().nullable().optional(),
  preferred_language: z.enum(['ja', 'en']).optional(),
});
```
