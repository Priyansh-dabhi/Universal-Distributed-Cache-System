import React, { useState } from 'react';
import { RotateCcw, PlusCircle, Trash2, Zap } from 'lucide-react';

interface EvictionControlsProps {
  onSet: (key: string, value: string) => void;
  onGet: (key: string) => void;
  onDelete: (key: string) => void;
  onFillCapacity: () => void;
  onClear: () => void;
  capacity: number;
  onCapacityChange: (cap: number) => void;
  currentSize: number;
}

export const EvictionControls: React.FC<EvictionControlsProps> = ({
  onSet,
  onGet,
  onDelete,
  onFillCapacity,
  onClear,
  capacity,
  onCapacityChange,
  currentSize,
}) => {
  const [key, setKey] = useState('item:hot_1');
  const [value, setValue] = useState('value_data');

  const handleSetSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!key.trim()) return;
    onSet(key.trim(), value.trim() || 'data');
  };

  const handleGetSubmit = () => {
    if (!key.trim()) return;
    onGet(key.trim());
  };

  const handleDeleteSubmit = () => {
    if (!key.trim()) return;
    onDelete(key.trim());
  };

  return (
    <div className="space-y-4 font-mono text-xs">
      {/* Input row */}
      <form onSubmit={handleSetSubmit} className="grid grid-cols-1 sm:grid-cols-12 gap-3 items-end">
        <div className="sm:col-span-5">
          <label className="block text-[10px] text-slate-400 mb-1">Key Name</label>
          <input
            type="text"
            value={key}
            onChange={(e) => setKey(e.target.value)}
            placeholder="key_name"
            className="w-full px-3 py-2 bg-slate-900 border border-white/10 rounded-xl text-white focus:outline-none focus:border-cyan-500/50"
          />
        </div>

        <div className="sm:col-span-4">
          <label className="block text-[10px] text-slate-400 mb-1">Value</label>
          <input
            type="text"
            value={value}
            onChange={(e) => setValue(e.target.value)}
            placeholder="data"
            className="w-full px-3 py-2 bg-slate-900 border border-white/10 rounded-xl text-white focus:outline-none focus:border-cyan-500/50"
          />
        </div>

        <div className="sm:col-span-3 flex items-center gap-1.5">
          <button
            type="submit"
            className="flex-1 py-2 px-3 rounded-xl bg-cyan-500 hover:bg-cyan-400 text-slate-950 font-bold transition-colors cursor-pointer"
          >
            Set
          </button>
          <button
            type="button"
            onClick={handleGetSubmit}
            className="py-2 px-3 rounded-xl bg-white/5 hover:bg-white/10 text-cyan-300 border border-cyan-500/30 transition-colors cursor-pointer"
            title="Read key and update recency/frequency"
          >
            Get
          </button>
          <button
            type="button"
            onClick={handleDeleteSubmit}
            className="py-2 px-2.5 rounded-xl bg-white/5 hover:bg-rose-500/20 text-rose-400 border border-white/10 transition-colors cursor-pointer"
            title="Delete key"
          >
            <Trash2 className="w-3.5 h-3.5" />
          </button>
        </div>
      </form>

      {/* Quick Action Buttons & Capacity Config */}
      <div className="pt-2 border-t border-white/5 flex flex-wrap items-center justify-between gap-3">
        {/* Capacity Slider / Selector */}
        <div className="flex items-center gap-2">
          <span className="text-[11px] text-slate-400">Capacity:</span>
          <div className="flex items-center gap-1">
            {[4, 6, 8, 10].map((cap) => (
              <button
                key={cap}
                type="button"
                onClick={() => onCapacityChange(cap)}
                className={`px-2.5 py-1 rounded-lg text-xs font-bold transition-colors cursor-pointer ${
                  capacity === cap
                    ? 'bg-cyan-500 text-slate-950'
                    : 'bg-white/5 text-slate-400 hover:text-white'
                }`}
              >
                {cap}
              </button>
            ))}
          </div>
        </div>

        {/* Quick Operations */}
        <div className="flex items-center gap-2 flex-wrap">
          <button
            type="button"
            onClick={onFillCapacity}
            disabled={currentSize >= capacity}
            className="flex items-center gap-1 px-3 py-1.5 rounded-xl bg-emerald-500/10 hover:bg-emerald-500/20 border border-emerald-500/30 text-emerald-300 transition-colors cursor-pointer disabled:opacity-40"
          >
            <PlusCircle className="w-3.5 h-3.5" />
            <span>Fill to Capacity</span>
          </button>

          <button
            type="button"
            onClick={() => onSet(`overflow:${Date.now().toString().slice(-4)}`, 'new_data')}
            className="flex items-center gap-1 px-3 py-1.5 rounded-xl bg-amber-500/10 hover:bg-amber-500/20 border border-amber-500/30 text-amber-300 transition-colors cursor-pointer"
          >
            <Zap className="w-3.5 h-3.5" />
            <span>Force Eviction (+1)</span>
          </button>

          <button
            type="button"
            onClick={onClear}
            className="flex items-center gap-1 px-3 py-1.5 rounded-xl bg-white/5 hover:bg-white/10 text-slate-400 hover:text-white transition-colors cursor-pointer"
          >
            <RotateCcw className="w-3.5 h-3.5" />
            <span>Reset</span>
          </button>
        </div>
      </div>
    </div>
  );
};
