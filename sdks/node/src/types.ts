// --- Convert ---

/** Options for PDF/image conversion. */
export interface ConvertOptions {
  /** Paper size: A4, Letter, etc. */
  page_size?: string;
  /** Landscape orientation. */
  landscape?: boolean;
  /** Top margin, e.g. "20mm". */
  margin_top?: string;
  /** Right margin, e.g. "20mm". */
  margin_right?: string;
  /** Bottom margin, e.g. "20mm". */
  margin_bottom?: string;
  /** Left margin, e.g. "20mm". */
  margin_left?: string;
  /** Custom header HTML. */
  header_html?: string;
  /** Custom footer HTML. */
  footer_html?: string;
  /** Additional CSS injected into the page. */
  css?: string;
  /** Additional JS executed in the page. */
  js?: string;
  /** CSS selector to wait for before conversion. */
  wait_for?: string;
  /** Delay in milliseconds before conversion. */
  delay_ms?: number;
  /** Page scale factor. */
  scale?: number;
  /** Print background graphics. */
  print_background?: boolean;
  /** Viewport width (screenshots). */
  width?: number;
  /** Viewport height (screenshots). */
  height?: number;
  /** Image quality 0-100 (jpeg/webp only). */
  quality?: number;
  /** Capture full page screenshot. */
  full_page?: boolean;
  /** PDF encryption options. */
  encrypt?: EncryptOptions;
}

/** PDF encryption options. */
export interface EncryptOptions {
  user_password?: string;
  owner_password?: string;
}

/** Output format for conversion. */
export type OutputFormat = 'pdf' | 'png' | 'jpeg' | 'webp';

/** Synchronous convert request payload. */
export interface ConvertRequest {
  /** HTML string or URL to convert. */
  source: string;
  /** Output format (defaults to "pdf"). */
  format?: OutputFormat;
  /** Conversion options. */
  options?: ConvertOptions;
}

/** Delivery configuration for async conversions. */
export interface DeliveryConfig {
  /** Delivery method: "webhook" or "s3". */
  method: 'webhook' | 's3';
  /** Method-specific configuration. */
  config?: Record<string, unknown>;
}

/** Asynchronous convert request payload. */
export interface ConvertAsyncRequest {
  /** HTML string or URL to convert. */
  source: string;
  /** Output format (defaults to "pdf"). */
  format?: OutputFormat;
  /** Conversion options. */
  options?: ConvertOptions;
  /** How to deliver the result. */
  delivery?: DeliveryConfig;
}

// --- Jobs ---

/** Job status. */
export type JobStatus = 'pending' | 'processing' | 'completed' | 'failed';

/** Input type. */
export type InputType = 'url' | 'html' | 'template';

/** Delivery method. */
export type DeliveryMethod = 'sync' | 'webhook' | 's3';

/** A conversion job. */
export interface Job {
  id: string;
  org_id: string;
  api_key_id: string;
  status: JobStatus;
  input_type: InputType;
  input_source: string;
  input_data?: unknown;
  output_format: OutputFormat;
  options: unknown;
  delivery_method: DeliveryMethod;
  delivery_config?: unknown;
  result_url?: string;
  result_size?: number;
  pages_count?: number;
  duration_ms?: number;
  error_message?: string;
  is_test: boolean;
  created_at: string;
  completed_at?: string;
}

/** Parameters for listing jobs. */
export interface ListJobsParams {
  /** Page number (starting from 1). */
  page?: number;
  /** Results per page (max 100). */
  per_page?: number;
  /** Filter by status. */
  status?: JobStatus;
  /** Filter by output format. */
  format?: OutputFormat;
}

// --- Templates ---

/** Template engine type. */
export type TemplateEngine = 'handlebars' | 'liquid';

/** A stored template. */
export interface Template {
  id: string;
  org_id: string;
  created_by: string;
  name: string;
  engine: TemplateEngine;
  html_content: string;
  css_content?: string;
  sample_data?: unknown;
  created_at: string;
  updated_at: string;
}

/** Payload for creating a template. */
export interface CreateTemplateData {
  name: string;
  engine: TemplateEngine;
  html_content: string;
  css_content?: string;
  sample_data?: Record<string, unknown>;
}

/** Payload for updating a template. */
export interface UpdateTemplateData {
  name?: string;
  engine?: TemplateEngine;
  html_content?: string;
  css_content?: string;
  sample_data?: Record<string, unknown>;
}

// --- Merge ---

/** A source entry in a merge request. */
export interface MergeSource {
  source: string;
}

/** Merge request payload. */
export interface MergeRequest {
  sources: MergeSource[];
  options?: ConvertOptions;
}

// --- Plans ---

/** A billing plan. */
export interface Plan {
  id: string;
  name: string;
  monthly_quota: number;
  overage_price: number;
  price_monthly: number;
  price_yearly: number;
  max_file_size: number;
  timeout_seconds: number;
  features: unknown;
  active: boolean;
}

// --- API Keys ---

/** API key environment. */
export type ApiKeyEnvironment = 'live' | 'test';

/** An API key (without the raw secret). */
export interface ApiKey {
  id: string;
  org_id: string;
  created_by: string;
  key_prefix: string;
  name: string;
  environment: ApiKeyEnvironment;
  last_used_at?: string;
  expires_at?: string;
  created_at: string;
}

// --- Usage ---

/** Monthly usage data. */
export interface MonthlyUsage {
  month: string;
  conversions: number;
  test_conversions: number;
  overage_amount: number;
}

/** Quota status for the current billing period. */
export interface QuotaStatus {
  allowed: boolean;
  used: number;
  limit: number;
  remaining: number;
}

/** Response for GET /v1/usage. */
export interface UsageResponse {
  month: string;
  conversions: number;
  test_conversions: number;
  quota: QuotaStatus;
}

// --- Pagination ---

/** Pagination metadata. */
export interface Pagination {
  page: number;
  per_page: number;
  total: number;
  total_pages: number;
}

/** A paginated API response. */
export interface PaginatedResponse<T> {
  data: T[];
  pagination: Pagination;
}

// --- Error response ---

/** API error response body. */
export interface ErrorResponseBody {
  error: string;
  message: string;
}
