import React from 'react';
import { motion } from 'framer-motion';
import { AnimatedCounter } from '../shared/AnimatedCounter';
import { Activity } from 'lucide-react';

interface HitRateGaugeProps {
  hitRate: number; // 0.0 to 1.0
  totalRequests: number;
}

export const HitRateGauge: React.FC<HitRateGaugeProps> = ({ hitRate, totalRequests }) => {
  const percent = Math.round(hitRate * 100);

  // SVG Gauge Math (semi-circle / 260 deg arc)
  const size = 200;
  const strokeWidth = 12;
  const radius = (size - strokeWidth) / 2;
  const circumference = 2 * Math.PI * radius;
  const strokeDashoffset = circumference - (circumference * percent) / 100;

  const color =
    percent >= 70
      ? { text: 'text-emerald-400', stroke: '#10b981', label: 'Optimal Performance' }
      : percent >= 40
      ? { text: 'text-amber-400', stroke: '#f59e0b', label: 'Moderate Efficiency' }
      : { text: 'text-rose-400', stroke: '#f43f5e', label: 'Cold / Suboptimal' };

  return (
    <div className="flex flex-col items-center justify-center p-4 font-mono text-center">
      <div className="relative w-48 h-48 flex items-center justify-center">
        <svg className="w-full h-full -rotate-90 transform">
          {/* Background Track */}
          <circle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            fill="none"
            stroke="#1e293b"
            strokeWidth={strokeWidth}
            className="opacity-40"
          />

          {/* Animated Value Arc */}
          <motion.circle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            fill="none"
            stroke={color.stroke}
            strokeWidth={strokeWidth}
            strokeLinecap="round"
            strokeDasharray={circumference}
            initial={{ strokeDashoffset: circumference }}
            animate={{ strokeDashoffset }}
            transition={{ duration: 1, ease: 'easeOut' }}
            filter="drop-shadow(0 0 8px rgba(16,185,129,0.3))"
          />
        </svg>

        {/* Center Readout */}
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <Activity className="w-5 h-5 text-slate-500 mb-1" />
          <div className={`text-4xl font-extrabold tracking-tight ${color.text}`}>
            <AnimatedCounter value={percent} suffix="%" />
          </div>
          <span className="text-[10px] text-slate-400 uppercase tracking-widest mt-0.5">
            Hit Ratio
          </span>
        </div>
      </div>

      {/* Description Status Badge */}
      <div className="mt-2 space-y-1">
        <span
          className="text-xs font-bold px-2.5 py-0.5 rounded-full border inline-block"
          style={{
            color: color.stroke,
            borderColor: `${color.stroke}40`,
            backgroundColor: `${color.stroke}10`,
          }}
        >
          {color.label}
        </span>
        <p className="text-[11px] text-slate-500 font-sans">
          Evaluated across {totalRequests.toLocaleString()} total queries
        </p>
      </div>
    </div>
  );
};
