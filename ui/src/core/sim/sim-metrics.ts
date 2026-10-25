import type { CacheSnapshot, HTTPSnapshot } from '../types';

/**
 * Cache metrics tracker.
 * Direct TypeScript port of `internal/metrics/metrics.go` CacheMetrics.
 */
export class SimCacheMetrics {
  private hits: number = 0;
  private misses: number = 0;
  private sets: number = 0;
  private deletes: number = 0;
  private evictions: number = 0;
  private expired: number = 0;

  public incHits(): void {
    this.hits++;
  }

  public incMisses(): void {
    this.misses++;
  }

  public incSets(): void {
    this.sets++;
  }

  public incDeletes(): void {
    this.deletes++;
  }

  public incEvictions(): void {
    this.evictions++;
  }

  public incExpired(): void {
    this.expired++;
  }

  public reset(): void {
    this.hits = 0;
    this.misses = 0;
    this.sets = 0;
    this.deletes = 0;
    this.evictions = 0;
    this.expired = 0;
  }

  public getHits(): number {
    return this.hits;
  }

  public getMisses(): number {
    return this.misses;
  }

  public getSets(): number {
    return this.sets;
  }

  public getDeletes(): number {
    return this.deletes;
  }

  public getEvictions(): number {
    return this.evictions;
  }

  public getExpired(): number {
    return this.expired;
  }

  public hitRate(): number {
    const total = this.hits + this.misses;
    if (total === 0) return 0;
    return this.hits / total;
  }

  public snapshot(): CacheSnapshot {
    return {
      hits: this.hits,
      misses: this.misses,
      hitRate: this.hitRate(),
      sets: this.sets,
      deletes: this.deletes,
      evictions: this.evictions,
      expired: this.expired,
    };
  }
}

/**
 * HTTP request telemetry tracker.
 * Direct TypeScript port of `internal/metrics/metrics.go` HTTPMetrics.
 */
export class SimHTTPMetrics {
  private requests: number = 0;
  private successes: number = 0;
  private errors4xx: number = 0;
  private errors5xx: number = 0;
  private totalLatencyMs: number = 0;

  public recordRequest(statusCode: number, latencyMs: number): void {
    this.requests++;
    this.totalLatencyMs += latencyMs;

    if (statusCode >= 200 && statusCode < 300) {
      this.successes++;
    } else if (statusCode >= 400 && statusCode < 500) {
      this.errors4xx++;
    } else if (statusCode >= 500) {
      this.errors5xx++;
    }
  }

  public reset(): void {
    this.requests = 0;
    this.successes = 0;
    this.errors4xx = 0;
    this.errors5xx = 0;
    this.totalLatencyMs = 0;
  }

  public avgLatencyMs(): number {
    if (this.requests === 0) return 0;
    return this.totalLatencyMs / this.requests;
  }

  public snapshot(): HTTPSnapshot {
    return {
      requests: this.requests,
      successes: this.successes,
      errors4xx: this.errors4xx,
      errors5xx: this.errors5xx,
      avgLatencyMs: Number(this.avgLatencyMs().toFixed(2)),
    };
  }
}
