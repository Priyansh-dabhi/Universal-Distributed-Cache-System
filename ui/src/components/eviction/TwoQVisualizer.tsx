import React from 'react';
import { AnimatePresence } from 'framer-motion';
import { EvictionSlot } from './EvictionSlot';
import type { CacheEntry } from '../../core/types';
import { ShieldCheck, ArrowRight } from 'lucide-react';

interface TwoQVisualizerProps {
  a1Entries: CacheEntry[];
  amEntries: CacheEntry[];
  a1Cap: number;
  amCap: number;
  totalCap: number;
  recentKey: string | null;
  onSlotClick: (key: string) => void;
}

export const TwoQVisualizer: React.FC<TwoQVisualizerProps> = ({
  a1Entries,
  amEntries,
  a1Cap,
  amCap,
  totalCap,
  recentKey,
  onSlotClick,
}) => {
  return (
    <div className="space-y-4">
      {/* Top Header */}
      <div className="flex items-center justify-between text-xs font-mono">
        <div className="flex items-center gap-2">
          <span className="px-2.5 py-1 rounded-full bg-cyan-500/10 text-cyan-400 border border-cyan-500/20 font-bold flex items-center gap-1.5">
            <ShieldCheck className="w-3.5 h-3.5" />
            Scan-Resistant Dual Queues
          </span>
        </div>

        <span className="text-slate-400">
          Total Memory: <strong className="text-white">{a1Entries.length + amEntries.length} / {totalCap}</strong>
        </span>
      </div>

      {/* Side-by-Side Dual Queue Split */}
      <div className="grid grid-cols-1 md:grid-cols-12 gap-6 items-start">
        {/* Left Queue: A1 (FIFO Probation) */}
        <div className="md:col-span-5 p-4 rounded-2xl border border-cyan-500/25 bg-cyan-500/[0.02] space-y-3">
          <div className="flex items-center justify-between pb-2 border-b border-white/10 font-mono text-xs">
            <span className="font-bold text-cyan-400 flex items-center gap-1.5">
              <span>A1 &bull; Probation (FIFO)</span>
            </span>
            <span className="text-[10px] text-slate-400">
              {a1Entries.length} / {a1Cap} slots
            </span>
          </div>

          <p className="text-[11px] text-slate-400 font-sans leading-relaxed">
            New keys land here on first insertion. If never accessed again, they are dropped from the tail without polluting the main cache.
          </p>

          <div className="space-y-2.5 min-h-[140px]">
            {a1Entries.length === 0 ? (
              <div className="h-32 flex flex-col items-center justify-center text-center p-4 border border-dashed border-white/10 rounded-xl text-slate-500 text-xs">
                Queue A1 is currently empty
              </div>
            ) : (
              <AnimatePresence>
                {a1Entries.map((entry, idx) => (
                  <EvictionSlot
                    key={entry.key}
                    entry={entry}
                    isTail={idx === a1Entries.length - 1 && a1Entries.length >= a1Cap}
                    isRecentlyAccessed={entry.key === recentKey}
                    policy="2Q"
                    onClick={() => onSlotClick(entry.key)}
                  />
                ))}
              </AnimatePresence>
            )}
          </div>
        </div>

        {/* Center Promotion Flow Arrow */}
        <div className="hidden md:flex md:col-span-2 flex-col items-center justify-center py-16 text-center space-y-2 font-mono">
          <div className="w-8 h-8 rounded-full bg-emerald-500/10 border border-emerald-500/30 flex items-center justify-center text-emerald-400">
            <ArrowRight className="w-4 h-4" />
          </div>
          <span className="text-[10px] text-emerald-400 font-bold uppercase tracking-wider">
            2nd Access
          </span>
          <span className="text-[9px] text-slate-500">
            Promotes to Am
          </span>
        </div>

        {/* Right Queue: Am (LRU Main) */}
        <div className="md:col-span-5 p-4 rounded-2xl border border-emerald-500/25 bg-emerald-500/[0.02] space-y-3">
          <div className="flex items-center justify-between pb-2 border-b border-white/10 font-mono text-xs">
            <span className="font-bold text-emerald-400 flex items-center gap-1.5">
              <span>Am &bull; Main Cache (LRU)</span>
            </span>
            <span className="text-[10px] text-slate-400">
              {amEntries.length} / {amCap} slots
            </span>
          </div>

          <p className="text-[11px] text-slate-400 font-sans leading-relaxed">
            Proven hot data. Once promoted, keys reside here under standard LRU management, safe from one-time batch scans.
          </p>

          <div className="space-y-2.5 min-h-[140px]">
            {amEntries.length === 0 ? (
              <div className="h-32 flex flex-col items-center justify-center text-center p-4 border border-dashed border-white/10 rounded-xl text-slate-500 text-xs">
                Queue Am is currently empty
              </div>
            ) : (
              <AnimatePresence>
                {amEntries.map((entry, idx) => (
                  <EvictionSlot
                    key={entry.key}
                    entry={entry}
                    isHead={idx === 0}
                    isTail={idx === amEntries.length - 1 && amEntries.length >= amCap}
                    isRecentlyAccessed={entry.key === recentKey}
                    policy="2Q"
                    onClick={() => onSlotClick(entry.key)}
                  />
                ))}
              </AnimatePresence>
            )}
          </div>
        </div>
      </div>

      {/* Explanatory Legend */}
      <div className="p-3 rounded-xl bg-slate-900/50 border border-white/5 text-[11px] font-mono text-slate-400 flex flex-wrap items-center justify-between gap-2">
        <span><strong>First Access:</strong> Pushed to A1 (probation)</span>
        <span><strong>Second Access:</strong> Promoted to Am (main hot cache)</span>
        <span><strong>Scan Resistance:</strong> Bulk unrepeated keys flush out A1 without clearing Am</span>
      </div>
    </div>
  );
};
