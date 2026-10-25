import { useState, useEffect, useCallback, useRef } from 'react';
import { useBackend } from './useBackend';
import type { CacheSnapshot, HTTPSnapshot } from '../core/types';

export interface MetricPoint {
  time: string;
  timestamp: number;
  requests: number;
  hits: number;
  misses: number;
  hitRate: number;
  latencyMs: number;
}

export interface UseMetricsReturn {
  cacheMetrics: CacheSnapshot;
  httpMetrics: HTTPSnapshot;
  timeSeries: MetricPoint[];
  resetMetrics: () => void;
  refresh: () => Promise<void>;
}

export function useMetrics(pollIntervalMs: number = 2000): UseMetricsReturn {
  const { mode, simRouter, apiClient, lastUpdate } = useBackend();

  const [cacheMetrics, setCacheMetrics] = useState<CacheSnapshot>(() =>
    simRouter.getAggregateCacheMetrics()
  );
  const [httpMetrics, setHttpMetrics] = useState<HTTPSnapshot>(() =>
    simRouter.getHTTPMetrics()
  );
  const [timeSeries, setTimeSeries] = useState<MetricPoint[]>([]);

  const lastReqsRef = useRef<number>(0);
  const lastHitsRef = useRef<number>(0);
  const lastMissesRef = useRef<number>(0);

  const fetchMetrics = useCallback(async () => {
    const now = new Date();
    const timeStr = now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });

    if (mode === 'live') {
      try {
        const httpRes = await apiClient.getRouterMetrics();
        if (httpRes.ok && httpRes.data) {
          setHttpMetrics(httpRes.data);

          const deltaReqs = Math.max(0, httpRes.data.requests - lastReqsRef.current);
          lastReqsRef.current = httpRes.data.requests;

          setTimeSeries((prev) => [
            ...prev.slice(-30),
            {
              time: timeStr,
              timestamp: Date.now(),
              requests: deltaReqs,
              hits: 0,
              misses: 0,
              hitRate: 0,
              latencyMs: httpRes.data!.avgLatencyMs,
            },
          ]);
        }
      } catch (err) {
        console.error('Failed to poll live metrics:', err);
      }
    } else {
      // Simulator mode: immediate atomic read
      const cSnap = simRouter.getAggregateCacheMetrics();
      const hSnap = simRouter.getHTTPMetrics();

      setCacheMetrics(cSnap);
      setHttpMetrics(hSnap);

      const deltaReqs = Math.max(0, hSnap.requests - lastReqsRef.current);
      const deltaHits = Math.max(0, cSnap.hits - lastHitsRef.current);
      const deltaMisses = Math.max(0, cSnap.misses - lastMissesRef.current);

      lastReqsRef.current = hSnap.requests;
      lastHitsRef.current = cSnap.hits;
      lastMissesRef.current = cSnap.misses;

      setTimeSeries((prev) => [
        ...prev.slice(-30),
        {
          time: timeStr,
          timestamp: Date.now(),
          requests: deltaReqs,
          hits: deltaHits,
          misses: deltaMisses,
          hitRate: cSnap.hitRate * 100,
          latencyMs: hSnap.avgLatencyMs,
        },
      ]);
    }
  }, [mode, apiClient, simRouter]);

  // Periodic polling
  useEffect(() => {
    let active = true;
    const poll = () => {
      if (active) {
        void fetchMetrics();
      }
    };

    poll();
    const timer = setInterval(poll, pollIntervalMs);
    return () => {
      active = false;
      clearInterval(timer);
    };
  }, [fetchMetrics, pollIntervalMs, lastUpdate]);

  const resetMetrics = useCallback(() => {
    if (mode === 'sim') {
      for (const node of simRouter.getNodes()) {
        const n = simRouter.getNode(node.id);
        n?.clear();
      }
      setCacheMetrics(simRouter.getAggregateCacheMetrics());
      setHttpMetrics(simRouter.getHTTPMetrics());
      setTimeSeries([]);
      lastReqsRef.current = 0;
      lastHitsRef.current = 0;
      lastMissesRef.current = 0;
    }
  }, [mode, simRouter]);

  return {
    cacheMetrics,
    httpMetrics,
    timeSeries,
    resetMetrics,
    refresh: fetchMetrics,
  };
}
