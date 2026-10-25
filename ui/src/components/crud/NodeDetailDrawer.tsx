import React from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { getNodeColor } from '../../core/constants';
import type { CacheEntry, NodeInfo } from '../../core/types';
import { X, Server, Trash2, Search, KeyRound, Clock, Layers } from 'lucide-react';

interface NodeDetailDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  node: NodeInfo | null;
  entries: CacheEntry[];
  onGetKey: (key: string) => void;
  onDeleteKey: (key: string) => void;
}

export const NodeDetailDrawer: React.FC<NodeDetailDrawerProps> = ({
  isOpen,
  onClose,
  node,
  entries,
  onGetKey,
  onDeleteKey,
}) => {
  const [mountTime] = React.useState(() => Date.now());

  if (!node) return null;

  const color = getNodeColor(node.id);

  return (
    <AnimatePresence>
      {isOpen && (
        <>
          {/* Backdrop */}
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            onClick={onClose}
            className="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm"
          />

          {/* Slide-over Drawer Panel */}
          <motion.div
            initial={{ x: '100%' }}
            animate={{ x: 0 }}
            exit={{ x: '100%' }}
            transition={{ type: 'spring', damping: 30, stiffness: 300 }}
            className="fixed inset-y-0 right-0 z-50 w-full max-w-md bg-slate-950/95 border-l border-white/10 shadow-2xl flex flex-col font-mono"
          >
            {/* Header */}
            <div className="p-5 border-b border-white/10 flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div
                  className="p-2 rounded-xl border"
                  style={{
                    backgroundColor: `${color.bg}15`,
                    borderColor: `${color.bg}30`,
                    color: color.bg,
                  }}
                >
                  <Server className="w-5 h-5" />
                </div>
                <div>
                  <h3 className="font-bold text-white text-base">{node.id} Inspector</h3>
                  <p className="text-xs text-slate-400">{node.address} &bull; {node.policy}</p>
                </div>
              </div>
              <button
                onClick={onClose}
                className="p-2 rounded-xl text-slate-400 hover:text-white bg-white/5 hover:bg-white/10 transition-colors cursor-pointer"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            {/* Sub-header info */}
            <div className="px-5 py-3 bg-slate-900/60 border-b border-white/5 flex items-center justify-between text-xs">
              <span className="text-slate-400">Stored Keys: <strong className="text-white">{entries.length}</strong></span>
              <span className="text-slate-400">Capacity: <strong className="text-white">{node.capacity}</strong></span>
              <span className="text-slate-400">Policy: <strong className="text-white">{node.policy}</strong></span>
            </div>

            {/* Key List */}
            <div className="flex-1 overflow-y-auto p-5 space-y-3">
              {entries.length === 0 ? (
                <div className="h-60 flex flex-col items-center justify-center text-center p-6 border border-dashed border-white/10 rounded-2xl text-slate-400">
                  <KeyRound className="w-8 h-8 text-slate-500 mb-2 opacity-50" />
                  <p className="text-sm font-medium text-slate-300">No Resident Keys</p>
                  <p className="text-xs text-slate-500 mt-1 max-w-xs font-sans">
                    Use the CRUD studio to PUT keys into the cluster. The hash ring will route relevant keys here.
                  </p>
                </div>
              ) : (
                entries.map((entry) => {
                  const remainingTtl = entry.expiresAt ? Math.max(0, Math.round((entry.expiresAt - mountTime) / 1000)) : null;

                  return (
                    <div
                      key={entry.key}
                      className="p-3.5 rounded-xl bg-slate-900/80 border border-white/5 hover:border-white/15 transition-all space-y-2"
                    >
                      {/* Key & Action Row */}
                      <div className="flex items-center justify-between">
                        <span className="font-bold text-white text-xs truncate max-w-[200px]">
                          "{entry.key}"
                        </span>
                        <div className="flex items-center gap-1.5">
                          <button
                            onClick={() => onGetKey(entry.key)}
                            className="p-1 rounded-md text-cyan-400 hover:bg-cyan-500/10 transition-colors cursor-pointer"
                            title="Query Key"
                          >
                            <Search className="w-3.5 h-3.5" />
                          </button>
                          <button
                            onClick={() => onDeleteKey(entry.key)}
                            className="p-1 rounded-md text-rose-400 hover:bg-rose-500/10 transition-colors cursor-pointer"
                            title="Delete Key"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        </div>
                      </div>

                      {/* Value preview */}
                      <p className="text-xs text-slate-300 font-sans truncate bg-slate-950/60 px-2 py-1 rounded-md border border-white/5">
                        {entry.value}
                      </p>

                      {/* Metadata Row */}
                      <div className="flex items-center gap-3 text-[10px] text-slate-400">
                        {entry.expiresAt ? (
                          <span className="flex items-center gap-1 text-amber-400">
                            <Clock className="w-3 h-3" />
                            <span>TTL: {remainingTtl}s</span>
                          </span>
                        ) : (
                          <span className="text-slate-500">Persistent</span>
                        )}

                        {entry.freq !== undefined && (
                          <span className="text-violet-400">Freq: {entry.freq}&times;</span>
                        )}

                        {entry.qType && (
                          <span className="text-emerald-400 flex items-center gap-1">
                            <Layers className="w-3 h-3" />
                            <span>Queue: {entry.qType}</span>
                          </span>
                        )}
                      </div>
                    </div>
                  );
                })
              )}
            </div>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  );
};
