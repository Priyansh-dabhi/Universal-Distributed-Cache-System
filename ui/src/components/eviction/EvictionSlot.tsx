import React from 'react';
import { motion } from 'framer-motion';
import { clsx } from 'clsx';
import type { CacheEntry } from '../../core/types';
import { Clock, Zap } from 'lucide-react';

interface EvictionSlotProps {
  entry: CacheEntry;
  isHead?: boolean;
  isTail?: boolean;
  isMinFreq?: boolean;
  isRecentlyAccessed?: boolean;
  isEvicted?: boolean;
  policy: 'LRU' | 'LFU' | '2Q';
  onClick?: () => void;
}

export const EvictionSlot: React.FC<EvictionSlotProps> = ({
  entry,
  isHead,
  isTail,
  isMinFreq,
  isRecentlyAccessed,
  isEvicted,
  policy,
  onClick,
}) => {
  return (
    <motion.div
      layout
      layoutId={entry.key}
      initial={{ scale: 0.7, opacity: 0, y: -15 }}
      animate={{
        scale: 1,
        opacity: isEvicted ? 0 : 1,
        y: 0,
        borderColor: isRecentlyAccessed
          ? 'rgba(6, 182, 212, 0.9)'
          : isTail && policy === 'LRU'
          ? 'rgba(244, 63, 94, 0.5)'
          : isMinFreq
          ? 'rgba(245, 158, 11, 0.5)'
          : 'rgba(255, 255, 255, 0.1)',
      }}
      exit={{ scale: 0.6, opacity: 0, y: 20 }}
      transition={{
        type: 'spring',
        stiffness: 350,
        damping: 25,
      }}
      onClick={onClick}
      className={clsx(
        'relative p-3 rounded-xl border select-none transition-shadow cursor-pointer font-mono group',
        isRecentlyAccessed
          ? 'bg-cyan-500/15 shadow-[0_0_20px_rgba(6,182,212,0.3)]'
          : isTail && policy === 'LRU'
          ? 'bg-rose-500/[0.08] shadow-[0_0_15px_rgba(244,63,94,0.15)]'
          : 'bg-slate-900/90 hover:bg-slate-800 hover:border-white/20'
      )}
    >
      {/* Top Indicators: MRU / LRU / Promoted */}
      <div className="flex items-center justify-between pb-1.5 border-b border-white/5 mb-1.5">
        <span className="font-bold text-xs text-white truncate max-w-[120px]">
          "{entry.key}"
        </span>

        {isHead && (
          <span className="text-[9px] font-bold px-1.5 py-0.5 rounded bg-emerald-500/20 text-emerald-400 border border-emerald-500/30">
            MRU
          </span>
        )}
        {isTail && policy === 'LRU' && (
          <span className="text-[9px] font-bold px-1.5 py-0.5 rounded bg-rose-500/20 text-rose-400 border border-rose-500/30 animate-pulse">
            NEXT EVICT
          </span>
        )}
        {policy === 'LFU' && entry.freq !== undefined && (
          <span className="text-[10px] font-bold px-1.5 py-0.5 rounded bg-violet-500/20 text-violet-300 border border-violet-500/30 flex items-center gap-0.5">
            <Zap className="w-2.5 h-2.5 text-violet-400" />
            {entry.freq}&times;
          </span>
        )}
        {policy === '2Q' && entry.qType && (
          <span
            className={clsx(
              'text-[9px] font-bold px-1.5 py-0.5 rounded border',
              entry.qType === 'Am'
                ? 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30'
                : 'bg-cyan-500/20 text-cyan-300 border-cyan-500/30'
            )}
          >
            {entry.qType === 'Am' ? 'Am (Hot)' : 'A1 (Probation)'}
          </span>
        )}
      </div>

      {/* Value preview */}
      <p className="text-[11px] text-slate-300 font-sans truncate bg-black/30 px-2 py-1 rounded border border-white/5">
        {entry.value}
      </p>

      {/* Footer metadata */}
      <div className="flex items-center justify-between pt-1.5 text-[9px] text-slate-400">
        <span className="flex items-center gap-1">
          <Clock className="w-2.5 h-2.5" />
          {entry.expiresAt ? 'TTL active' : 'No TTL'}
        </span>
        <span className="text-slate-500 group-hover:text-cyan-400 transition-colors">
          Click to Get &rarr;
        </span>
      </div>
    </motion.div>
  );
};
