import type { CacheSnapshot, HTTPSnapshot, NodeInfo } from '../types';

export interface ApiClientConfig {
  baseUrl?: string;
  timeoutMs?: number;
}

export interface ApiResponse<T = unknown> {
  ok: boolean;
  status: number;
  data?: T;
  error?: string;
  latencyMs: number;
}

/**
 * ApiClient handles HTTP communication with the live Go distributed cache cluster.
 * Communicates with Router (:9000) and Cache Nodes (:8001+:800N).
 */
export class ApiClient {
  private baseUrl: string;
  private timeoutMs: number;

  constructor(config: ApiClientConfig = {}) {
    // In Vite dev, '/api' proxies to http://127.0.0.1:9000
    this.baseUrl = config.baseUrl ?? '/api';
    this.timeoutMs = config.timeoutMs ?? 5000;
  }

  public setBaseUrl(url: string): void {
    this.baseUrl = url.replace(/\/+$/, '');
  }

  public getBaseUrl(): string {
    return this.baseUrl;
  }

  private async request<T = unknown>(
    path: string,
    options: RequestInit = {}
  ): Promise<ApiResponse<T>> {
    const start = performance.now();
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.timeoutMs);

    const url = `${this.baseUrl}${path.startsWith('/') ? path : `/${path}`}`;

    try {
      const response = await fetch(url, {
        ...options,
        signal: controller.signal,
        headers: {
          'Content-Type': 'application/json',
          ...(options.headers || {}),
        },
      });

      clearTimeout(timeoutId);
      const latencyMs = Number((performance.now() - start).toFixed(2));

      let body: unknown = null;
      const contentType = response.headers.get('content-type') || '';
      if (contentType.includes('application/json')) {
        body = await response.json().catch(() => null);
      } else {
        const text = await response.text().catch(() => '');
        body = text ? { message: text } : null;
      }

      if (!response.ok) {
        const parsedError = (body as { error?: string } | null)?.error;
        return {
          ok: false,
          status: response.status,
          error: parsedError || `HTTP ${response.status}: ${response.statusText}`,
          latencyMs,
        };
      }

      return {
        ok: true,
        status: response.status,
        data: body as T,
        latencyMs,
      };
    } catch (err: unknown) {
      clearTimeout(timeoutId);
      const latencyMs = Number((performance.now() - start).toFixed(2));
      const isTimeout = err instanceof Error && err.name === 'AbortError';
      const errorMessage = err instanceof Error ? err.message : 'Network connection failed';

      return {
        ok: false,
        status: isTimeout ? 504 : 0,
        error: isTimeout ? 'Request timed out' : errorMessage,
        latencyMs,
      };
    }
  }

  /**
   * Ping Router health check: GET /health
   */
  public async healthCheck(): Promise<boolean> {
    try {
      const res = await this.request<{ status: string }>('/health', { method: 'GET' });
      return res.ok && res.data?.status === 'ok';
    } catch {
      return false;
    }
  }

  /**
   * Query registered cluster nodes: GET /nodes
   */
  public async getNodes(): Promise<ApiResponse<{ nodes: { id: string; address: string }[]; replicas: number }>> {
    return this.request<{ nodes: { id: string; address: string }[]; replicas: number }>('/nodes', { method: 'GET' });
  }

  /**
   * Query router HTTP metrics: GET /metrics
   */
  public async getRouterMetrics(): Promise<ApiResponse<HTTPSnapshot>> {
    return this.request<HTTPSnapshot>('/metrics', { method: 'GET' });
  }

  /**
   * Cache GET: GET /cache/:key
   */
  public async getCache(key: string): Promise<ApiResponse<{ key: string; value: string }>> {
    const encodedKey = encodeURIComponent(key);
    return this.request<{ key: string; value: string }>(`/cache/${encodedKey}`, { method: 'GET' });
  }

  /**
   * Cache PUT: PUT /cache/:key
   */
  public async setCache(
    key: string,
    value: string,
    ttl?: string
  ): Promise<ApiResponse<{ message: string }>> {
    const encodedKey = encodeURIComponent(key);
    const body: Record<string, string> = { value };
    if (ttl) {
      body.ttl = ttl;
    }

    return this.request<{ message: string }>(`/cache/${encodedKey}`, {
      method: 'PUT',
      body: JSON.stringify(body),
    });
  }

  /**
   * Cache DELETE: DELETE /cache/:key
   */
  public async deleteCache(key: string): Promise<ApiResponse<{ message: string }>> {
    const encodedKey = encodeURIComponent(key);
    return this.request<{ message: string }>(`/cache/${encodedKey}`, { method: 'DELETE' });
  }

  /**
   * Direct node metrics: GET {nodeAddress}/metrics
   */
  public async getNodeMetrics(nodeAddress: string): Promise<ApiResponse<CacheSnapshot>> {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 2000);
    const start = performance.now();

    try {
      const res = await fetch(`${nodeAddress}/metrics`, { signal: controller.signal });
      clearTimeout(timeoutId);
      const latencyMs = Number((performance.now() - start).toFixed(2));
      const data = await res.json();
      return { ok: res.ok, status: res.status, data, latencyMs };
    } catch (err: unknown) {
      clearTimeout(timeoutId);
      const errorMessage = err instanceof Error ? err.message : 'Failed to connect to node';
      return {
        ok: false,
        status: 0,
        error: errorMessage,
        latencyMs: Number((performance.now() - start).toFixed(2)),
      };
    }
  }

  /**
   * Direct node info: GET {nodeAddress}/node
   */
  public async getNodeInfo(nodeAddress: string): Promise<ApiResponse<NodeInfo>> {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 2000);
    const start = performance.now();

    try {
      const res = await fetch(`${nodeAddress}/node`, { signal: controller.signal });
      clearTimeout(timeoutId);
      const latencyMs = Number((performance.now() - start).toFixed(2));
      const data = await res.json();
      return { ok: res.ok, status: res.status, data, latencyMs };
    } catch (err: unknown) {
      clearTimeout(timeoutId);
      const errorMessage = err instanceof Error ? err.message : 'Failed to connect to node';
      return {
        ok: false,
        status: 0,
        error: errorMessage,
        latencyMs: Number((performance.now() - start).toFixed(2)),
      };
    }
  }
}

// Export singleton instance configured for local Vite proxy
export const apiClient = new ApiClient();
