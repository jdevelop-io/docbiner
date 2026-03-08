import { DocbinerError } from './errors';
import type {
  ConvertRequest,
  ConvertAsyncRequest,
  Job,
  ListJobsParams,
  PaginatedResponse,
  Template,
  CreateTemplateData,
  UpdateTemplateData,
  MergeRequest,
  UsageResponse,
  MonthlyUsage,
  ErrorResponseBody,
} from './types';

/** Configuration options for the Docbiner client. */
export interface DocbinerOptions {
  /** Your Docbiner API key. */
  apiKey: string;
  /** Override the base URL (defaults to https://api.docbiner.com). */
  baseURL?: string;
  /** Maximum number of retry attempts on 5xx errors (defaults to 3). */
  maxRetries?: number;
  /** AbortSignal for request cancellation. */
  signal?: AbortSignal;
}

const DEFAULT_BASE_URL = 'https://api.docbiner.com';
const DEFAULT_MAX_RETRIES = 3;
const RETRY_BASE_DELAY_MS = 1000;

/**
 * Official Docbiner client for Node.js.
 *
 * @example
 * ```ts
 * const docbiner = new Docbiner({ apiKey: 'db_live_...' });
 * const pdf = await docbiner.convert({ source: '<h1>Hello</h1>' });
 * ```
 */
export class Docbiner {
  private readonly apiKey: string;
  private readonly baseURL: string;
  private readonly maxRetries: number;
  private readonly signal?: AbortSignal;

  constructor(options: DocbinerOptions) {
    if (!options.apiKey) {
      throw new Error('apiKey is required');
    }
    this.apiKey = options.apiKey;
    this.baseURL = (options.baseURL || DEFAULT_BASE_URL).replace(/\/+$/, '');
    this.maxRetries = options.maxRetries ?? DEFAULT_MAX_RETRIES;
    this.signal = options.signal;
  }

  // ---------------------------------------------------------------------------
  // Convert
  // ---------------------------------------------------------------------------

  /**
   * Convert HTML/URL to PDF or image synchronously.
   * Returns the file content as a Buffer.
   */
  async convert(request: ConvertRequest): Promise<Buffer> {
    return this.requestBinary('/v1/convert', {
      method: 'POST',
      body: JSON.stringify(request),
    });
  }

  /**
   * Convert HTML/URL to PDF or image asynchronously.
   * Returns the created job.
   */
  async convertAsync(request: ConvertAsyncRequest): Promise<Job> {
    return this.request<Job>('/v1/convert/async', {
      method: 'POST',
      body: JSON.stringify(request),
    });
  }

  // ---------------------------------------------------------------------------
  // Jobs
  // ---------------------------------------------------------------------------

  /** Job-related operations. */
  readonly jobs = {
    /**
     * Get a job by ID.
     */
    get: (id: string): Promise<Job> => {
      return this.request<Job>(`/v1/jobs/${encodeURIComponent(id)}`);
    },

    /**
     * List jobs with optional pagination and filters.
     */
    list: (params?: ListJobsParams): Promise<PaginatedResponse<Job>> => {
      const query = new URLSearchParams();
      if (params?.page != null) query.set('page', String(params.page));
      if (params?.per_page != null) query.set('per_page', String(params.per_page));
      if (params?.status) query.set('status', params.status);
      if (params?.format) query.set('format', params.format);

      const qs = query.toString();
      const path = qs ? `/v1/jobs?${qs}` : '/v1/jobs';
      return this.request<PaginatedResponse<Job>>(path);
    },

    /**
     * Download the result file of a completed job.
     * Returns the file content as a Buffer.
     */
    download: (id: string): Promise<Buffer> => {
      return this.requestBinary(`/v1/jobs/${encodeURIComponent(id)}/download`);
    },

    /**
     * Delete a job and its associated result file.
     */
    delete: async (id: string): Promise<void> => {
      await this.request<void>(`/v1/jobs/${encodeURIComponent(id)}`, {
        method: 'DELETE',
      });
    },
  };

  // ---------------------------------------------------------------------------
  // Templates
  // ---------------------------------------------------------------------------

