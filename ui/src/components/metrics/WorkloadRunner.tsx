import React, { useState } from 'react';
import { GlassCard } from '../shared/GlassCard';
import { useBackend } from '../../hooks/useBackend';
import { Play, Flame, Shuffle, ArrowRightLeft, Square, CheckCircle } from 'lucide-react';
import { toast } from 'sonner';

type WorkloadType = 'zipfian' | 'uniform' | 'scan';

interface WorkloadRunnerProps {
  onWorkloadComplete: () => void;
}

export const WorkloadRunner: React.FC<WorkloadRunnerProps> = ({ onWorkloadComplete }) => {
  const { mode, simRouter, apiClient } = useBackend();
  const [selectedType, setSelectedType] = useState<WorkloadType>('zipfian');
  const [totalOps, setTotalOps] = useState<number>(200);
  const [isRunning, setIsRunning] = useState<boolean>(false);
  const [progress, setProgress] = useState<{
    completed: number;
    hits: number;
    misses: number;
  }>({ completed: 0, hits: 0, misses: 0 });

  const abortRef = React.useRef(false);

  const WORKLOAD_PROFILES = [
    {
      type: 'zipfian' as WorkloadType,
      name: 'Zipfian (80/20 Distribution)',
      desc: '20% of keys receive 80% of all requests. Simulates realistic social media / ecommerce hot content.',
      icon: <Flame className="w-4 h-4 text-amber-400" />,
      color: 'border-amber-500/30 text-amber-300',
    },
    {
      type: 'uniform' as WorkloadType,
      name: 'Uniform Random',
      desc: 'Keys are chosen purely at random across a wide keyspace. Tests capacity limits under low locality.',
      icon: <Shuffle className="w-4 h-4 text-cyan-400" />,
      color: 'border-cyan-500/30 text-cyan-300',
    },
    {
      type: 'scan' as WorkloadType,
      name: 'Sequential Scan Batch',
      desc: 'Never-repeated keys simulating an unindexed SQL database scan. Tests cache eviction stress.',
      icon: <ArrowRightLeft className="w-4 h-4 text-violet-400" />,
      color: 'border-violet-500/30 text-violet-300',
    },
  ];

  const handleStartWorkload = async () => {
    setIsRunning(true);
    abortRef.current = false;
    setProgress({ completed: 0, hits: 0, misses: 0 });

    toast.info(`Starting ${selectedType.toUpperCase()} traffic workload (${totalOps} ops)...`);

    // Pre-populate 30 seed keys so there are targets to hit
    const keyPool = Array.from({ length: 30 }, (_, i) => `item:${i + 1}`);
    if (mode === 'sim') {
      for (const k of keyPool) {
        await simRouter.set(k, `val_${k}`);
      }
    } else {
      for (const k of keyPool.slice(0, 10)) {
        await apiClient.setCache(k, `val_${k}`);
      }
    }

    let completed = 0;
    let hits = 0;
    let misses = 0;

    // Helper: generate key according to distribution
    const generateKey = (step: number): string => {
      if (selectedType === 'zipfian') {
        // 80% probability to hit top 20% of keys
        if (Math.random() < 0.8) {
          const hotIdx = Math.floor(Math.random() * 6); // top 6 out of 30
          return keyPool[hotIdx];
        } else {
          const coldIdx = 6 + Math.floor(Math.random() * 24);
          return keyPool[coldIdx];
        }
      } else if (selectedType === 'uniform') {
        // Randomly pick from keypool or generate new
        if (Math.random() < 0.5) {
          const idx = Math.floor(Math.random() * keyPool.length);
          return keyPool[idx];
        }
        return `rand:${Math.floor(Math.random() * 500)}`;
      } else {
        // Sequential scan: strictly unique keys
        return `scan_key:${step}`;
      }
    };

    // Execute in batches of 10
    const batchSize = 10;
    for (let i = 0; i < totalOps; i += batchSize) {
      if (abortRef.current) break;

      const currentBatch = Math.min(batchSize, totalOps - i);
      const promises: Promise<unknown>[] = [];

      for (let j = 0; j < currentBatch; j++) {
        const key = generateKey(i + j);
        const isPut = Math.random() < 0.15; // 15% write, 85% read

        if (mode === 'sim') {
          if (isPut) {
            promises.push(simRouter.set(key, `updated_${Date.now()}`));
          } else {
            promises.push(
              simRouter.get(key).then((res) => {
                if (res.isHit) hits++;
                else misses++;
              })
            );
          }
        } else {
          if (isPut) {
            promises.push(apiClient.setCache(key, `updated_${Date.now()}`));
          } else {
            promises.push(
              apiClient.getCache(key).then((res) => {
                if (res.ok) hits++;
                else misses++;
              })
            );
          }
        }
      }

      await Promise.all(promises);
      completed += currentBatch;
      setProgress({ completed, hits, misses });

      // Small 25ms yield to allow React state updates and animation frames
      await new Promise((r) => setTimeout(r, 25));
    }

    setIsRunning(false);
    onWorkloadComplete();
    const finalRate = hits + misses > 0 ? Math.round((hits / (hits + misses)) * 100) : 0;
    toast.success(`Workload complete: ${completed} ops executed!`, {
      description: `Hit Rate: ${finalRate}% (${hits} Hits, ${misses} Misses)`,
    });
  };

  const handleStop = () => {
    abortRef.current = true;
    setIsRunning(false);
  };

  const currentPercent = Math.min(100, Math.round((progress.completed / totalOps) * 100));

  return (
    <GlassCard
      title="Synthetic Workload & Traffic Generator"
      subtitle="Stress-test the cluster with parameterized access distributions"
      icon={<Flame className="w-5 h-5 text-amber-400" />}
    >
      <div className="space-y-4 font-mono text-xs">
        {/* Profile Selector Cards */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
          {WORKLOAD_PROFILES.map((p) => (
            <div
              key={p.type}
              onClick={() => !isRunning && setSelectedType(p.type)}
              className={`p-3.5 rounded-xl border transition-all cursor-pointer ${
                selectedType === p.type
                  ? `${p.color} bg-white/[0.04] shadow-md`
                  : 'border-white/10 bg-slate-900/60 opacity-70 hover:opacity-100'
              }`}
            >
              <div className="flex items-center gap-2 pb-1.5 mb-1.5 border-b border-white/5">
                {p.icon}
                <span className="font-bold text-white text-xs">{p.name}</span>
              </div>
              <p className="text-[11px] font-sans text-slate-400 leading-relaxed">
                {p.desc}
              </p>
            </div>
          ))}
        </div>

        {/* Volume & Execution Controls */}
        <div className="p-3.5 rounded-xl bg-slate-900/80 border border-white/10 flex flex-wrap items-center justify-between gap-3">
          {/* Volume Selector */}
          <div className="flex items-center gap-2">
            <span className="text-slate-400">Request Volume:</span>
            <div className="flex items-center gap-1">
              {[100, 200, 500].map((v) => (
                <button
                  key={v}
                  disabled={isRunning}
                  onClick={() => setTotalOps(v)}
                  className={`px-2.5 py-1 rounded-lg text-xs font-bold transition-colors cursor-pointer ${
                    totalOps === v
                      ? 'bg-cyan-500 text-slate-950'
                      : 'bg-white/5 text-slate-400 hover:text-white'
                  }`}
                >
                  {v} ops
                </button>
              ))}
            </div>
          </div>

          {/* Action Button */}
          <div className="flex items-center gap-2">
            {isRunning ? (
              <button
                onClick={handleStop}
                className="px-4 py-2 rounded-xl bg-rose-500 hover:bg-rose-600 text-white font-bold flex items-center gap-2 transition-colors cursor-pointer shadow-lg shadow-rose-500/20"
              >
                <Square className="w-3.5 h-3.5 fill-current" />
                <span>Stop Workload</span>
              </button>
            ) : (
              <button
                onClick={handleStartWorkload}
                className="px-4 py-2 rounded-xl bg-gradient-to-r from-amber-500 to-amber-600 hover:from-amber-400 hover:to-amber-500 text-slate-950 font-bold flex items-center gap-2 transition-all cursor-pointer shadow-lg shadow-amber-500/20"
              >
                <Play className="w-3.5 h-3.5 fill-current" />
                <span>RUN {selectedType.toUpperCase()} TRAFFIC</span>
              </button>
            )}
          </div>
        </div>

        {/* Real-time Execution Progress Bar */}
        {(isRunning || progress.completed > 0) && (
          <div className="p-3.5 rounded-xl bg-slate-950/80 border border-white/10 space-y-2">
            <div className="flex items-center justify-between text-xs">
              <span className="text-slate-400 flex items-center gap-1.5">
                {progress.completed === totalOps ? (
                  <CheckCircle className="w-3.5 h-3.5 text-emerald-400" />
                ) : (
                  <span className="w-2 h-2 rounded-full bg-amber-400 animate-ping" />
                )}
                <span>
                  Executed: <strong>{progress.completed} / {totalOps}</strong> operations ({currentPercent}%)
                </span>
              </span>

              <span className="text-slate-300">
                Hits: <strong className="text-emerald-400">{progress.hits}</strong> &bull; Misses:{' '}
                <strong className="text-rose-400">{progress.misses}</strong>
              </span>
            </div>

            <div className="w-full h-2 rounded-full bg-slate-900 border border-white/5 overflow-hidden">
              <div
                className="h-full bg-gradient-to-r from-amber-500 to-emerald-400 rounded-full transition-all duration-150"
                style={{ width: `${currentPercent}%` }}
              />
            </div>
          </div>
        )}
      </div>
    </GlassCard>
  );
};
