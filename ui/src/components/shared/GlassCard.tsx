import React from 'react';
import { motion } from 'framer-motion';
import { clsx } from 'clsx';

interface GlassCardProps {
  title?: React.ReactNode;
  subtitle?: string;
  icon?: React.ReactNode;
  action?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
  hoverEffect?: boolean;
  glow?: 'cyan' | 'emerald' | 'violet' | 'amber' | 'none';
  onClick?: () => void;
}

export const GlassCard: React.FC<GlassCardProps> = ({
  title,
  subtitle,
  icon,
  action,
  children,
  className,
  hoverEffect = false,
  glow = 'none',
  onClick,
}) => {
  const glowStyles = {
    cyan: 'hover:border-cyan-500/40 hover:shadow-[0_0_25px_rgba(6,182,212,0.15)]',
    emerald: 'hover:border-emerald-500/40 hover:shadow-[0_0_25px_rgba(16,185,129,0.15)]',
    violet: 'hover:border-violet-500/40 hover:shadow-[0_0_25px_rgba(139,92,246,0.15)]',
    amber: 'hover:border-amber-500/40 hover:shadow-[0_0_25px_rgba(245,158,11,0.15)]',
    none: '',
  }[glow];

  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.25 }}
      onClick={onClick}
      className={clsx(
        'glass-panel rounded-2xl p-5 relative overflow-hidden transition-all duration-300',
        hoverEffect && 'glass-panel-hover cursor-pointer',
        glowStyles,
        className
      )}
    >
      {/* Subtle top ambient specular highlight */}
      <div className="absolute inset-x-0 top-0 h-[1px] bg-gradient-to-r from-transparent via-white/15 to-transparent pointer-events-none" />

      {(title || action) && (
        <div className="flex items-center justify-between pb-3.5 mb-4 border-b border-white/5">
          <div className="flex items-center gap-3">
            {icon && (
              <div className="p-2 rounded-xl bg-white/5 text-cyan-400 border border-white/10 shadow-inner">
                {icon}
              </div>
            )}
            <div>
              {typeof title === 'string' ? (
                <h3 className="font-semibold text-slate-100 text-sm tracking-wide">{title}</h3>
              ) : (
                title
              )}
              {subtitle && <p className="text-xs text-slate-400 mt-0.5">{subtitle}</p>}
            </div>
          </div>
          {action && <div className="flex items-center gap-2">{action}</div>}
        </div>
      )}
      {children}
    </motion.div>
  );
};
