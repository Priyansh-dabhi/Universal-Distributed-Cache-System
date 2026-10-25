import React, { useState } from 'react';
import { Search, Plus, Trash2, Eye, EyeOff, Layers, Play } from 'lucide-react';
import type { EvictionPolicy, RingNode } from '../../core/types';
import { getNodeColor } from '../../core/constants';

interface RingControlsProps {
  nodes: RingNode[];
  replicas: number;
  showVirtualNodes: boolean;
  onToggleVirtualNodes: () => void;
  onRouteKey: (key: string) => void;
  onAddNode: (node: { id: string; address?: string; capacity?: number; policy?: EvictionPolicy }, color?: string) => void;
  onRemoveNode: (nodeId: string) => void;
  isLiveMode: boolean;
}

const PRESET_KEYS = [
  'user:101',
  'session:tok_abc',
  'order:9872',
  'product:laptop',
  'cache:zipfian_1',
  'analytics:pageview',
];

export const RingControls: React.FC<RingControlsProps> = ({
  nodes,
  replicas,
  showVirtualNodes,
  onToggleVirtualNodes,
  onRouteKey,
  onAddNode,
  onRemoveNode,
  isLiveMode,
}) => {
  const [inputKey, setInputKey] = useState('user:101');
  const [showAddModal, setShowAddModal] = useState(false);
  const [newNodeId, setNewNodeId] = useState('');
  const [newNodePolicy, setNewNodePolicy] = useState<EvictionPolicy>('LRU');

  const handleRoute = (e?: React.FormEvent) => {
    if (e) e.preventDefault();
    if (inputKey.trim()) {
      onRouteKey(inputKey.trim());
    }
  };

  const handlePresetClick = (preset: string) => {
    setInputKey(preset);
    onRouteKey(preset);
  };

  const handleAddSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const id = newNodeId.trim() || `node-${nodes.length + 1}`;
    onAddNode({
      id,
      capacity: 50,
      policy: newNodePolicy,
    });
    setNewNodeId('');
    setShowAddModal(false);
  };

  return (
    <div className="space-y-4">
      {/* Search Input Bar */}
      <form onSubmit={handleRoute} className="flex gap-2">
        <div className="relative flex-1">
          <Search className="w-4 h-4 absolute left-3.5 top-3 text-slate-400" />
          <input
            type="text"
            value={inputKey}
            onChange={(e) => setInputKey(e.target.value)}
            placeholder="Enter cache key (e.g. user:42, session:token)..."
            className="w-full pl-10 pr-4 py-2 bg-slate-900/90 border border-white/10 rounded-xl text-xs font-mono text-slate-100 placeholder-slate-500 focus:outline-none focus:border-cyan-500/50 focus:ring-1 focus:ring-cyan-500/30 transition-all"
          />
        </div>
        <button
          type="submit"
          className="px-4 py-2 rounded-xl bg-gradient-to-r from-cyan-500 to-cyan-600 hover:from-cyan-400 hover:to-cyan-500 text-slate-950 font-semibold text-xs flex items-center gap-1.5 shadow-md shadow-cyan-500/20 transition-all cursor-pointer"
        >
          <Play className="w-3.5 h-3.5 fill-current" />
          <span>Route</span>
        </button>
      </form>

      {/* Preset Pills */}
      <div className="flex items-center gap-1.5 flex-wrap">
        <span className="text-[11px] font-mono text-slate-400 mr-1">Presets:</span>
        {PRESET_KEYS.map((preset) => (
          <button
            key={preset}
            onClick={() => handlePresetClick(preset)}
            className="px-2.5 py-1 rounded-lg bg-white/5 hover:bg-white/10 text-[11px] font-mono text-slate-300 border border-white/5 transition-colors cursor-pointer hover:border-cyan-500/30"
          >
            {preset}
          </button>
        ))}
      </div>

      {/* Control Toggles & Cluster Actions */}
      <div className="pt-2 border-t border-white/5 flex flex-wrap items-center justify-between gap-3">
        {/* Toggle Virtual Nodes */}
        <button
          onClick={onToggleVirtualNodes}
          className={`flex items-center gap-2 px-3 py-1.5 rounded-xl text-xs font-medium border transition-colors cursor-pointer ${
            showVirtualNodes
              ? 'bg-cyan-500/10 border-cyan-500/30 text-cyan-300'
              : 'bg-white/5 border-white/10 text-slate-400 hover:text-slate-200'
          }`}
        >
          {showVirtualNodes ? <Eye className="w-3.5 h-3.5" /> : <EyeOff className="w-3.5 h-3.5" />}
          <span>{showVirtualNodes ? 'Tokens Visible' : 'Tokens Hidden'} ({nodes.length * replicas})</span>
        </button>

        {/* Cluster Node Badges & Management */}
        <div className="flex items-center gap-2 flex-wrap">
          {nodes.map((n) => {
            const color = getNodeColor(n.id);
            return (
              <div
                key={n.id}
                className="flex items-center gap-1.5 px-2.5 py-1 rounded-lg border text-xs font-mono"
                style={{
                  backgroundColor: `${color.bg}10`,
                  borderColor: `${color.bg}30`,
                  color: color.bg,
                }}
              >
                <span className="w-1.5 h-1.5 rounded-full" style={{ backgroundColor: color.bg }} />
                <span>{n.id}</span>
                {!isLiveMode && nodes.length > 1 && (
                  <button
                    onClick={() => onRemoveNode(n.id)}
                    className="ml-1 opacity-60 hover:opacity-100 hover:text-rose-400 cursor-pointer"
                    title={`Remove ${n.id} from cluster`}
                  >
                    <Trash2 className="w-3 h-3" />
                  </button>
                )}
              </div>
            );
          })}

          {/* Add Node Button (Sim Mode only) */}
          {!isLiveMode && nodes.length < 8 && (
            <button
              onClick={() => setShowAddModal(true)}
              className="flex items-center gap-1 px-2.5 py-1 rounded-lg bg-emerald-500/10 hover:bg-emerald-500/20 border border-emerald-500/30 text-emerald-400 text-xs font-medium transition-colors cursor-pointer"
              title="Add a node to the cluster"
            >
              <Plus className="w-3.5 h-3.5" />
              <span>Add Node</span>
            </button>
          )}
        </div>
      </div>

      {/* Add Node Simple Modal */}
      {showAddModal && (
        <div className="p-3.5 rounded-xl border border-emerald-500/30 bg-slate-900/90 backdrop-blur-md space-y-3">
          <div className="flex items-center justify-between text-xs font-mono font-semibold text-emerald-400">
            <span className="flex items-center gap-1.5">
              <Layers className="w-4 h-4" />
              Add Node to Hash Ring
            </span>
            <button
              onClick={() => setShowAddModal(false)}
              className="text-slate-400 hover:text-white"
            >
              &times;
            </button>
          </div>
          <form onSubmit={handleAddSubmit} className="space-y-2">
            <div>
              <label className="text-[10px] font-mono text-slate-400">Node ID</label>
              <input
                type="text"
                value={newNodeId}
                onChange={(e) => setNewNodeId(e.target.value)}
                placeholder={`node-${nodes.length + 1}`}
                className="w-full px-3 py-1.5 bg-slate-950 border border-white/10 rounded-lg text-xs font-mono text-white focus:outline-none focus:border-emerald-500/50"
              />
            </div>
            <div>
              <label className="text-[10px] font-mono text-slate-400">Eviction Policy</label>
              <select
                value={newNodePolicy}
                onChange={(e) => setNewNodePolicy(e.target.value as EvictionPolicy)}
                className="w-full px-3 py-1.5 bg-slate-950 border border-white/10 rounded-lg text-xs font-mono text-white focus:outline-none focus:border-emerald-500/50"
              >
                <option value="LRU">LRU (Least Recently Used)</option>
                <option value="LFU">LFU (Least Frequently Used)</option>
                <option value="2Q">2Q (Scan-Resistant Two-Queue)</option>
              </select>
            </div>
            <div className="flex justify-end gap-2 pt-1">
              <button
                type="button"
                onClick={() => setShowAddModal(false)}
                className="px-3 py-1 rounded-lg bg-white/5 text-slate-400 text-xs"
              >
                Cancel
              </button>
              <button
                type="submit"
                className="px-3.5 py-1 rounded-lg bg-emerald-500 text-slate-950 font-semibold text-xs"
              >
                Confirm Add
              </button>
            </div>
          </form>
        </div>
      )}
    </div>
  );
};
