import { AnimatedCounter } from '../shared/AnimatedCounter';
import type { CacheSnapshot } from '../../core/types';
import { CheckCircle2, XCircle, ArrowUpCircle, Trash2, Zap, Clock } from 'lucide-react';

interface CounterCardsProps {
  metrics: CacheSnapshot;
}

export const CounterCards: React.FC<CounterCardsProps> = ({ metrics }) => {
  const cards = [
    {
      label: 'Cache Hits',
      value: metrics.hits,
      icon: <CheckCircle2 className="w-4 h-4 text-emerald-400" />,
      color: 'text-emerald-400',
      border: 'border-emerald-500/20',
      bg: 'bg-emerald-500/[0.03]',
      desc: 'Keys served from memory',
    },
    {
      label: 'Cache Misses',
      value: metrics.misses,
      icon: <XCircle className="w-4 h-4 text-rose-400" />,
      color: 'text-rose-400',
      border: 'border-rose-500/20',
      bg: 'bg-rose-500/[0.03]',
      desc: 'Keys not found in nodes',
    },
    {
      label: 'Total Sets (PUT)',
      value: metrics.sets,
      icon: <ArrowUpCircle className="w-4 h-4 text-cyan-400" />,
      color: 'text-cyan-400',
      border: 'border-cyan-500/20',
      bg: 'bg-cyan-500/[0.03]',
      desc: 'New keys written to cache',
    },
    {
      label: 'Evictions',
      value: metrics.evictions,
      icon: <Zap className="w-4 h-4 text-amber-400" />,
      color: 'text-amber-400',
      border: 'border-amber-500/20',
      bg: 'bg-amber-500/[0.03]',
      desc: 'Items discarded on overflow',
    },
    {
      label: 'Expired (TTL)',
      value: metrics.expired,
      icon: <Clock className="w-4 h-4 text-violet-400" />,
      color: 'text-violet-400',
      border: 'border-violet-500/20',
      bg: 'bg-violet-500/[0.03]',
      desc: 'Purged by lazy expiration',
    },
    {
      label: 'Deletes',
      value: metrics.deletes,
      icon: <Trash2 className="w-4 h-4 text-slate-400" />,
      color: 'text-slate-300',
      border: 'border-slate-500/20',
      bg: 'bg-slate-500/[0.03]',
      desc: 'Explicit client removals',
    },
  ];

  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3 font-mono">
      {cards.map((c) => (
        <div
          key={c.label}
          className={`p-3.5 rounded-2xl border ${c.border} ${c.bg} backdrop-blur-md space-y-1 transition-all hover:scale-[1.02]`}
        >
          <div className="flex items-center justify-between text-slate-400 text-[11px]">
            <span className="truncate">{c.label}</span>
            {c.icon}
          </div>
          <div className={`text-xl sm:text-2xl font-bold tracking-tight ${c.color}`}>
            <AnimatedCounter value={c.value} />
          </div>
          <p className="text-[10px] text-slate-500 font-sans truncate">{c.desc}</p>
        </div>
      ))}
    </div>
  );
};
