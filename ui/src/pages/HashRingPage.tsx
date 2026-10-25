import React, { useState, useMemo } from 'react';
import { useHashRing } from '../hooks/useHashRing';
import { useBackend } from '../hooks/useBackend';
import { HashRingCanvas } from '../components/ring/HashRingCanvas';
import { RingControls } from '../components/ring/RingControls';
import { RouteResultCard } from '../components/ring/RouteResultCard';
import { RedistributionStats } from '../components/ring/RedistributionStats';
import { GlassCard } from '../components/shared/GlassCard';
import { AnimatedCounter } from '../components/shared/AnimatedCounter';
import { Badge } from '../components/shared/Badge';
import { CircleDot, Server, Layers, Cpu, Hash } from 'lucide-react';
import type { RouteResult } from '../core/types';

export const HashRingPage: React.FC = () => {
  const { mode } = useBackend();
  const { nodes, positions, replicas, routeKey, addNode, removeNode } = useHashRing();
  const [userRoute, setUserRoute] = useState<RouteResult | null>(null);
  const [showVirtualNodes, setShowVirtualNodes] = useState<boolean>(true);

  // Derive active route from user interaction or default to 'user:101'
  const activeRoute = useMemo(() => {
    if (userRoute) return userRoute;
    if (nodes.length > 0) {
      try {
        return routeKey('user:101');
      } catch {
        return null;
      }
    }
    return null;
  }, [userRoute, nodes, routeKey]);

  const handleRouteKey = (key: string) => {
    try {
      const result = routeKey(key);
      setUserRoute(result);
    } catch (err: unknown) {
      console.error('Route error:', err);
    }
  };

  return (
    <div className="space-y-6">
      {/* Page Title & Status Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <div className="flex items-center gap-2.5">
            <h2 className="text-xl sm:text-2xl font-bold text-white tracking-tight flex items-center gap-2">
              <CircleDot className="w-6 h-6 text-cyan-400" />
              Consistent Hash Ring Visualizer
            </h2>
            <Badge variant={mode === 'live' ? 'live' : 'sim'} pulse>
              {mode === 'live' ? 'Live Cluster' : 'Web Simulator'}
            </Badge>
          </div>
          <p className="text-xs sm:text-sm text-slate-400 mt-1">
            Observe circular 360° key placement, virtual node replica distribution, and O(log R) clockwise routing lookups.
          </p>
        </div>

        {/* Quick Top Stat Pills */}
        <div className="flex items-center gap-2 font-mono text-xs">
          <div className="px-3 py-1.5 rounded-xl bg-slate-900/80 border border-white/10 flex items-center gap-2">
            <Server className="w-3.5 h-3.5 text-cyan-400" />
            <span className="text-slate-400">Nodes:</span>
            <AnimatedCounter value={nodes.length} className="font-bold text-white" />
          </div>
          <div className="px-3 py-1.5 rounded-xl bg-slate-900/80 border border-white/10 flex items-center gap-2">
            <Layers className="w-3.5 h-3.5 text-violet-400" />
            <span className="text-slate-400">Tokens:</span>
            <AnimatedCounter value={positions.length} className="font-bold text-white" />
          </div>
          <div className="px-3 py-1.5 rounded-xl bg-slate-900/80 border border-white/10 hidden md:flex items-center gap-2">
            <Cpu className="w-3.5 h-3.5 text-emerald-400" />
            <span className="text-slate-400">Replicas:</span>
            <span className="font-bold text-white">{replicas}/node</span>
          </div>
        </div>
      </div>

      {/* Main Interactive Visualizer Layout */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
        {/* Left Column: 360° SVG Hash Ring */}
        <div className="lg:col-span-7">
          <GlassCard
            title="Circular Token Perimeter (0 → 2³² - 1)"
            subtitle="Virtual nodes are scattered deterministically using FNV-1a(nodeId#replica)"
            icon={<Hash className="w-5 h-5 text-cyan-400" />}
            className="flex flex-col items-center justify-center p-6"
          >
            <HashRingCanvas
              nodes={nodes}
              positions={positions}
              activeRoute={activeRoute}
              showVirtualNodes={showVirtualNodes}
            />
          </GlassCard>
        </div>

        {/* Right Column: Controls & Routing Step-by-Step Breakdown */}
        <div className="lg:col-span-5 space-y-6">
          {/* Key Routing Controls */}
          <GlassCard
            title="Interactive Key Router"
            subtitle="Enter any cache key to compute its position and trace its path"
            icon={<CircleDot className="w-5 h-5 text-emerald-400" />}
          >
            <RingControls
              nodes={nodes}
              replicas={replicas}
              showVirtualNodes={showVirtualNodes}
              onToggleVirtualNodes={() => setShowVirtualNodes(!showVirtualNodes)}
              onRouteKey={handleRouteKey}
              onAddNode={addNode}
              onRemoveNode={removeNode}
              isLiveMode={mode === 'live'}
            />
          </GlassCard>

          {/* Mathematical Step-by-Step Inspector */}
          <GlassCard
            title="Clockwise Lookup Inspector"
            subtitle="Step-by-step resolution from raw key string to backend target"
            icon={<Layers className="w-5 h-5 text-violet-400" />}
          >
            <RouteResultCard route={activeRoute} />
          </GlassCard>
        </div>
      </div>

      {/* Bottom Section: Educational Minimal Key Redistribution Proof */}
      <RedistributionStats nodeCount={nodes.length} />
    </div>
  );
};
