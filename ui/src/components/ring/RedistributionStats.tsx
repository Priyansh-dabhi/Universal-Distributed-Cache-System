import React, { useState } from 'react';
import { GlassCard } from '../shared/GlassCard';
import { fnv1a } from '../../core/sim/fnv1a';
import { ConsistentHashRing } from '../../core/sim/hash-ring';
import { ShieldAlert, Zap, Play } from 'lucide-react';

interface RedistributionStatsProps {
  nodeCount: number;
}

export const RedistributionStats: React.FC<RedistributionStatsProps> = ({ nodeCount }) => {
  const [isRunning, setIsRunning] = useState(false);
  const [testResult, setTestResult] = useState<{
    totalKeys: number;
    consistentMoved: number;
    consistentPct: number;
    moduloMoved: number;
    moduloPct: number;
  } | null>(null);

  const runSimulation = () => {
    setIsRunning(true);

    const testKeys = Array.from({ length: 100 }, (_, i) => `benchmark:key_${i * 37 + 11}`);

    // Consistent Ring Before (N nodes)
    const ringBefore = new ConsistentHashRing(50);
    for (let i = 1; i <= nodeCount; i++) {
      ringBefore.addNode({ id: `node-${i}`, address: `http://localhost:${8000 + i}`, color: '#06b6d4' });
    }

    // Consistent Ring After (N + 1 nodes)
    const ringAfter = new ConsistentHashRing(50);
    for (let i = 1; i <= nodeCount + 1; i++) {
      ringAfter.addNode({ id: `node-${i}`, address: `http://localhost:${8000 + i}`, color: '#06b6d4' });
    }

    let consistentMoved = 0;
    let moduloMoved = 0;

    for (const key of testKeys) {
      // Consistent Hashing
      const beforeNode = ringBefore.route(key).targetNodeId;
      const afterNode = ringAfter.route(key).targetNodeId;
      if (beforeNode !== afterNode) {
        consistentMoved++;
      }

      // Modulo Hashing: hash % N vs hash % (N+1)
      const h = fnv1a(key);
      const modBefore = h % nodeCount;
      const modAfter = h % (nodeCount + 1);
      if (modBefore !== modAfter) {
        moduloMoved++;
      }
    }

    setTimeout(() => {
      setTestResult({
        totalKeys: testKeys.length,
        consistentMoved,
        consistentPct: Math.round((consistentMoved / testKeys.length) * 100),
        moduloMoved,
        moduloPct: Math.round((moduloMoved / testKeys.length) * 100),
      });
      setIsRunning(false);
    }, 200);
  };

  const theoreticalConsistentPct = Math.round((1 / (nodeCount + 1)) * 100);
  const theoreticalModuloPct = Math.round((nodeCount / (nodeCount + 1)) * 100);

  return (
    <GlassCard
      title="Minimal Key Redistribution Proof"
      subtitle={`Adding node #${nodeCount + 1}: Consistent Hashing vs Traditional Modulo`}
      icon={<Zap className="w-5 h-5 text-amber-400" />}
      action={
        <button
          onClick={runSimulation}
          disabled={isRunning}
          className="px-3 py-1.5 rounded-xl bg-amber-500/10 hover:bg-amber-500/20 border border-amber-500/30 text-amber-300 text-xs font-medium flex items-center gap-1.5 transition-colors cursor-pointer disabled:opacity-50"
        >
          <Play className={`w-3.5 h-3.5 ${isRunning ? 'animate-spin' : ''}`} />
          <span>Test 100 Sample Keys</span>
        </button>
      }
    >
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 pt-1 font-mono text-xs">
        {/* Consistent Hashing Card */}
        <div className="p-4 rounded-xl border border-emerald-500/25 bg-emerald-500/[0.04] space-y-2">
          <div className="flex items-center justify-between">
            <span className="font-bold text-emerald-400 flex items-center gap-1.5">
              <Zap className="w-4 h-4" /> Consistent Hash Ring
            </span>
            <span className="text-[10px] uppercase font-mono px-2 py-0.5 rounded-full bg-emerald-500/20 text-emerald-300">
              Optimal: 1/(N+1)
            </span>
          </div>

          <p className="text-slate-300 text-[11px] font-sans leading-relaxed">
            Only keys falling on the newly registered node's clockwise arc are remapped. All other nodes retain their resident caches.
          </p>

          <div className="pt-2 flex items-baseline gap-2">
            <span className="text-2xl font-bold text-emerald-300">
              {testResult ? `${testResult.consistentPct}%` : `~${theoreticalConsistentPct}%`}
            </span>
            <span className="text-slate-400 text-[11px]">
              keys moved ({testResult ? `${testResult.consistentMoved}/100` : `theoretical`})
            </span>
          </div>
          <div className="w-full h-1.5 rounded-full bg-slate-800 overflow-hidden">
            <div
              className="h-full bg-emerald-400 rounded-full transition-all duration-500"
              style={{ width: `${testResult ? testResult.consistentPct : theoreticalConsistentPct}%` }}
            />
          </div>
        </div>

        {/* Traditional Modulo Card */}
        <div className="p-4 rounded-xl border border-rose-500/25 bg-rose-500/[0.04] space-y-2">
          <div className="flex items-center justify-between">
            <span className="font-bold text-rose-400 flex items-center gap-1.5">
              <ShieldAlert className="w-4 h-4" /> Traditional Modulo (hash % N)
            </span>
            <span className="text-[10px] uppercase font-mono px-2 py-0.5 rounded-full bg-rose-500/20 text-rose-300">
              Flawed: N/(N+1)
            </span>
          </div>

          <p className="text-slate-300 text-[11px] font-sans leading-relaxed">
            Changing the divisor shifts almost every single key's destination. Triggers a massive cache stampede crashing external databases.
          </p>

          <div className="pt-2 flex items-baseline gap-2">
            <span className="text-2xl font-bold text-rose-300">
              {testResult ? `${testResult.moduloPct}%` : `~${theoreticalModuloPct}%`}
            </span>
            <span className="text-slate-400 text-[11px]">
              keys displaced ({testResult ? `${testResult.moduloMoved}/100` : `theoretical`})
            </span>
          </div>
          <div className="w-full h-1.5 rounded-full bg-slate-800 overflow-hidden">
            <div
              className="h-full bg-rose-400 rounded-full transition-all duration-500"
              style={{ width: `${testResult ? testResult.moduloPct : theoreticalModuloPct}%` }}
            />
          </div>
        </div>
      </div>
    </GlassCard>
  );
};
