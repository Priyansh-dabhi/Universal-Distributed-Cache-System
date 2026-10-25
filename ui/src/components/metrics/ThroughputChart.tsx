import React from 'react';
import {
  ResponsiveContainer,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  Tooltip,
  CartesianGrid,
} from 'recharts';
import type { MetricPoint } from '../../hooks/useMetrics';

interface ThroughputChartProps {
  data: MetricPoint[];
}

export const ThroughputChart: React.FC<ThroughputChartProps> = ({ data }) => {
  // If data is empty, supply minimal initial padding
  const chartData =
    data.length === 0
      ? Array.from({ length: 6 }, (_, i) => ({
          time: `00:0${i}`,
          requests: 0,
          hits: 0,
          misses: 0,
        }))
      : data;

  return (
    <div className="w-full h-64 font-mono text-xs">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={chartData} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
          <defs>
            {/* Gradients */}
            <linearGradient id="reqGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#06b6d4" stopOpacity={0.35} />
              <stop offset="95%" stopColor="#06b6d4" stopOpacity={0.0} />
            </linearGradient>
            <linearGradient id="hitsGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#10b981" stopOpacity={0.35} />
              <stop offset="95%" stopColor="#10b981" stopOpacity={0.0} />
            </linearGradient>
            <linearGradient id="missGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#f43f5e" stopOpacity={0.35} />
              <stop offset="95%" stopColor="#f43f5e" stopOpacity={0.0} />
            </linearGradient>
          </defs>

          <CartesianGrid strokeDasharray="3 3" stroke="#1e293b" opacity={0.6} />

          <XAxis
            dataKey="time"
            stroke="#64748b"
            fontSize={10}
            tickLine={false}
            axisLine={{ stroke: '#334155' }}
          />

          <YAxis
            stroke="#64748b"
            fontSize={10}
            tickLine={false}
            axisLine={{ stroke: '#334155' }}
            allowDecimals={false}
          />

          <Tooltip
            contentStyle={{
              backgroundColor: 'rgba(15, 23, 42, 0.95)',
              borderColor: 'rgba(255, 255, 255, 0.1)',
              borderRadius: '0.75rem',
              color: '#f8fafc',
              fontSize: '11px',
              fontFamily: 'monospace',
              boxShadow: '0 10px 25px rgba(0,0,0,0.5)',
            }}
          />

          <Area
            type="monotone"
            dataKey="requests"
            name="Ops/sec"
            stroke="#06b6d4"
            strokeWidth={2}
            fillOpacity={1}
            fill="url(#reqGradient)"
          />

          <Area
            type="monotone"
            dataKey="hits"
            name="Hits/sec"
            stroke="#10b981"
            strokeWidth={2}
            fillOpacity={1}
            fill="url(#hitsGradient)"
          />

          <Area
            type="monotone"
            dataKey="misses"
            name="Misses/sec"
            stroke="#f43f5e"
            strokeWidth={2}
            fillOpacity={1}
            fill="url(#missGradient)"
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
};
