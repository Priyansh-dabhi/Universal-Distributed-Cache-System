import { ConsistentHashRing } from './hash-ring';
import { SimNode } from './sim-node';
import { SimHTTPMetrics } from './sim-metrics';
import { NODE_COLORS } from '../constants';
import type {
  CacheSnapshot,
  EvictionPolicy,
  HTTPSnapshot,
  NodeInfo,
  RingNode,
  RouteResult,
} from '../types';

export interface RouterResponse<T = unknown> {
  status: number;
  data?: T;
  error?: string;
  targetNodeId?: string;
  targetAddress?: string;
  latencyMs: number;
  isHit?: boolean;
}

/**
 * SimRouter acts as a simulated distributed reverse-proxy router.
 * Direct in-browser TypeScript implementation of `internal/router/router.go`.
 */
export class SimRouter {
  private ring: ConsistentHashRing;
  private nodes: Map<string, SimNode> = new Map();
  private httpMetrics: SimHTTPMetrics;

  constructor(replicas = 50) {
    this.ring = new ConsistentHashRing(replicas);
    this.httpMetrics = new SimHTTPMetrics();
    this.initializeDefaultCluster();
  }

  private initializeDefaultCluster(): void {
    const defaultNodes: { id: string; port: number; capacity: number; policy: EvictionPolicy }[] = [
      { id: 'node-1', port: 8001, capacity: 50, policy: 'LRU' },
      { id: 'node-2', port: 8002, capacity: 50, policy: 'LFU' },
      { id: 'node-3', port: 8003, capacity: 50, policy: '2Q' },
    ];

    for (const d of defaultNodes) {
      const address = `http://localhost:${d.port}`;
      const color = NODE_COLORS[d.id]?.bg || '#06b6d4';
      const simNode = new SimNode({
        id: d.id,
        address,
        capacity: d.capacity,
        policy: d.policy,
      });

      this.nodes.set(d.id, simNode);
      this.ring.addNode({ id: d.id, address, color });
    }
  }

  public getRing(): ConsistentHashRing {
    return this.ring;
  }

  public getHTTPMetrics(): HTTPSnapshot {
    return this.httpMetrics.snapshot();
  }

  public getNodes(): NodeInfo[] {
    return Array.from(this.nodes.values()).map((n) => n.getInfo());
  }

  public getRingNodes(): RingNode[] {
    return this.ring.getNodes();
  }

  public getNode(nodeId: string): SimNode | undefined {
    return this.nodes.get(nodeId);
  }

  public addNode(
    config: { id: string; address?: string; capacity?: number; policy?: EvictionPolicy },
    color?: string
  ): void {
    if (this.nodes.has(config.id)) {
      throw new Error(`Node ${config.id} already exists`);
    }

    const port = 8000 + this.nodes.size + 1;
    const address = config.address || `http://localhost:${port}`;
    const capacity = config.capacity || 50;
    const policy = config.policy || 'LRU';

    const simNode = new SimNode({
      id: config.id,
      address,
      capacity,
      policy,
    });

    this.nodes.set(config.id, simNode);
    this.ring.addNode({
      id: config.id,
      address,
      color: color || '#10b981',
    });
  }

  public removeNode(nodeId: string): void {
    if (!this.nodes.has(nodeId)) {
      throw new Error(`Node ${nodeId} not found`);
    }
    this.nodes.delete(nodeId);
    this.ring.removeNode(nodeId);
  }

  public route(key: string): RouteResult {
    return this.ring.route(key);
  }

  /**
   * Simulated HTTP GET /cache/:key
   */
  public async get(key: string): Promise<RouterResponse<{ key: string; value: string }>> {
    const start = performance.now();

    if (!key || key.trim() === '') {
      const latencyMs = Number((performance.now() - start).toFixed(2));
      this.httpMetrics.recordRequest(400, latencyMs);
      return { status: 400, error: 'key is required', latencyMs };
    }

    let routeRes: RouteResult;
    try {
      routeRes = this.route(key);
    } catch (err: unknown) {
      const latencyMs = Number((performance.now() - start).toFixed(2));
      const errorMsg = err instanceof Error ? err.message : 'Routing failed';
      this.httpMetrics.recordRequest(503, latencyMs);
      return { status: 503, error: errorMsg, latencyMs };
    }

    const node = this.nodes.get(routeRes.targetNodeId);
    if (!node) {
      const latencyMs = Number((performance.now() - start).toFixed(2));
      this.httpMetrics.recordRequest(502, latencyMs);
      return { status: 502, error: 'cache node unavailable', latencyMs };
    }

    // Small simulated network jitter (0.3ms - 1.2ms)
    await new Promise((r) => setTimeout(r, Math.random() * 0.9 + 0.3));

    const cacheRes = node.get(key);
    const latencyMs = Number((performance.now() - start).toFixed(2));

    if (!cacheRes.found) {
      this.httpMetrics.recordRequest(404, latencyMs);
      return {
        status: 404,
        error: cacheRes.expired ? 'key expired' : 'key not found',
        targetNodeId: node.id,
        targetAddress: node.address,
        latencyMs,
        isHit: false,
      };
    }

    this.httpMetrics.recordRequest(200, latencyMs);
    return {
      status: 200,
      data: { key, value: cacheRes.value! },
      targetNodeId: node.id,
      targetAddress: node.address,
      latencyMs,
      isHit: true,
    };
  }

