import React, { useState } from 'react';
import { Send, Clock, KeyRound, FileText } from 'lucide-react';
import { clsx } from 'clsx';

export type HttpMethod = 'GET' | 'PUT' | 'DELETE';

interface RequestBarProps {
  onExecute: (method: HttpMethod, key: string, value?: string, ttl?: string) => Promise<void>;
  isExecuting: boolean;
  knownKeys?: string[];
}

const TTL_OPTIONS = [
  { label: 'No TTL (Persistent)', value: '' },
  { label: '5 seconds', value: '5s' },
  { label: '10 seconds', value: '10s' },
  { label: '30 seconds', value: '30s' },
  { label: '1 minute', value: '1m' },
  { label: '5 minutes', value: '5m' },
];

export const RequestBar: React.FC<RequestBarProps> = ({
  onExecute,
  isExecuting,
  knownKeys = ['user:101', 'session:tok_99', 'order:5432', 'cart:alice'],
}) => {
  const [method, setMethod] = useState<HttpMethod>('PUT');
  const [key, setKey] = useState('user:101');
  const [value, setValue] = useState('{"id": 101, "name": "Alice Cooper", "role": "admin"}');
  const [ttl, setTtl] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!key.trim()) return;
    await onExecute(method, key.trim(), method === 'PUT' ? value : undefined, ttl || undefined);
  };

  const methodColors: Record<HttpMethod, { active: string; idle: string }> = {
    GET: {
      active: 'bg-cyan-500 text-slate-950 shadow-md shadow-cyan-500/20 font-bold',
      idle: 'text-cyan-400 hover:bg-cyan-500/10 border-cyan-500/30',
    },
    PUT: {
      active: 'bg-emerald-500 text-slate-950 shadow-md shadow-emerald-500/20 font-bold',
      idle: 'text-emerald-400 hover:bg-emerald-500/10 border-emerald-500/30',
    },
    DELETE: {
      active: 'bg-rose-500 text-slate-950 shadow-md shadow-rose-500/20 font-bold',
      idle: 'text-rose-400 hover:bg-rose-500/10 border-rose-500/30',
    },
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-3 font-mono">
      {/* Top Controls: Method Selector & Quick Key Presets */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        {/* Method Pills */}
        <div className="flex items-center gap-1.5 p-1 rounded-xl bg-slate-950/80 border border-white/10">
          {(['GET', 'PUT', 'DELETE'] as HttpMethod[]).map((m) => (
            <button
              key={m}
              type="button"
              onClick={() => setMethod(m)}
              className={clsx(
                'px-3 py-1 rounded-lg text-xs tracking-wider transition-all cursor-pointer font-mono',
                method === m ? methodColors[m].active : `border border-transparent ${methodColors[m].idle}`
              )}
            >
              {m}
            </button>
          ))}
        </div>

        {/* Quick Suggestion Pills */}
        <div className="flex items-center gap-1.5 flex-wrap">
          <span className="text-[10px] text-slate-400">Sample Keys:</span>
          {knownKeys.slice(0, 4).map((k) => (
            <button
              key={k}
              type="button"
              onClick={() => setKey(k)}
              className="text-[10px] px-2 py-0.5 rounded-md bg-white/5 hover:bg-white/10 text-slate-300 border border-white/5 transition-colors cursor-pointer"
            >
              {k}
            </button>
          ))}
        </div>
      </div>

      {/* Inputs Section */}
      <div className="grid grid-cols-1 md:grid-cols-12 gap-3 items-start">
        {/* Key Input */}
        <div className={clsx(method === 'PUT' ? 'md:col-span-5' : 'md:col-span-8')}>
          <label className="block text-[11px] text-slate-400 mb-1 flex items-center gap-1.5">
            <KeyRound className="w-3.5 h-3.5 text-cyan-400" />
            <span>Cache Key</span>
          </label>
          <input
            type="text"
            value={key}
            onChange={(e) => setKey(e.target.value)}
            placeholder="e.g. user:101 or session:token"
            required
            className="w-full px-3.5 py-2 rounded-xl bg-slate-900 border border-white/10 text-xs text-white placeholder-slate-500 focus:outline-none focus:border-cyan-500/50"
          />
        </div>

        {/* TTL Selector */}
        <div className={clsx(method === 'PUT' ? 'md:col-span-3' : 'md:col-span-4')}>
          <label className="block text-[11px] text-slate-400 mb-1 flex items-center gap-1.5">
            <Clock className="w-3.5 h-3.5 text-amber-400" />
            <span>TTL (Time-To-Live)</span>
          </label>
          <select
            value={ttl}
            onChange={(e) => setTtl(e.target.value)}
            disabled={method !== 'PUT'}
            className="w-full px-3 py-2 rounded-xl bg-slate-900 border border-white/10 text-xs text-slate-200 focus:outline-none focus:border-cyan-500/50 disabled:opacity-40"
          >
            {TTL_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        </div>

        {/* Value Textarea (visible for PUT) */}
        {method === 'PUT' && (
          <div className="md:col-span-4">
            <label className="block text-[11px] text-slate-400 mb-1 flex items-center gap-1.5">
              <FileText className="w-3.5 h-3.5 text-emerald-400" />
              <span>Payload Value (JSON / String)</span>
            </label>
            <input
              type="text"
              value={value}
              onChange={(e) => setValue(e.target.value)}
              placeholder='{"data": "value"}'
              required
              className="w-full px-3.5 py-2 rounded-xl bg-slate-900 border border-white/10 text-xs text-white placeholder-slate-500 focus:outline-none focus:border-emerald-500/50"
            />
          </div>
        )}
      </div>

      {/* Submit Button */}
      <div className="flex justify-end pt-1">
        <button
          type="submit"
          disabled={isExecuting || !key.trim()}
          className={clsx(
            'px-5 py-2.5 rounded-xl font-bold text-xs flex items-center gap-2 transition-all cursor-pointer disabled:opacity-50 text-slate-950',
            method === 'GET' && 'bg-cyan-400 hover:bg-cyan-300 shadow-lg shadow-cyan-500/20',
            method === 'PUT' && 'bg-emerald-400 hover:bg-emerald-300 shadow-lg shadow-emerald-500/20',
            method === 'DELETE' && 'bg-rose-400 hover:bg-rose-300 shadow-lg shadow-rose-500/20'
          )}
        >
          <Send className={clsx('w-3.5 h-3.5', isExecuting && 'animate-spin')} />
          <span>{isExecuting ? 'Dispatching...' : `SEND ${method} REQUEST`}</span>
        </button>
      </div>
    </form>
  );
};
