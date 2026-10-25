import React, { useState } from 'react';
import { GlassCard } from '../shared/GlassCard';
import { LRUEngine } from '../../core/sim/lru-engine';
import { TwoQEngine } from '../../core/sim/twoq-engine';
import { ShieldAlert, ShieldCheck, Play, RotateCcw } from 'lucide-react';

export const ScanAttackDemo: React.FC = () => {
  const [isRunning, setIsRunning] = useState(false);
  const [hasRun, setHasRun] = useState(false);
  const [lruSurvivors, setLruSurvivors] = useState<string[]>([]);
  const [twoQSurvivors, setTwoQSurvivors] = useState<string[]>([]);

  const HOT_KEYS = ['hot:alice', 'hot:session_jwt', 'hot:config_api'];

  const runScanSimulation = () => {
    setIsRunning(true);
    setHasRun(true);

    const cap = 5;
    const lru = new LRUEngine(cap);
    const twoQ = new TwoQEngine(cap, 0.4); // 2 slots A1, 3 slots Am

    // Step 1: Pre-populate hot keys and access them to establish high recency/promotion
    for (const k of HOT_KEYS) {
      lru.set(k, 'hot_val');
      lru.get(k);

      twoQ.set(k, 'hot_val');
      twoQ.get(k); // 2nd access: promotes to Am!
    }

    // Step 2: Inject a sequential scan attack of 6 unique cold keys
    const SCAN_KEYS = ['scan:table_01', 'scan:table_02', 'scan:table_03', 'scan:table_04', 'scan:table_05', 'scan:table_06'];

    setTimeout(() => {
      for (const sk of SCAN_KEYS) {
        lru.set(sk, 'scan_row');
        twoQ.set(sk, 'scan_row');
      }

      // Check which hot keys survived
      const lruSurv = HOT_KEYS.filter((k) => lru.get(k).found);
      const twoQSurv = HOT_KEYS.filter((k) => twoQ.get(k).found);

      setLruSurvivors(lruSurv);
      setTwoQSurvivors(twoQSurv);
      setIsRunning(false);
    }, 400);
  };

  const handleReset = () => {
    setHasRun(false);
    setLruSurvivors([]);
    setTwoQSurvivors([]);
  };

  return (
    <GlassCard
      title="Cache Pollution & Scan Attack Simulator"
      subtitle="Compare how LRU flushes hot data vs how 2Q contains sequential scans"
      icon={<ShieldAlert className="w-5 h-5 text-rose-400" />}
      action={
        <div className="flex items-center gap-2">
          {hasRun && (
            <button
              onClick={handleReset}
              className="p-1.5 rounded-lg bg-white/5 hover:bg-white/10 text-slate-400 text-xs transition-colors cursor-pointer"
              title="Reset test"
            >
              <RotateCcw className="w-3.5 h-3.5" />
            </button>
          )}
          <button
            onClick={runScanSimulation}
            disabled={isRunning}
            className="px-3.5 py-1.5 rounded-xl bg-gradient-to-r from-rose-500/20 to-rose-600/20 hover:from-rose-500/30 hover:to-rose-600/30 border border-rose-500/40 text-rose-300 text-xs font-mono font-medium flex items-center gap-1.5 transition-all cursor-pointer disabled:opacity-50"
          >
            <Play className={`w-3.5 h-3.5 ${isRunning ? 'animate-spin' : ''}`} />
            <span>Simulate Table Scan Attack</span>
          </button>
        </div>
      }
    >
      <div className="space-y-4 font-mono text-xs pt-1">
        <p className="text-slate-300 text-[11px] font-sans leading-relaxed">
          In real systems, an unindexed database query or sequential batch job reads thousands of rows once. A standard <strong>LRU cache</strong> flushes all valuable frequently-accessed keys to make room for one-time scan noise. <strong>2Q</strong> prevents this by confining scans to the A1 probation queue.
        </p>

        {/* Results Comparison Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {/* LRU Result */}
          <div className="p-4 rounded-xl border border-rose-500/30 bg-rose-500/[0.03] space-y-2.5">
            <div className="flex items-center justify-between">
              <span className="font-bold text-rose-400 flex items-center gap-1.5">
                <ShieldAlert className="w-4 h-4" />
                Standard LRU Cache
              </span>
              <span className="text-[10px] uppercase px-2 py-0.5 rounded-full bg-rose-500/20 text-rose-300 font-bold">
                Polluted
              </span>
            </div>

            <p className="text-[11px] text-slate-400 font-sans">
              Hot keys were completely evicted by cold scan data.
            </p>

            <div className="pt-1">
              <span className="text-[10px] text-slate-400">Hot Keys Survived:</span>
              <div className="flex items-center gap-1.5 pt-1">
                {HOT_KEYS.map((k) => {
                  const survived = lruSurvivors.includes(k);
                  return (
                    <span
                      key={k}
                      className={`px-2 py-0.5 rounded text-[10px] border ${
                        hasRun
                          ? survived
                            ? 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30'
                            : 'bg-rose-500/20 text-rose-400 border-rose-500/30 line-through'
                          : 'bg-white/5 text-slate-400 border-white/5'
                      }`}
                    >
                      {k.replace('hot:', '')}
                    </span>
                  );
                })}
              </div>
            </div>

            <p className="text-[10px] text-rose-400 pt-1">
              {hasRun ? `Outcome: ${lruSurvivors.length}/${HOT_KEYS.length} hot keys survived (Cache Flushed)` : 'Awaiting attack simulation...'}
            </p>
          </div>

          {/* 2Q Result */}
          <div className="p-4 rounded-xl border border-emerald-500/30 bg-emerald-500/[0.03] space-y-2.5">
            <div className="flex items-center justify-between">
              <span className="font-bold text-emerald-400 flex items-center gap-1.5">
                <ShieldCheck className="w-4 h-4" />
                2Q (Two-Queue) Cache
              </span>
              <span className="text-[10px] uppercase px-2 py-0.5 rounded-full bg-emerald-500/20 text-emerald-300 font-bold">
                Protected
              </span>
            </div>

            <p className="text-[11px] text-slate-400 font-sans">
              Scan items only cycled through A1. Queue Am shielded all hot keys.
            </p>

            <div className="pt-1">
              <span className="text-[10px] text-slate-400">Hot Keys Survived:</span>
              <div className="flex items-center gap-1.5 pt-1">
                {HOT_KEYS.map((k) => {
                  const survived = twoQSurvivors.includes(k);
                  return (
                    <span
                      key={k}
                      className={`px-2 py-0.5 rounded text-[10px] border ${
                        hasRun
                          ? survived
                            ? 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30'
                            : 'bg-rose-500/20 text-rose-400 border-rose-500/30 line-through'
                          : 'bg-white/5 text-slate-400 border-white/5'
                      }`}
                    >
                      {k.replace('hot:', '')}
                    </span>
                  );
                })}
              </div>
            </div>

            <p className="text-[10px] text-emerald-400 pt-1">
              {hasRun ? `Outcome: ${twoQSurvivors.length}/${HOT_KEYS.length} hot keys survived (100% Protected)` : 'Awaiting attack simulation...'}
            </p>
          </div>
        </div>
      </div>
    </GlassCard>
  );
};
