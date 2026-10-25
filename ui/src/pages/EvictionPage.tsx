import React, { useState, useMemo } from 'react';
import { GlassCard } from '../components/shared/GlassCard';
import { Badge } from '../components/shared/Badge';
import { LRUEngine } from '../core/sim/lru-engine';
import { LFUEngine } from '../core/sim/lfu-engine';
import { TwoQEngine } from '../core/sim/twoq-engine';
import { LRUVisualizer } from '../components/eviction/LRUVisualizer';
import { LFUVisualizer } from '../components/eviction/LFUVisualizer';
import { TwoQVisualizer } from '../components/eviction/TwoQVisualizer';
import { EvictionControls } from '../components/eviction/EvictionControls';
import { ScanAttackDemo } from '../components/eviction/ScanAttackDemo';
import { Layers, ShieldCheck, Zap, ArrowLeftRight } from 'lucide-react';
import type { EvictionPolicy } from '../core/types';
import { toast } from 'sonner';

export const EvictionPage: React.FC = () => {
  const [policy, setPolicy] = useState<EvictionPolicy>('LRU');
  const [capacity, setCapacity] = useState<number>(6);
  const [recentKey, setRecentKey] = useState<string | null>(null);
  const [, setVersion] = useState<number>(0);

  // Dedicated visualizer engines
  const lruEngine = useMemo(() => {
    const engine = new LRUEngine(capacity);
    engine.set('session:1', 'Alice');
    engine.set('user:profile', 'Bob');
    engine.set('cart:active', 'Items(3)');
    return engine;
  }, [capacity]);

  const lfuEngine = useMemo(() => {
    const engine = new LFUEngine(capacity);
    engine.set('api:rate_limit', '100');
    engine.set('meta:version', 'v2.4');
    engine.set('hot:landing_page', 'HTML');
    // Bump frequencies
    engine.get('hot:landing_page');
    engine.get('hot:landing_page');
    engine.get('meta:version');
    return engine;
  }, [capacity]);

  const twoQEngine = useMemo(() => {
    const engine = new TwoQEngine(capacity, 0.33); // 2 in A1, 4 in Am
    engine.set('probation:temp1', 'NewVal1');
    engine.set('probation:temp2', 'NewVal2');
    engine.set('promoted:main_session', 'PersistentVal');
    engine.get('promoted:main_session'); // 2nd access -> promotes to Am!
    return engine;
  }, [capacity]);

  const triggerUpdate = () => {
    setVersion((v) => v + 1);
  };

  const handleSet = (key: string, value: string) => {
    setRecentKey(key);
    let evicted = false;
    let evictedKey: string | undefined;

    if (policy === 'LRU') {
      const res = lruEngine.set(key, value);
      evicted = res.evicted;
      evictedKey = res.evictedKey;
    } else if (policy === 'LFU') {
      const res = lfuEngine.set(key, value);
      evicted = res.evicted;
      evictedKey = res.evictedKey;
    } else if (policy === '2Q') {
      const res = twoQEngine.set(key, value);
      evicted = res.evicted;
      evictedKey = res.evictedKey;
    }

    triggerUpdate();
    if (evicted) {
      toast.warning(`Eviction triggered: "${evictedKey}" was discarded to maintain capacity.`);
    } else {
      toast.success(`Stored "${key}" in ${policy} cache`);
    }
  };

  const handleGet = (key: string) => {
    setRecentKey(key);
    let found = false;

    if (policy === 'LRU') {
      const res = lruEngine.get(key);
      found = res.found;
    } else if (policy === 'LFU') {
      const res = lfuEngine.get(key);
      found = res.found;
      if (found) {
        toast.info(`Key "${key}" access count incremented to ${res.freq}×! Promoted to higher tier.`);
      }
    } else if (policy === '2Q') {
      const res = twoQEngine.get(key);
      found = res.found;
      if (res.promoted) {
        toast.success(`Key "${key}" accessed second time: Promoted from A1 to Am!`);
      }
    }

    triggerUpdate();
    if (!found) {
      toast.error(`Key "${key}" not found (Cache Miss)`);
    } else if (policy !== 'LFU') {
      toast.info(`Key "${key}" accessed: Updated recency in ${policy} cache`);
    }
  };

  const handleDelete = (key: string) => {
    if (policy === 'LRU') lruEngine.delete(key);
    else if (policy === 'LFU') lfuEngine.delete(key);
    else if (policy === '2Q') twoQEngine.delete(key);

    triggerUpdate();
    toast.success(`Deleted "${key}" from ${policy} cache`);
  };

  const handleFillCapacity = () => {
    let current = 0;
    if (policy === 'LRU') current = lruEngine.size();
    else if (policy === 'LFU') current = lfuEngine.size();
    else if (policy === '2Q') current = twoQEngine.size();

    let count = 1;
    while (current < capacity) {
      handleSet(`item:${count}`, `sample_data_${count}`);
      count++;
      if (policy === 'LRU') current = lruEngine.size();
      else if (policy === 'LFU') current = lfuEngine.size();
      else if (policy === '2Q') current = twoQEngine.size();
    }
  };

  const handleClear = () => {
    if (policy === 'LRU') lruEngine.clear();
    else if (policy === 'LFU') lfuEngine.clear();
    else if (policy === '2Q') twoQEngine.clear();
    setRecentKey(null);
    triggerUpdate();
    toast.info(`Cleared all items from ${policy} cache.`);
  };

  // Get current state snapshot
  let currentSize = 0;
  if (policy === 'LRU') currentSize = lruEngine.size();
  else if (policy === 'LFU') currentSize = lfuEngine.size();
  else if (policy === '2Q') currentSize = twoQEngine.size();

  return (
    <div className="space-y-6">
      {/* Page Title & Status Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <div className="flex items-center gap-2.5">
            <h2 className="text-xl sm:text-2xl font-bold text-white tracking-tight flex items-center gap-2">
              <Layers className="w-6 h-6 text-violet-400" />
              Eviction Policy Visualizer
            </h2>
            <Badge variant="violet" size="md">
              {policy} Active
            </Badge>
          </div>
          <p className="text-xs sm:text-sm text-slate-400 mt-1">
            Observe real-time linked-list reordering, frequency tier promotion, and scan-resistant dual queues.
          </p>
        </div>
      </div>

      {/* 3-Policy Switcher Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 font-mono text-xs">
        {/* LRU Card */}
        <div
          onClick={() => {
            setPolicy('LRU');
            setRecentKey(null);
          }}
          className={`p-4 rounded-2xl border transition-all cursor-pointer ${
            policy === 'LRU'
              ? 'border-cyan-500/50 bg-cyan-500/[0.08] shadow-[0_0_20px_rgba(6,182,212,0.2)]'
              : 'border-white/10 bg-slate-900/60 hover:bg-slate-800'
          }`}
        >
          <div className="flex items-center justify-between pb-2 mb-2 border-b border-white/5">
            <span className="font-bold text-sm text-white flex items-center gap-2">
              <ArrowLeftRight className="w-4 h-4 text-cyan-400" />
              LRU
            </span>
            <span className="text-[10px] text-cyan-400 uppercase font-bold">Least Recently Used</span>
          </div>
          <p className="text-slate-300 font-sans text-xs leading-relaxed">
            Maintains a doubly linked list. The least recently accessed item at the TAIL is evicted when capacity is reached.
          </p>
        </div>

        {/* LFU Card */}
        <div
          onClick={() => {
            setPolicy('LFU');
            setRecentKey(null);
          }}
          className={`p-4 rounded-2xl border transition-all cursor-pointer ${
            policy === 'LFU'
              ? 'border-violet-500/50 bg-violet-500/[0.08] shadow-[0_0_20px_rgba(139,92,246,0.2)]'
              : 'border-white/10 bg-slate-900/60 hover:bg-slate-800'
          }`}
        >
          <div className="flex items-center justify-between pb-2 mb-2 border-b border-white/5">
            <span className="font-bold text-sm text-white flex items-center gap-2">
              <Zap className="w-4 h-4 text-violet-400" />
              LFU
            </span>
            <span className="text-[10px] text-violet-400 uppercase font-bold">Least Frequently Used</span>
          </div>
          <p className="text-slate-300 font-sans text-xs leading-relaxed">
            Maintains frequency buckets. Evicts the item with the minimum access count, using LRU as a tie-breaker.
          </p>
        </div>

        {/* 2Q Card */}
        <div
          onClick={() => {
            setPolicy('2Q');
            setRecentKey(null);
          }}
          className={`p-4 rounded-2xl border transition-all cursor-pointer ${
            policy === '2Q'
              ? 'border-emerald-500/50 bg-emerald-500/[0.08] shadow-[0_0_20px_rgba(16,185,129,0.2)]'
              : 'border-white/10 bg-slate-900/60 hover:bg-slate-800'
          }`}
        >
          <div className="flex items-center justify-between pb-2 mb-2 border-b border-white/5">
            <span className="font-bold text-sm text-white flex items-center gap-2">
              <ShieldCheck className="w-4 h-4 text-emerald-400" />
              2Q
            </span>
            <span className="text-[10px] text-emerald-400 uppercase font-bold">Scan-Resistant Two-Queue</span>
          </div>
          <p className="text-slate-300 font-sans text-xs leading-relaxed">
            Separates items into A1 (probation) and Am (hot). Promotes to Am only on a second access to block cache pollution.
          </p>
        </div>
      </div>

      {/* Main Algorithm Visualizer Canvas */}
      <GlassCard
        title={`${policy} Live Memory Architecture`}
        subtitle={`Visualizing active slots (${currentSize} / ${capacity} capacity). Click any item to trigger a GET lookup.`}
        icon={<Layers className="w-5 h-5 text-violet-400" />}
      >
        <div className="py-2">
          {policy === 'LRU' && (
            <LRUVisualizer
              entries={lruEngine.getOrderedEntries()}
              capacity={capacity}
              recentKey={recentKey}
              onSlotClick={handleGet}
            />
          )}

          {policy === 'LFU' && (
            <LFUVisualizer
              buckets={lfuEngine.getBucketSnapshots()}
              capacity={capacity}
              totalSize={lfuEngine.size()}
              recentKey={recentKey}
              onSlotClick={handleGet}
            />
          )}

          {policy === '2Q' && (
            <TwoQVisualizer
              a1Entries={twoQEngine.getA1Entries()}
              amEntries={twoQEngine.getAmEntries()}
              a1Cap={twoQEngine.getA1Capacity()}
              amCap={twoQEngine.getAmCapacity()}
              totalCap={capacity}
              recentKey={recentKey}
              onSlotClick={handleGet}
            />
          )}
        </div>
      </GlassCard>

      {/* Interactive Controls Bar */}
      <GlassCard
        title="Eviction Experiment Controls"
        subtitle="Insert keys, trigger promotions, and force capacity overflows"
        icon={<Zap className="w-5 h-5 text-cyan-400" />}
      >
        <EvictionControls
          onSet={handleSet}
          onGet={handleGet}
          onDelete={handleDelete}
          onFillCapacity={handleFillCapacity}
          onClear={handleClear}
          capacity={capacity}
          onCapacityChange={(c) => setCapacity(c)}
          currentSize={currentSize}
        />
      </GlassCard>

      {/* Bottom Section: Scan Attack Simulation */}
      <ScanAttackDemo />
    </div>
  );
};
