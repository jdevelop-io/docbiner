export interface User {
  id: string;
  email: string;
  username: string;
  display_name: string;
  avatar_url?: string;
}

export interface Organization {
  id: string;
  name: string;
  slug: string;
  plan_id: string;
}

export interface Job {
  id: string;
  org_id: string;
  status: 'pending' | 'processing' | 'completed' | 'failed';
  input_type: 'url' | 'html' | 'template';
  output_format: 'pdf' | 'png' | 'jpeg' | 'webp';
  delivery_method: 'sync' | 'webhook' | 's3';
  result_url?: string;
  result_size?: number;
  pages_count?: number;
  duration_ms?: number;
  error_message?: string;
  is_test: boolean;
  created_at: string;
  completed_at?: string;
}

export interface Template {
  id: string;
  org_id: string;
  name: string;
  engine: 'handlebars' | 'liquid';
  html_content: string;
  css_content?: string;
  sample_data?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface ApiKey {
  id: string;
  name: string;
  key_prefix: string;
  environment: 'live' | 'test';
  last_used_at?: string;
  created_at: string;
}

export type MemberRole = 'owner' | 'admin' | 'member';

export interface Member {
  id: string;
  user_id: string;
  display_name: string;
  email: string;
  role: MemberRole;
  avatar_url?: string;
  joined_at: string;
}

export interface Plan {
  id: string;
  name: string;
  monthly_quota: number;
  overage_price: number;
  price_monthly: number;
  price_yearly: number;
  max_file_size: number;
  features: Record<string, boolean>;
}

export interface MonthlyUsage {
  month: string;
  conversions: number;
  test_conversions: number;
  overage_amount: number;
}

export interface QuotaStatus {
  used: number;
  limit: number;
  remaining: number;
}

export interface UsageSummary {
  quota: QuotaStatus;
  current_month: MonthlyUsage;
  avg_duration_ms: number;
  success_rate: number;
}

export interface Pagination {
  page: number;
  per_page: number;
  total: number;
  total_pages: number;
}

export interface PaginatedResponse<T> {
  data: T[];
  pagination: Pagination;
}
