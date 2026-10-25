import { useState, useCallback } from 'react';
import { useBackend } from './useBackend';
import type { NodeInfo, RequestLog } from '../core/types';
import { toast } from 'sonner';

export interface UseCacheReturn {
  get: (key: string) => Promise<RequestLog>;
  set: (key: string, value: string, ttl?: string) => Promise<RequestLog>;
  deleteKey: (key: string) => Promise<RequestLog>;
  history: RequestLog[];
  lastLog: RequestLog | null;
  isExecuting: boolean;
  clearHistory: () => void;
  nodes: NodeInfo[];
  refreshNodes: () => Promise<void>;
}

export function useCache(): UseCacheReturn {
  const { mode, simRouter, apiClient, triggerUpdate } = useBackend();
  const [history, setHistory] = useState<RequestLog[]>([]);
  const [lastLog, setLastLog] = useState<RequestLog | null>(null);
  const [isExecuting, setIsExecuting] = useState<boolean>(false);
  const [nodes, setNodes] = useState<NodeInfo[]>(() => {
    return simRouter.getNodes();
  });

  const appendLog = useCallback((log: RequestLog) => {
    setLastLog(log);
    setHistory((prev) => [log, ...prev].slice(0, 50));
  }, []);

  const refreshNodes = useCallback(async () => {
    if (mode === 'live') {
      try {
        const res = await apiClient.getNodes();
        if (res.ok && res.data?.nodes) {
          const detailedNodes: NodeInfo[] = [];
          for (const n of res.data.nodes) {
            // Attempt to query node info directly
            const infoRes = await apiClient.getNodeInfo(n.address);
            if (infoRes.ok && infoRes.data) {
              detailedNodes.push({
                ...infoRes.data,
                address: n.address,
                health: 'healthy',
              });
            } else {
              detailedNodes.push({
                id: n.id,
                address: n.address,
                capacity: 100,
                policy: 'LRU',
                size: 0,
                health: 'healthy',
              });
            }
          }
          setNodes(detailedNodes);
        }
      } catch (err) {
        console.error('Failed to refresh live nodes info:', err);
      }
    } else {
      setNodes(simRouter.getNodes());
    }
  }, [mode, apiClient, simRouter]);

  const get = useCallback(async (key: string): Promise<RequestLog> => {
    setIsExecuting(true);
    const id = Math.random().toString(36).substring(2, 9);
    const timestamp = Date.now();

    try {
      if (mode === 'live') {
        const res = await apiClient.getCache(key);
        const log: RequestLog = {
          id,
          timestamp,
          method: 'GET',
          key,
          value: res.data?.value,
          status: res.status,
          statusText: res.ok ? '200 OK' : `${res.status} ${res.error || 'Error'}`,
          latencyMs: res.latencyMs,
          isHit: res.ok,
        };

        appendLog(log);
        if (res.ok) {
          toast.success(`HIT: "${key}" retrieved`, { description: `Value: "${res.data?.value}" (${res.latencyMs}ms)` });
        } else {
          toast.error(`MISS: "${key}" not found`, { description: `Status: ${res.status} (${res.latencyMs}ms)` });
        }
        await refreshNodes();
        return log;
      } else {
        const res = await simRouter.get(key);
        const log: RequestLog = {
          id,
          timestamp,
          method: 'GET',
          key,
          value: res.data?.value,
          status: res.status,
          statusText: res.status === 200 ? '200 OK' : `${res.status} ${res.error}`,
          routedToNodeId: res.targetNodeId,
          routedToAddress: res.targetAddress,
          latencyMs: res.latencyMs,
          isHit: res.isHit,
        };

        appendLog(log);
        triggerUpdate();
        if (res.isHit) {
          toast.success(`HIT: "${key}" retrieved`, {
            description: `From ${res.targetNodeId} in ${res.latencyMs}ms`,
          });
        } else {
          toast.error(`MISS: "${key}" not found`, {
            description: `Routed to ${res.targetNodeId} in ${res.latencyMs}ms (${res.error})`,
          });
        }
        setNodes(simRouter.getNodes());
        return log;
      }
    } finally {
      setIsExecuting(false);
    }
  }, [mode, apiClient, simRouter, appendLog, refreshNodes, triggerUpdate]);

  const set = useCallback(async (key: string, value: string, ttl?: string): Promise<RequestLog> => {
    setIsExecuting(true);
    const id = Math.random().toString(36).substring(2, 9);
    const timestamp = Date.now();

    // Parse TTL string like "10s", "1m" into ms for simulator
    let ttlMs: number | null = null;
    if (ttl) {
      if (ttl.endsWith('s')) {
        ttlMs = parseInt(ttl) * 1000;
      } else if (ttl.endsWith('m')) {
        ttlMs = parseInt(ttl) * 60 * 1000;
      }
    }

    try {
      if (mode === 'live') {
        const res = await apiClient.setCache(key, value, ttl);
        const log: RequestLog = {
          id,
          timestamp,
          method: 'PUT',
          key,
          value,
          ttl,
          status: res.status,
          statusText: res.ok ? '200 OK' : `${res.status} ${res.error}`,
          latencyMs: res.latencyMs,
        };

        appendLog(log);
        if (res.ok) {
          toast.success(`SET: "${key}" saved`, { description: `Latency: ${res.latencyMs}ms` });
        } else {
          toast.error(`SET failed for "${key}"`, { description: res.error });
        }
        await refreshNodes();
        return log;
      } else {
        const res = await simRouter.set(key, value, ttlMs);
        const log: RequestLog = {
          id,
          timestamp,
          method: 'PUT',
          key,
          value,
          ttl,
          status: res.status,
          statusText: res.status === 200 ? '200 OK' : `${res.status} ${res.error}`,
          routedToNodeId: res.targetNodeId,
          routedToAddress: res.targetAddress,
          latencyMs: res.latencyMs,
        };

        appendLog(log);
        triggerUpdate();
        toast.success(`SET: "${key}" stored`, {
          description: `Routed to ${res.targetNodeId} (${res.latencyMs}ms)${res.data?.evicted ? ' • Caused Eviction!' : ''}`,
        });
        setNodes(simRouter.getNodes());
        return log;
      }
    } finally {
      setIsExecuting(false);
    }
  }, [mode, apiClient, simRouter, appendLog, refreshNodes, triggerUpdate]);

  const deleteKey = useCallback(async (key: string): Promise<RequestLog> => {
    setIsExecuting(true);
    const id = Math.random().toString(36).substring(2, 9);
    const timestamp = Date.now();

    try {
      if (mode === 'live') {
        const res = await apiClient.deleteCache(key);
        const log: RequestLog = {
          id,
          timestamp,
          method: 'DELETE',
          key,
          status: res.status,
          statusText: res.ok ? '200 OK' : `${res.status} ${res.error}`,
          latencyMs: res.latencyMs,
        };

        appendLog(log);
        if (res.ok) {
          toast.success(`DELETED: "${key}" removed`);
        } else {
          toast.error(`DELETE failed: "${key}" not found`);
        }
        await refreshNodes();
        return log;
      } else {
        const res = await simRouter.delete(key);
        const log: RequestLog = {
          id,
          timestamp,
          method: 'DELETE',
          key,
          status: res.status,
          statusText: res.status === 200 ? '200 OK' : `${res.status} ${res.error}`,
          routedToNodeId: res.targetNodeId,
          routedToAddress: res.targetAddress,
          latencyMs: res.latencyMs,
        };

        appendLog(log);
        triggerUpdate();
        if (res.status === 200) {
          toast.success(`DELETED: "${key}" from ${res.targetNodeId}`);
        } else {
          toast.error(`DELETE failed: "${key}" (${res.error})`);
        }
        setNodes(simRouter.getNodes());
        return log;
      }
    } finally {
      setIsExecuting(false);
    }
  }, [mode, apiClient, simRouter, appendLog, refreshNodes, triggerUpdate]);

  const clearHistory = useCallback(() => {
    setHistory([]);
    setLastLog(null);
  }, []);

  return {
    get,
    set,
    deleteKey,
    history,
    lastLog,
    isExecuting,
    clearHistory,
    nodes,
    refreshNodes,
  };
}
