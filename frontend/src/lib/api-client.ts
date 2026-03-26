'use client';

import axios, { AxiosInstance } from 'axios';
import type {
  RegisterInput,
  LoginInput,
  UpdateUserInput,
  UpdateInterestsInput,
  FeedbackInput,
  ArticleListParams,
  TrendingParams,
  PaginationParams,
  AuthResponse,
  TokenResponse,
  ApiResponse,
  PaginatedResponse,
  User,
  UserInterest,
  ArticleSummary,
  ArticleDetail,
  RecommendationItem,
  BookmarkItem,
  AdminStats,
  IngestionJob,
  Source,
} from '@/types/api';

let accessToken: string | null = null;

export function setAccessToken(token: string | null) {
  accessToken = token;
}

export function getAccessToken(): string | null {
  return accessToken;
}

const instance: AxiosInstance = axios.create({
  baseURL: '/api/v1',
  headers: { 'Content-Type': 'application/json' },
  withCredentials: true, // send HttpOnly refresh_token cookie
});

// Attach access token to every request
instance.interceptors.request.use((config) => {
  if (accessToken) {
    config.headers['Authorization'] = `Bearer ${accessToken}`;
  }
  return config;
});

// Auto-refresh on 401 token_expired
let refreshing: Promise<string> | null = null;

instance.interceptors.response.use(
  (response) => response,
  async (error) => {
    const code = error.response?.data?.code as string | undefined;

    if (
      error.response?.status === 401 &&
      code === 'token_expired' &&
      !error.config._retry
    ) {
      error.config._retry = true;

      if (!refreshing) {
        refreshing = instance
          .post<TokenResponse>('/auth/refresh')
          .then((res) => {
            const newToken = res.data.data.access_token;
            setAccessToken(newToken);
            return newToken;
          })
          .finally(() => {
            refreshing = null;
          });
      }

      try {
        const newToken = await refreshing;
        error.config.headers['Authorization'] = `Bearer ${newToken}`;
        return instance(error.config);
      } catch {
        setAccessToken(null);
        window.location.href = '/login';
        return Promise.reject(error);
      }
    }

    return Promise.reject(error);
  }
);

async function get<T>(path: string, params?: Record<string, unknown>): Promise<T> {
  const res = await instance.get<T>(path, { params });
  return res.data;
}

async function post<T>(path: string, data?: unknown): Promise<T> {
  const res = await instance.post<T>(path, data);
  return res.data;
}

async function patch_<T>(path: string, data?: unknown): Promise<T> {
  const res = await instance.patch<T>(path, data);
  return res.data;
}

async function put_<T>(path: string, data?: unknown): Promise<T> {
  const res = await instance.put<T>(path, data);
  return res.data;
}

async function del<T>(path: string): Promise<T> {
  const res = await instance.delete<T>(path);
  return res.data;
}

export const apiClient = {
  auth: {
    register: (data: RegisterInput) => post<AuthResponse>('/auth/register', data),
    login: (data: LoginInput) => post<AuthResponse>('/auth/login', data),
    logout: () => post<void>('/auth/logout'),
    refresh: () => post<TokenResponse>('/auth/refresh'),
  },

  recommendations: {
    getFeed: (params: PaginationParams) =>
      get<PaginatedResponse<RecommendationItem>>('/recommendations', params as Record<string, unknown>),
    requestRefresh: () => post<void>('/recommendations/refresh'),
  },

  articles: {
    list: (params: ArticleListParams) =>
      get<PaginatedResponse<ArticleSummary>>('/articles', params as Record<string, unknown>),
    detail: (id: string) =>
      get<ApiResponse<ArticleDetail>>(`/articles/${id}`),
    trending: (params: TrendingParams) =>
      get<PaginatedResponse<ArticleSummary>>('/articles/trending', params as Record<string, unknown>),
  },

  bookmarks: {
    list: (params: PaginationParams) =>
      get<PaginatedResponse<BookmarkItem>>('/bookmarks', params as Record<string, unknown>),
    add: (articleId: string) =>
      post<ApiResponse<BookmarkItem>>('/bookmarks', { article_id: articleId }),
    remove: (articleId: string) =>
      del<void>(`/bookmarks/${articleId}`),
  },

  feedback: {
    submit: (data: FeedbackInput) =>
      post<ApiResponse<{ article_id: string; feedback_type: string }>>('/feedback', data),
    remove: (articleId: string) =>
      del<void>(`/feedback/${articleId}`),
  },

  users: {
    me: () => get<ApiResponse<User>>('/users/me'),
    update: (data: UpdateUserInput) => patch_<ApiResponse<User>>('/users/me', data),
    interests: () => get<ApiResponse<UserInterest[]>>('/users/me/interests'),
    updateInterests: (data: UpdateInterestsInput) =>
      put_<ApiResponse<UserInterest[]>>('/users/me/interests', data),
  },

  admin: {
    getStats: () => get<ApiResponse<AdminStats>>('/admin/stats'),
    listSources: (params?: PaginationParams) =>
      get<PaginatedResponse<Source>>('/admin/sources', params as Record<string, unknown>),
    createSource: (data: Partial<Source>) =>
      post<ApiResponse<Source>>('/admin/sources', data),
    updateSource: (id: string, data: Partial<Source>) =>
      patch_<ApiResponse<Source>>(`/admin/sources/${id}`, data),
    deleteSource: (id: string) => del<void>(`/admin/sources/${id}`),
    triggerFetch: (id: string) =>
      post<ApiResponse<{ job_id: string }>>(`/admin/sources/${id}/fetch`),
    listJobs: (params?: PaginationParams) =>
      get<PaginatedResponse<IngestionJob>>('/admin/ingestion-jobs', params as Record<string, unknown>),
  },
};
