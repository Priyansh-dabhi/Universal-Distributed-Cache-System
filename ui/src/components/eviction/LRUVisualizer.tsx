import React from 'react';
import { AnimatePresence } from 'framer-motion';
import { EvictionSlot } from './EvictionSlot';
import type { CacheEntry } from '../../core/types';
import { ArrowLeftRight, Layers } from 'lucide-react';

interface LRUVisualizerProps {
  entries: CacheEntry[];
  capacity: number;
  recentKey: string | null;
  onSlotClick: (key: string) => void;
}

export const LRUVisualizer: React.FC<LRUVisualizerProps> = ({
  entries,
  capacity,
  recentKey,
  onSlotClick,
}) => {
  return (
    <div className="space-y-4">
      {/* Header Info */}
      <div className="flex items-center justify-between text-xs font-mono">
        <div className="flex items-center gap-2">
          <span className="px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-bold">
            MRU &bull; HEAD (Most Recent)
          </span>
          <ArrowLeftRight className="w-3.5 h-3.5 text-slate-500" />
          <span className="px-2 py-0.5 rounded-full bg-rose-500/10 text-rose-400 border border-rose-500/20 font-bold">
            LRU &bull; TAIL (Least Recent)
          </span>
        </div>

        <span className="text-slate-400">
          Capacity: <strong className="text-white">{entries.length} / {capacity}</strong>
        </span>
      </div>

      {/* Linked List Track */}
      {entries.length === 0 ? (
        <div className="h-44 flex flex-col items-center justify-center p-6 text-center border border-dashed border-white/10 rounded-2xl bg-white/[0.01]">
          <Layers className="w-8 h-8 text-slate-500 mb-2 opacity-50" />
          <p className="text-sm font-medium text-slate-300">LRU Doubly Linked List Empty</p>
          <p className="text-xs text-slate-500 mt-1 max-w-sm">
            Insert keys using the controls below. New or accessed entries jump to the front (HEAD). When capacity fills up, the tail node is evicted.
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3 relative">
          <AnimatePresence>
            {entries.map((entry, index) => {
              const isHead = index === 0;
              const isTail = index === entries.length - 1;
              const isRecent = entry.key === recentKey;

              return (
                <EvictionSlot
                  key={entry.key}
                  entry={entry}
                  isHead={isHead}
                  isTail={isTail && entries.length >= capacity}
                  isRecentlyAccessed={isRecent}
                  policy="LRU"
                  onClick={() => onSlotClick(entry.key)}
                />
              );
            })}
          </AnimatePresence>
        </div>
      )}

      {/* Explanatory Legend */}
      <div className="p-3 rounded-xl bg-slate-900/50 border border-white/5 text-[11px] font-mono text-slate-400 flex flex-wrap items-center justify-between gap-2">
        <span className="flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-emerald-400" />
          <span><strong>HEAD:</strong> Survives longest</span>
        </span>
        <span className="flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-cyan-400" />
          <span><strong>GET(key):</strong> Moves to HEAD in $O(1)$</span>
        </span>
        <span className="flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-rose-400" />
          <span><strong>TAIL:</strong> Removed when cache hits capacity</span>
        </span>
      </div>
    </div>
  );
};