  /**
   * Simulated HTTP PUT /cache/:key
   */
  public async set(
    key: string,
    value: string,
    ttlMs?: number | null
  ): Promise<RouterResponse<{ message: string; evicted?: boolean }>> {
    const start = performance.now();

    if (!key || key.trim() === '') {
      const latencyMs = Number((performance.now() - start).toFixed(2));
      this.httpMetrics.recordRequest(400, latencyMs);
      return { status: 400, error: 'key is required', latencyMs };
    }

    let routeRes: RouteResult;
    try {
      routeRes = this.route(key);
    } catch (err: unknown) {
      const latencyMs = Number((performance.now() - start).toFixed(2));
      const errorMsg = err instanceof Error ? err.message : 'Routing failed';
      this.httpMetrics.recordRequest(503, latencyMs);
      return { status: 503, error: errorMsg, latencyMs };
    }

    const node = this.nodes.get(routeRes.targetNodeId);
    if (!node) {
      const latencyMs = Number((performance.now() - start).toFixed(2));
      this.httpMetrics.recordRequest(502, latencyMs);
      return { status: 502, error: 'cache node unavailable', latencyMs };
    }

    await new Promise((r) => setTimeout(r, Math.random() * 0.9 + 0.3));

    const setRes = node.set(key, value, ttlMs);
    const latencyMs = Number((performance.now() - start).toFixed(2));

    this.httpMetrics.recordRequest(200, latencyMs);
    return {
      status: 200,
      data: { message: 'stored', evicted: setRes.evicted },
      targetNodeId: node.id,
      targetAddress: node.address,
      latencyMs,
    };
  }

  /**
   * Simulated HTTP DELETE /cache/:key
   */
  public async delete(key: string): Promise<RouterResponse<{ message: string }>> {
    const start = performance.now();

    if (!key || key.trim() === '') {
      const latencyMs = Number((performance.now() - start).toFixed(2));
      this.httpMetrics.recordRequest(400, latencyMs);
      return { status: 400, error: 'key is required', latencyMs };
    }

    let routeRes: RouteResult;
    try {
      routeRes = this.route(key);
    } catch (err: unknown) {
      const latencyMs = Number((performance.now() - start).toFixed(2));
      const errorMsg = err instanceof Error ? err.message : 'Routing failed';
      this.httpMetrics.recordRequest(503, latencyMs);
      return { status: 503, error: errorMsg, latencyMs };
    }

    const node = this.nodes.get(routeRes.targetNodeId);
    if (!node) {
      const latencyMs = Number((performance.now() - start).toFixed(2));
      this.httpMetrics.recordRequest(502, latencyMs);
      return { status: 502, error: 'cache node unavailable', latencyMs };
    }

    await new Promise((r) => setTimeout(r, Math.random() * 0.9 + 0.3));

    const delRes = node.delete(key);
    const latencyMs = Number((performance.now() - start).toFixed(2));

    if (!delRes.deleted) {
      this.httpMetrics.recordRequest(404, latencyMs);
      return {
        status: 404,
        error: delRes.expired ? 'key expired' : 'key not found',
        targetNodeId: node.id,
        targetAddress: node.address,
        latencyMs,
      };
    }

    this.httpMetrics.recordRequest(200, latencyMs);
    return {
      status: 200,
      data: { message: 'deleted' },
      targetNodeId: node.id,
      targetAddress: node.address,
      latencyMs,
    };
  }

  /**
   * Aggregate metrics across all simulated nodes.
   */
  public getAggregateCacheMetrics(): CacheSnapshot {
    let hits = 0;
    let misses = 0;
    let sets = 0;
    let deletes = 0;
    let evictions = 0;
    let expired = 0;

    for (const node of this.nodes.values()) {
      const m = node.getMetrics();
      hits += m.hits;
      misses += m.misses;
      sets += m.sets;
      deletes += m.deletes;
      evictions += m.evictions;
      expired += m.expired;
    }

    const total = hits + misses;
    const hitRate = total === 0 ? 0 : Number((hits / total).toFixed(4));

    return { hits, misses, hitRate, sets, deletes, evictions, expired };
  }
}
