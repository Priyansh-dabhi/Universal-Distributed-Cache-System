import React from 'react';
import { useMetrics } from '../hooks/useMetrics';
import { useBackend } from '../hooks/useBackend';
import { CounterCards } from '../components/metrics/CounterCards';
import { HitRateGauge } from '../components/metrics/HitRateGauge';
import { ThroughputChart } from '../components/metrics/ThroughputChart';
import { LatencyChart } from '../components/metrics/LatencyChart';
import { WorkloadRunner } from '../components/metrics/WorkloadRunner';
import { GlassCard } from '../components/shared/GlassCard';
import { Badge } from '../components/shared/Badge';
import { Activity, RotateCcw, TrendingUp, Gauge } from 'lucide-react';
import { toast } from 'sonner';

export const MetricsPage: React.FC = () => {
  const { mode } = useBackend();
  const { cacheMetrics, httpMetrics, timeSeries, resetMetrics, refresh } = useMetrics(1500);

  const handleReset = () => {
    resetMetrics();
    toast.info('Telemetry counters reset to zero.');
  };

  const totalQueries = cacheMetrics.hits + cacheMetrics.misses;

  return (
    <div className="space-y-6">
      {/* Page Title & Status Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <div className="flex items-center gap-2.5">
            <h2 className="text-xl sm:text-2xl font-bold text-white tracking-tight flex items-center gap-2">
              <Activity className="w-6 h-6 text-amber-400" />
              Cluster Telemetry & Observability
            </h2>
            <Badge variant={mode === 'live' ? 'live' : 'sim'} pulse>
              {mode === 'live' ? 'Live Telemetry' : 'Web Simulator Telemetry'}
            </Badge>
          </div>
          <p className="text-xs sm:text-sm text-slate-400 mt-1">
            Real-time lock-free metrics, dynamic hit rate evaluation, ops/sec throughput streaming, and synthetic traffic injection.
          </p>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-2">
          {mode === 'sim' && (
            <button
              onClick={handleReset}
              className="px-3 py-1.5 rounded-xl bg-white/5 hover:bg-rose-500/10 text-slate-400 hover:text-rose-400 text-xs font-mono border border-white/5 transition-colors flex items-center gap-1.5 cursor-pointer"
            >
              <RotateCcw className="w-3.5 h-3.5" />
              <span>Reset Counters</span>
            </button>
          )}
        </div>
      </div>

      {/* Row 1: Atomic Cache Engine Counter Cards */}
      <CounterCards metrics={cacheMetrics} />

      {/* Row 2: Throughput Chart & Hit Rate Radial Gauge */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
        {/* Left: Operations per Second Area Chart */}
        <div className="lg:col-span-8">
          <GlassCard
            title="Throughput & Query Activity (Rolling Stream)"
            subtitle="Real-time operations/sec, cache hits, and cache misses over time"
            icon={<TrendingUp className="w-5 h-5 text-cyan-400" />}
          >
            <ThroughputChart data={timeSeries} />
          </GlassCard>
        </div>

        {/* Right: Circular Hit Rate Gauge */}
        <div className="lg:col-span-4">
          <GlassCard
            title="Cache Hit Ratio"
            subtitle="hits / (hits + misses)"
            icon={<Activity className="w-5 h-5 text-emerald-400" />}
          >
            <HitRateGauge
              hitRate={cacheMetrics.hitRate}
              totalRequests={totalQueries}
            />
          </GlassCard>
        </div>
      </div>

      {/* Row 3: HTTP Level Telemetry & Latency */}
      <GlassCard
        title="HTTP Request Telemetry & Latency Distribution"
        subtitle="Reverse-proxy response classes and millisecond latency averages"
        icon={<Gauge className="w-5 h-5 text-amber-400" />}
      >
        <LatencyChart metrics={httpMetrics} />
      </GlassCard>

      {/* Row 4: Synthetic Workload & Traffic Generator */}
      <WorkloadRunner onWorkloadComplete={refresh} />
    </div>
  );
};
