import { LRUEngine } from './lru-engine';
import { LFUEngine } from './lfu-engine';
import { TwoQEngine } from './twoq-engine';
import { SimCacheMetrics } from './sim-metrics';
import type { CacheEntry, CacheSnapshot, EvictionPolicy, NodeInfo } from '../types';

export interface SimNodeConfig {
  id: string;
  address: string;
  capacity: number;
  policy: EvictionPolicy;
}

/**
 * SimNode encapsulates an independent cache server node.
 * Mirrors `internal/node/node.go` and `internal/server/server.go`.
 */
export class SimNode {
  public readonly id: string;
  public readonly address: string;
  private readonly capacity: number;
  private readonly policyType: EvictionPolicy;
  private engine: LRUEngine | LFUEngine | TwoQEngine;
  private metrics: SimCacheMetrics;

  constructor(config: SimNodeConfig) {
    this.id = config.id;
    this.address = config.address;
    this.capacity = config.capacity > 0 ? config.capacity : 100;
    this.policyType = config.policy;
    this.metrics = new SimCacheMetrics();

    switch (this.policyType) {
      case 'LFU':
        this.engine = new LFUEngine(this.capacity);
        break;
      case '2Q':
        this.engine = new TwoQEngine(this.capacity);
        break;
      case 'LRU':
      default:
        this.engine = new LRUEngine(this.capacity);
        break;
    }
  }

  public getPolicy(): EvictionPolicy {
    return this.policyType;
  }

  public getCapacity(): number {
    return this.capacity;
  }

  public size(): number {
    return this.engine.size();
  }

  public getMetrics(): CacheSnapshot {
    return this.metrics.snapshot();
  }

  public getInfo(): NodeInfo {
    return {
      id: this.id,
      address: this.address,
      capacity: this.capacity,
      policy: this.policyType,
      size: this.size(),
      health: 'healthy',
    };
  }

  public set(
    key: string,
    value: string,
    ttlMs?: number | null
  ): { evicted: boolean; evictedKey?: string } {
    const res = this.engine.set(key, value, ttlMs);
    this.metrics.incSets();
    if (res.evicted) {
      this.metrics.incEvictions();
    }
    return res;
  }

  public get(key: string): { value: string | null; found: boolean; expired: boolean } {
    const res = this.engine.get(key);
    if (res.expired) {
      this.metrics.incExpired();
      this.metrics.incMisses();
      return { value: null, found: false, expired: true };
    }
    if (!res.found) {
      this.metrics.incMisses();
      return { value: null, found: false, expired: false };
    }
    this.metrics.incHits();
    return res;
  }

  public delete(key: string): { deleted: boolean; expired: boolean } {
    const res = this.engine.delete(key);
    if (res.expired) {
      this.metrics.incExpired();
    }
    if (res.deleted) {
      this.metrics.incDeletes();
    }
    return res;
  }

  public clear(): void {
    this.engine.clear();
    this.metrics.reset();
  }

  /**
   * Introspection: returns all keys stored on this node.
   * Useful for the node drawer in Playground / Cluster inspector.
   */
  public getEntries(): CacheEntry[] {
    if (this.engine instanceof LRUEngine) {
      return this.engine.getOrderedEntries();
    } else if (this.engine instanceof LFUEngine) {
      return this.engine.getAllEntries();
    } else if (this.engine instanceof TwoQEngine) {
      return [...this.engine.getA1Entries(), ...this.engine.getAmEntries()];
    }
    return [];
  }

  public getRawEngine(): LRUEngine | LFUEngine | TwoQEngine {
    return this.engine;
  }
}
