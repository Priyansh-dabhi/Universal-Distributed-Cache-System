import React from 'react';
import type { HTTPSnapshot } from '../../core/types';
import { AnimatedCounter } from '../shared/AnimatedCounter';
import { Gauge, CheckCircle2, AlertTriangle, XCircle } from 'lucide-react';

interface LatencyChartProps {
  metrics: HTTPSnapshot;
}

export const LatencyChart: React.FC<LatencyChartProps> = ({ metrics }) => {
  const total = metrics.requests || 1;
  const pct2xx = Math.round((metrics.successes / total) * 100);
  const pct4xx = Math.round((metrics.errors4xx / total) * 100);
  const pct5xx = Math.round((metrics.errors5xx / total) * 100);

  return (
    <div className="space-y-4 font-mono text-xs">
      {/* Average Latency Hero Pill */}
      <div className="p-4 rounded-xl border border-white/10 bg-slate-900/80 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-xl bg-cyan-500/10 text-cyan-400 border border-cyan-500/20">
            <Gauge className="w-5 h-5" />
          </div>
          <div>
            <span className="text-[10px] text-slate-400 uppercase tracking-wider block">
              Average Latency
            </span>
            <div className="text-xl sm:text-2xl font-bold text-white flex items-baseline gap-1">
              <AnimatedCounter value={metrics.avgLatencyMs} decimals={2} />
              <span className="text-xs text-slate-400 font-normal">ms / op</span>
            </div>
          </div>
        </div>

        <div className="text-right">
          <span className="text-[10px] text-slate-400 uppercase tracking-wider block">
            Total HTTP Requests
          </span>
          <span className="text-sm font-bold text-slate-200">
            <AnimatedCounter value={metrics.requests} /> ops
          </span>
        </div>
      </div>

      {/* HTTP Status Code Distribution Bar */}
      <div className="space-y-2">
        <span className="text-[11px] text-slate-400 block">HTTP Response Status Ratio:</span>
        <div className="h-3 w-full rounded-full bg-slate-900 overflow-hidden flex border border-white/5">
          {pct2xx > 0 && (
            <div
              style={{ width: `${pct2xx}%` }}
              className="bg-emerald-500 transition-all duration-500"
              title={`2xx Success: ${metrics.successes} (${pct2xx}%)`}
            />
          )}
          {pct4xx > 0 && (
            <div
              style={{ width: `${pct4xx}%` }}
              className="bg-amber-500 transition-all duration-500"
              title={`4xx Client Error: ${metrics.errors4xx} (${pct4xx}%)`}
            />
          )}
          {pct5xx > 0 && (
            <div
              style={{ width: `${pct5xx}%` }}
              className="bg-rose-500 transition-all duration-500"
              title={`5xx Server Error: ${metrics.errors5xx} (${pct5xx}%)`}
            />
          )}
        </div>

        {/* Legend */}
        <div className="grid grid-cols-3 gap-2 pt-1 text-[11px]">
          <div className="flex items-center gap-1.5 text-emerald-400">
            <CheckCircle2 className="w-3.5 h-3.5" />
            <span>2xx: {metrics.successes}</span>
          </div>
          <div className="flex items-center gap-1.5 text-amber-400">
            <AlertTriangle className="w-3.5 h-3.5" />
            <span>4xx: {metrics.errors4xx}</span>
          </div>
          <div className="flex items-center gap-1.5 text-rose-400">
            <XCircle className="w-3.5 h-3.5" />
            <span>5xx: {metrics.errors5xx}</span>
          </div>
        </div>
      </div>
    </div>
  );
};
