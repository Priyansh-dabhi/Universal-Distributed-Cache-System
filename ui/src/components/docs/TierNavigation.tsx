import type { ArchitectureTier } from '../../core/docs/concepts-data';

interface TierNavigationProps {
  tiers: ArchitectureTier[];
  selectedTier: number;
  onSelectTier: (tierNumber: number) => void;
}

export const TierNavigation: React.FC<TierNavigationProps> = ({
  tiers,
  selectedTier,
  onSelectTier,
}) => {
  return (
    <div className="space-y-1.5 font-mono text-xs">
      <span className="text-[11px] uppercase text-slate-400 font-bold tracking-wider px-2 block mb-2">
        Architecture Tiers
      </span>

      {tiers.map((tier) => {
        const isSelected = tier.tierNumber === selectedTier;

        return (
          <button
            key={tier.tierNumber}
            onClick={() => onSelectTier(tier.tierNumber)}
            className={`w-full text-left p-2.5 rounded-xl transition-all cursor-pointer flex items-center justify-between border ${
              isSelected
                ? 'bg-cyan-500/10 border-cyan-500/40 text-cyan-300 shadow-sm shadow-cyan-500/10'
                : 'border-transparent text-slate-400 hover:text-slate-200 hover:bg-white/5'
            }`}
          >
            <div className="flex items-center gap-2">
              <span
                className={`w-5 h-5 rounded-lg flex items-center justify-center text-[10px] font-bold ${
                  isSelected ? 'bg-cyan-500 text-slate-950' : 'bg-slate-800 text-slate-400'
                }`}
              >
                {tier.tierNumber}
              </span>
              <span className="truncate max-w-[170px]">{tier.title}</span>
            </div>

            <span className="text-[10px] text-slate-500">
              {tier.concepts.length} topics
            </span>
          </button>
        );
      })}
    </div>
  );
};
