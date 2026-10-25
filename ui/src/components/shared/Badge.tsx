import React from 'react';
import { motion } from 'framer-motion';
import { clsx } from 'clsx';

export type BadgeVariant =
  | 'hit'
  | 'miss'
  | 'evicted'
  | 'expired'
  | 'live'
  | 'sim'
  | 'node'
  | 'neutral'
  | 'cyan'
  | 'violet';

interface BadgeProps {
  children: React.ReactNode;
  variant?: BadgeVariant;
  size?: 'sm' | 'md';
  pulse?: boolean;
  className?: string;
  icon?: React.ReactNode;
}

const VARIANT_STYLES: Record<BadgeVariant, { bg: string; text: string; border: string; dot: string }> = {
  hit: {
    bg: 'bg-emerald-500/10',
    text: 'text-emerald-400',
    border: 'border-emerald-500/20',
    dot: 'bg-emerald-400',
  },
  miss: {
    bg: 'bg-rose-500/10',
    text: 'text-rose-400',
    border: 'border-rose-500/20',
    dot: 'bg-rose-400',
  },
  evicted: {
    bg: 'bg-amber-500/10',
    text: 'text-amber-400',
    border: 'border-amber-500/20',
    dot: 'bg-amber-400',
  },
  expired: {
    bg: 'bg-slate-500/10',
    text: 'text-slate-400',
    border: 'border-slate-500/20',
    dot: 'bg-slate-400',
  },
  live: {
    bg: 'bg-emerald-500/15',
    text: 'text-emerald-300',
    border: 'border-emerald-500/30',
    dot: 'bg-emerald-400',
  },
  sim: {
    bg: 'bg-cyan-500/15',
    text: 'text-cyan-300',
    border: 'border-cyan-500/30',
    dot: 'bg-cyan-400',
  },
  node: {
    bg: 'bg-violet-500/10',
    text: 'text-violet-300',
    border: 'border-violet-500/20',
    dot: 'bg-violet-400',
  },
  neutral: {
    bg: 'bg-slate-800/60',
    text: 'text-slate-300',
    border: 'border-slate-700/50',
    dot: 'bg-slate-400',
  },
  cyan: {
    bg: 'bg-cyan-500/10',
    text: 'text-cyan-400',
    border: 'border-cyan-500/20',
    dot: 'bg-cyan-400',
  },
  violet: {
    bg: 'bg-violet-500/10',
    text: 'text-violet-400',
    border: 'border-violet-500/20',
    dot: 'bg-violet-400',
  },
};

export const Badge: React.FC<BadgeProps> = ({
  children,
  variant = 'neutral',
  size = 'md',
  pulse = false,
  className,
  icon,
}) => {
  const styles = VARIANT_STYLES[variant];

  return (
    <motion.span
      initial={{ scale: 0.9, opacity: 0 }}
      animate={{ scale: 1, opacity: 1 }}
      transition={{ duration: 0.15 }}
      className={clsx(
        'inline-flex items-center gap-1.5 font-mono font-medium rounded-full border tracking-wide uppercase',
        styles.bg,
        styles.text,
        styles.border,
        size === 'sm' ? 'px-2 py-0.5 text-[10px]' : 'px-2.5 py-1 text-xs',
        className
      )}
    >
      {pulse ? (
        <span className="relative flex h-1.5 w-1.5">
          <span className={clsx('animate-ping absolute inline-flex h-full w-full rounded-full opacity-75', styles.dot)} />
          <span className={clsx('relative inline-flex rounded-full h-1.5 w-1.5', styles.dot)} />
        </span>
      ) : (
        icon || <span className={clsx('w-1.5 h-1.5 rounded-full', styles.dot)} />
      )}
      {children}
    </motion.span>
  );
};
