import { useState, useEffect, useCallback, useMemo } from 'react';
import { useBackend } from './useBackend';
import { ConsistentHashRing } from '../core/sim/hash-ring';
import { getNodeColor } from '../core/constants';
import type { EvictionPolicy, RingNode, RingPosition, RouteResult } from '../core/types';
import { toast } from 'sonner';

export interface UseHashRingReturn {
  nodes: RingNode[];
  positions: RingPosition[];
  replicas: number;
  isLoading: boolean;
  routeKey: (key: string) => RouteResult;
  addNode: (node: { id: string; address?: string; capacity?: number; policy?: EvictionPolicy }, color?: string) => void;
  removeNode: (nodeId: string) => void;
  refresh: () => Promise<void>;
}

export function useHashRing(): UseHashRingReturn {
  const { mode, simRouter, apiClient, triggerUpdate } = useBackend();
  const [liveNodes, setLiveNodes] = useState<RingNode[]>([]);
  const [livePositions, setLivePositions] = useState<RingPosition[]>([]);
  const [liveReplicas, setLiveReplicas] = useState<number>(100);
  const [isLoading, setIsLoading] = useState<boolean>(false);

  // Client-side ring mirror for live mode to compute angles and positions
  const liveRingMirror = useMemo(() => new ConsistentHashRing(liveReplicas), [liveReplicas]);

  const refreshLiveNodes = useCallback(async () => {
    setIsLoading(true);
    try {
      const res = await apiClient.getNodes();
      if (res.ok && res.data?.nodes) {
        setLiveReplicas(res.data.replicas || 100);
        liveRingMirror.setReplicas(res.data.replicas || 100);

        // Clear existing nodes in mirror and populate
        const currentNodes = liveRingMirror.getNodes();
        for (const n of currentNodes) {
          try {
            liveRingMirror.removeNode(n.id);
          } catch {
            // ignore
          }
        }

        const formattedNodes: RingNode[] = res.data.nodes.map((n: { id: string; address: string }, i: number) => {
          const color = getNodeColor(n.id, i).bg;
          const nodeObj: RingNode = { id: n.id, address: n.address, color };
          try {
            liveRingMirror.addNode(nodeObj);
          } catch {
            // node might already be in mirror
          }
          return nodeObj;
        });

        setLiveNodes(formattedNodes);
        setLivePositions(liveRingMirror.getPositions());
      }
    } catch (err: unknown) {
      console.error('Failed to fetch live nodes:', err);
    } finally {
      setIsLoading(false);
    }
  }, [apiClient, liveRingMirror]);

  useEffect(() => {
    let active = true;
    let timer: ReturnType<typeof setTimeout> | null = null;

    if (mode === 'live') {
      timer = setTimeout(() => {
        if (active) void refreshLiveNodes();
      }, 0);
    }

    return () => {
      active = false;
      if (timer) clearTimeout(timer);
    };
  }, [mode, refreshLiveNodes]);

  const routeKey = useCallback((key: string): RouteResult => {
    if (mode === 'live') {
      return liveRingMirror.route(key);
    } else {
      return simRouter.route(key);
    }
  }, [mode, liveRingMirror, simRouter]);

  const addNode = useCallback((
    node: { id: string; address?: string; capacity?: number; policy?: EvictionPolicy },
    color?: string
  ) => {
    if (mode === 'live') {
      toast.warning('Node mutation disabled in Live Mode', {
        description: 'In Live Mode, backend nodes are controlled by the Go server CLI / flags.',
      });
      return;
    }

    try {
      simRouter.addNode(node, color);
      triggerUpdate();
      toast.success(`Node ${node.id} added to Hash Ring`, {
        description: `Virtual replicas scattered on ring. Key redistribution updated.`,
      });
    } catch (err: unknown) {
      const desc = err instanceof Error ? err.message : 'Unknown error';
      toast.error('Failed to add node', { description: desc });
    }
  }, [mode, simRouter, triggerUpdate]);

  const removeNode = useCallback((nodeId: string) => {
    if (mode === 'live') {
      toast.warning('Node mutation disabled in Live Mode', {
        description: 'In Live Mode, backend nodes are controlled by the Go server process.',
      });
      return;
    }

    try {
      simRouter.removeNode(nodeId);
      triggerUpdate();
      toast.info(`Node ${nodeId} removed from Hash Ring`, {
        description: `Traffic reassigned clockwise to neighboring nodes.`,
      });
    } catch (err: unknown) {
      const desc = err instanceof Error ? err.message : 'Unknown error';
      toast.error('Failed to remove node', { description: desc });
    }
  }, [mode, simRouter, triggerUpdate]);

  if (mode === 'live') {
    return {
      nodes: liveNodes,
      positions: livePositions,
      replicas: liveReplicas,
      isLoading,
      routeKey,
      addNode,
      removeNode,
      refresh: refreshLiveNodes,
    };
  }

  // Simulator mode
  const ring = simRouter.getRing();
  return {
    nodes: simRouter.getRingNodes(),
    positions: ring.getPositions(),
    replicas: ring.getReplicas(),
    isLoading: false,
    routeKey,
    addNode,
    removeNode,
    refresh: async () => {},
  };
}
