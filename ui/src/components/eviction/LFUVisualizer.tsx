import React from 'react';
import { AnimatePresence } from 'framer-motion';
import { EvictionSlot } from './EvictionSlot';
import type { CacheEntry } from '../../core/types';
import { Zap, Layers } from 'lucide-react';

interface LFUVisualizerProps {
  buckets: { freq: number; entries: CacheEntry[] }[];
  capacity: number;
  totalSize: number;
  recentKey: string | null;
  onSlotClick: (key: string) => void;
}

export const LFUVisualizer: React.FC<LFUVisualizerProps> = ({
  buckets,
  capacity,
  totalSize,
  recentKey,
  onSlotClick,
}) => {
  const minFreq = buckets.length > 0 ? buckets[0].freq : 0;

  return (
    <div className="space-y-4">
      {/* Header Info */}
      <div className="flex items-center justify-between text-xs font-mono">
        <div className="flex items-center gap-2">
          <span className="px-2 py-0.5 rounded-full bg-amber-500/10 text-amber-400 border border-amber-500/20 font-bold flex items-center gap-1">
            <Zap className="w-3 h-3" />
            Min Frequency Tier: {minFreq || '-'}&times; (Eviction Zone)
          </span>
        </div>

        <span className="text-slate-400">
          Capacity: <strong className="text-white">{totalSize} / {capacity}</strong>
        </span>
      </div>

      {/* Multi-tier Frequency Buckets */}
      {buckets.length === 0 ? (
        <div className="h-44 flex flex-col items-center justify-center p-6 text-center border border-dashed border-white/10 rounded-2xl bg-white/[0.01]">
          <Layers className="w-8 h-8 text-slate-500 mb-2 opacity-50" />
          <p className="text-sm font-medium text-slate-300">LFU Frequency Buckets Empty</p>
          <p className="text-xs text-slate-500 mt-1 max-w-sm">
            Add keys below to establish frequency columns. Every hit increments a key's count, promoting it to higher tiers. Eviction always targets the lowest frequency bucket.
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4 items-start">
          {buckets.map((bucket) => {
            const isMin = bucket.freq === minFreq;

            return (
              <div
                key={bucket.freq}
                className={`p-3 rounded-2xl border transition-all ${
                  isMin
                    ? 'border-amber-500/40 bg-amber-500/[0.03] shadow-[0_0_15px_rgba(245,158,11,0.1)]'
                    : 'border-white/10 bg-slate-900/60'
                }`}
              >
                {/* Bucket Header */}
                <div className="flex items-center justify-between pb-2 mb-2 border-b border-white/10 font-mono text-xs">
                  <span className="font-bold flex items-center gap-1 text-white">
                    <Zap className="w-3.5 h-3.5 text-violet-400" />
                    <span>{bucket.freq}&times; Accesses</span>
                  </span>
                  <span className="text-[10px] text-slate-400">
                    {bucket.entries.length} items
                  </span>
                </div>

                {/* Stack of Slots in this Frequency Tier */}
                <div className="space-y-2">
                  <AnimatePresence>
                    {bucket.entries.map((entry, idx) => {
                      const isRecent = entry.key === recentKey;
                      const isCandidateForEvict = isMin && idx === bucket.entries.length - 1 && totalSize >= capacity;

                      return (
                        <EvictionSlot
                          key={entry.key}
                          entry={entry}
                          isMinFreq={isCandidateForEvict}
                          isRecentlyAccessed={isRecent}
                          policy="LFU"
                          onClick={() => onSlotClick(entry.key)}
                        />
                      );
                    })}
                  </AnimatePresence>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Explanatory Legend */}
      <div className="p-3 rounded-xl bg-slate-900/50 border border-white/5 text-[11px] font-mono text-slate-400 flex flex-wrap items-center justify-between gap-2">
        <span><strong>Promotion:</strong> Each `GET(key)` elevates the item to a higher frequency tier</span>
        <span><strong>Tie-Breaking:</strong> If items have equal frequencies, LRU order resolves which to evict</span>
      </div>
    </div>
  );
};
