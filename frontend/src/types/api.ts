// ─── Pagination ──────────────────────────────────────────────────────────────

export interface Pagination {
  page: number;
  per_page: number;
  total: number;
  total_pages: number;
  has_next: boolean;
  has_prev: boolean;
}

export interface PaginatedResponse<T> {
  data: T[];
  pagination: Pagination;
}

export interface ApiResponse<T> {
  data: T;
}

export interface ApiError {
  code: string;
  message: string;
}

// ─── User ────────────────────────────────────────────────────────────────────

export interface User {
  id: string;
  email: string;
  display_name: string;
  is_admin: boolean;
  preferred_language: 'ja' | 'en';
  created_at: string;
}

export interface UserInterest {
  tag_name: string;
  weight: number;
  positive_count: number;
  negative_count: number;
  created_at: string;
}

// ─── Auth ────────────────────────────────────────────────────────────────────

export interface AuthResponse {
  data: {
    access_token: string;
    token_type: string;
    expires_in: number;
    user: User;
  };
}

export interface TokenResponse {
  data: {
    access_token: string;
    token_type: string;
    expires_in: number;
  };
}

// ─── Article ─────────────────────────────────────────────────────────────────

export interface Tag {
  id: string;
  name: string;
  confidence?: number;
}

export interface Source {
  id: string;
  name: string;
  category: string;
  language: string;
  site_url: string;
  quality_score: number;
  status: string;
  feed_url: string;
  description: string | null;
  fetch_interval_minutes: number;
  consecutive_failures: number;
  last_fetched_at: string | null;
}

export interface ArticleSummary {
  id: string;
  title: string;
  url: string;
  published_at: string | null;
  language: string | null;
  trend_score: number;
  status: string;
  source: Source | null;
  summary: string | null;
  tags: Tag[];
}

export interface ArticleDetail extends ArticleSummary {
  clean_content: string | null;
}

// ─── Recommendation ───────────────────────────────────────────────────────────

export interface RecommendationItem {
  article: ArticleSummary;
  score: number;
  reason: string;
  score_breakdown: {
    relevance: number;
    freshness: number;
    trend: number;
    source_quality: number;
    personalization: number;
  };
  user_feedback: 'like' | 'dislike' | null;
  is_bookmarked: boolean;
}

// ─── Bookmark ────────────────────────────────────────────────────────────────

export interface BookmarkItem {
  id: string;
  article_id: string;
  created_at: string;
  article: ArticleSummary;
}

// ─── Admin ───────────────────────────────────────────────────────────────────

export interface AdminStats {
  sources: {
    total: number;
    active: number;
    degraded: number;
    disabled: number;
  };
  articles: {
    total: number;
    processed: number;
    pending: number;
    failed: number;
  };
  ingestion_jobs: {
    last_24h_completed: number;
    last_24h_failed: number;
  };
  users: {
    total: number;
    active_last_7d: number;
  };
}

export interface IngestionJob {
  id: string;
  source_id: string;
  source_name: string;
  status: string;
  articles_found: number;
  articles_new: number;
  articles_skipped: number;
  error_message: string | null;
  started_at: string;
  completed_at: string | null;
}

// ─── Inputs ──────────────────────────────────────────────────────────────────

export interface RegisterInput {
  email: string;
  password: string;
  display_name: string;
}

export interface LoginInput {
  email: string;
  password: string;
}

export interface UpdateUserInput {
  display_name?: string;
  preferred_language?: 'ja' | 'en';
}

export interface UpdateInterestsInput {
  interests: Array<{ tag_name: string; weight: number }>;
}

export interface FeedbackInput {
  article_id: string;
  feedback_type: 'like' | 'dislike';
}

export interface ArticleListParams {
  page?: number;
  per_page?: number;
  language?: string;
  tag?: string;
}

export interface TrendingParams {
  period?: '24h' | '7d';
  page?: number;
  per_page?: number;
  language?: string;
}

export interface PaginationParams {
  page?: number;
  per_page?: number;
}

export type SearchParams = ArticleListParams;
