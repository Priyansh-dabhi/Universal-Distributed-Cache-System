export type EvictionPolicy = 'LRU' | 'LFU' | '2Q';

export interface CacheEntry {
  key: string;
  value: string;
  freq?: number;
  qType?: 'A1' | 'Am';
  expiresAt?: number | null; // Unix timestamp in ms
  createdAt: number;
}

export interface NodeInfo {
  id: string;
  address: string;
  capacity: number;
  policy: EvictionPolicy;
  size: number;
  health: 'healthy' | 'unhealthy' | 'warning';
}

export interface CacheSnapshot {
  hits: number;
  misses: number;
  hitRate: number;
  sets: number;
  deletes: number;
  evictions: number;
  expired: number;
}

export interface HTTPSnapshot {
  requests: number;
  successes: number;
  errors4xx: number;
  errors5xx: number;
  avgLatencyMs: number;
}

export interface RouteResult {
  key: string;
  hash: number;
  hashHex: string;
  angle: number; // 0 to 360 degrees
  targetNodeId: string;
  targetAddress: string;
  targetReplicaToken?: string;
  targetPosition: number;
}

export interface RingNode {
  id: string;
  address: string;
  color: string;
}

export interface RingPosition {
  position: number;
  angle: number;
  nodeId: string;
  token: string;
  isVirtual: boolean;
}

export interface RequestLog {
  id: string;
  timestamp: number;
  method: 'GET' | 'PUT' | 'DELETE';
  key: string;
  value?: string;
  ttl?: string;
  status: number;
  statusText: string;
  routedToNodeId?: string;
  routedToAddress?: string;
  latencyMs: number;
  isHit?: boolean;
}

export interface WorkloadProgress {
  isRunning: boolean;
  type: 'zipfian' | 'uniform' | 'scan';
  total: number;
  completed: number;
  hits: number;
  misses: number;
  hitRate: number;
}
