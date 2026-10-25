import React from 'react';
import { motion } from 'framer-motion';
import { getNodeColor } from '../../core/constants';
import type { RouteResult } from '../../core/types';
import { ArrowRight, Compass, Hash, Server, ShieldCheck } from 'lucide-react';

interface RouteResultCardProps {
  route: RouteResult | null;
}

export const RouteResultCard: React.FC<RouteResultCardProps> = ({ route }) => {
  if (!route) {
    return (
      <div className="h-full flex flex-col items-center justify-center p-6 text-center border border-dashed border-white/10 rounded-2xl bg-white/[0.01]">
        <Compass className="w-8 h-8 text-slate-500 mb-2 animate-spin-slow opacity-60" />
        <p className="text-sm font-medium text-slate-300">Awaiting Key to Route</p>
        <p className="text-xs text-slate-500 mt-1 max-w-xs">
          Type any cache key above or click a preset to see the step-by-step FNV-1a checksum and clockwise lookup.
        </p>
      </div>
    );
  }

  const nodeColor = getNodeColor(route.targetNodeId);
  const targetAngle = ((route.targetPosition >>> 0) / 0xffffffff) * 360;
  let angleDiff = targetAngle - route.angle;
  if (angleDiff < 0) angleDiff += 360;

  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2 }}
      className="space-y-4 font-mono text-xs"
    >
      {/* Route Summary Banner */}
      <div
        className="p-3.5 rounded-xl border flex items-center justify-between"
        style={{
          backgroundColor: `${nodeColor.bg}10`,
          borderColor: `${nodeColor.bg}35`,
        }}
      >
        <div className="flex items-center gap-2.5">
          <Server className="w-4 h-4" style={{ color: nodeColor.bg }} />
          <div>
            <span className="text-[10px] uppercase text-slate-400">Assigned Backend Node</span>
            <p className="text-sm font-bold text-white tracking-wide">{route.targetNodeId}</p>
          </div>
        </div>
        <div className="text-right">
          <span className="text-[10px] text-slate-400">Address</span>
          <p className="text-xs text-slate-300">{route.targetAddress}</p>
        </div>
      </div>

      {/* Step by Step Pipeline */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        {/* Step 1: Key & Hash */}
        <div className="p-3 rounded-xl bg-slate-900/60 border border-white/5 space-y-1">
          <div className="flex items-center gap-1.5 text-slate-400 text-[11px]">
            <Hash className="w-3.5 h-3.5 text-cyan-400" />
            <span>1. 32-bit FNV-1a Hash</span>
          </div>
          <p className="text-cyan-300 font-bold text-sm tracking-wider">{route.hashHex}</p>
          <p className="text-[10px] text-slate-500">Decimal: {route.hash >>> 0}</p>
        </div>

        {/* Step 2: Ring Angle */}
        <div className="p-3 rounded-xl bg-slate-900/60 border border-white/5 space-y-1">
          <div className="flex items-center gap-1.5 text-slate-400 text-[11px]">
            <Compass className="w-3.5 h-3.5 text-violet-400" />
            <span>2. Ring Position Angle</span>
          </div>
          <p className="text-violet-300 font-bold text-sm">{route.angle.toFixed(2)}°</p>
          <p className="text-[10px] text-slate-500">
            {((route.angle / 360) * 100).toFixed(1)}% of perimeter
          </p>
        </div>

        {/* Step 3: Successor Virtual Token */}
        <div className="p-3 rounded-xl bg-slate-900/60 border border-white/5 space-y-1">
          <div className="flex items-center gap-1.5 text-slate-400 text-[11px]">
            <ShieldCheck className="w-3.5 h-3.5 text-emerald-400" />
            <span>3. Nearest Clockwise Token</span>
          </div>
          <p className="text-emerald-300 font-bold text-xs truncate">
            {route.targetReplicaToken || 'default'}
          </p>
          <p className="text-[10px] text-slate-500">
            Token Hash: 0x{(route.targetPosition >>> 0).toString(16).toUpperCase()}
          </p>
        </div>

        {/* Step 4: Clockwise Delta */}
        <div className="p-3 rounded-xl bg-slate-900/60 border border-white/5 space-y-1">
          <div className="flex items-center gap-1.5 text-slate-400 text-[11px]">
            <ArrowRight className="w-3.5 h-3.5 text-amber-400" />
            <span>4. Clockwise Sweep Delta</span>
          </div>
          <p className="text-amber-300 font-bold text-sm">+{angleDiff.toFixed(2)}°</p>
          <p className="text-[10px] text-slate-500">Binary Search O(log R)</p>
        </div>
      </div>
    </motion.div>
  );
};