  /** Template-related operations. */
  readonly templates = {
    /**
     * Create a new template.
     */
    create: (data: CreateTemplateData): Promise<Template> => {
      return this.request<Template>('/v1/templates', {
        method: 'POST',
        body: JSON.stringify(data),
      });
    },

    /**
     * Get a template by ID.
     */
    get: (id: string): Promise<Template> => {
      return this.request<Template>(`/v1/templates/${encodeURIComponent(id)}`);
    },

    /**
     * List all templates for the organization.
     */
    list: (): Promise<Template[]> => {
      return this.request<Template[]>('/v1/templates');
    },

    /**
     * Update an existing template.
     */
    update: (id: string, data: UpdateTemplateData): Promise<Template> => {
      return this.request<Template>(`/v1/templates/${encodeURIComponent(id)}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      });
    },

    /**
     * Delete a template.
     */
    delete: async (id: string): Promise<void> => {
      await this.request<void>(`/v1/templates/${encodeURIComponent(id)}`, {
        method: 'DELETE',
      });
    },

    /**
     * Preview a rendered template. Returns the rendered HTML string.
     */
    preview: (id: string, data?: Record<string, unknown>): Promise<string> => {
      return this.request<{ html: string }>(`/v1/templates/${encodeURIComponent(id)}/preview`, {
        method: 'POST',
        body: JSON.stringify({ data: data ?? {} }),
      }).then((res) => res.html);
    },
  };

  // ---------------------------------------------------------------------------
  // Merge
  // ---------------------------------------------------------------------------

  /**
   * Merge multiple HTML/URL sources into a single PDF.
   * Returns the merged PDF as a Buffer.
   */
  async merge(request: MergeRequest): Promise<Buffer> {
    return this.requestBinary('/v1/merge', {
      method: 'POST',
      body: JSON.stringify(request),
    });
  }

  // ---------------------------------------------------------------------------
  // Usage
  // ---------------------------------------------------------------------------

  /**
   * Get current month usage and quota status.
   */
  async usage(): Promise<UsageResponse> {
    return this.request<UsageResponse>('/v1/usage');
  }

  /**
   * Get usage history for the last 12 months.
   */
  async usageHistory(): Promise<MonthlyUsage[]> {
    return this.request<MonthlyUsage[]>('/v1/usage/history');
  }

  // ---------------------------------------------------------------------------
  // Private helpers
  // ---------------------------------------------------------------------------

  /**
   * Make a JSON API request with retry logic.
   */
  private async request<T>(path: string, init?: RequestInit): Promise<T> {
    const response = await this.fetchWithRetry(path, init);

    // 204 No Content — nothing to parse.
    if (response.status === 204) {
      return undefined as unknown as T;
    }

    const body = await response.json();

    if (!response.ok) {
      const err = body as ErrorResponseBody;
      throw new DocbinerError(
        err.message || response.statusText,
        response.status,
        err.error,
      );
    }

    return body as T;
  }

  /**
   * Make a binary API request with retry logic.
   * Follows redirects and returns the file content as a Buffer.
   */
  private async requestBinary(path: string, init?: RequestInit): Promise<Buffer> {
    const response = await this.fetchWithRetry(path, init);

    if (!response.ok) {
      let errBody: ErrorResponseBody | undefined;
      try {
        errBody = (await response.json()) as ErrorResponseBody;
      } catch {
        // Response may not be JSON.
      }
      throw new DocbinerError(
        errBody?.message || response.statusText,
        response.status,
        errBody?.error,
      );
    }

    const arrayBuffer = await response.arrayBuffer();
    return Buffer.from(arrayBuffer);
  }

  /**
   * Execute a fetch with exponential backoff retry on 5xx errors.
   */
  private async fetchWithRetry(path: string, init?: RequestInit): Promise<Response> {
    const url = `${this.baseURL}${path}`;

    const headers: Record<string, string> = {
      'Authorization': `Bearer ${this.apiKey}`,
      'User-Agent': '@docbiner/sdk-node/0.1.0',
    };

    // Set Content-Type for requests with a body.
    if (init?.body) {
      headers['Content-Type'] = 'application/json';
    }

    let lastError: Error | undefined;

    for (let attempt = 0; attempt <= this.maxRetries; attempt++) {
      try {
        const response = await fetch(url, {
          ...init,
          headers: {
            ...headers,
            ...(init?.headers as Record<string, string> | undefined),
          },
          signal: this.signal,
          redirect: 'follow',
        });

        // Retry on 5xx server errors (except on the last attempt).
        if (response.status >= 500 && attempt < this.maxRetries) {
          lastError = new DocbinerError(
            `Server error: ${response.status}`,
            response.status,
          );
          await this.sleep(RETRY_BASE_DELAY_MS * Math.pow(2, attempt));
          continue;
        }

        return response;
      } catch (err) {
        // Don't retry on abort.
        if (err instanceof Error && err.name === 'AbortError') {
          throw err;
        }

        lastError = err instanceof Error ? err : new Error(String(err));

        // Retry network errors.
        if (attempt < this.maxRetries) {
          await this.sleep(RETRY_BASE_DELAY_MS * Math.pow(2, attempt));
          continue;
        }
      }
    }

    throw lastError ?? new Error('Request failed after retries');
  }

  /**
   * Sleep for the given number of milliseconds.
   */
  private sleep(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }
}
