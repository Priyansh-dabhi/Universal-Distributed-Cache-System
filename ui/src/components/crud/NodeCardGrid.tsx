import React from 'react';
import { GlassCard } from '../shared/GlassCard';
import { Badge } from '../shared/Badge';
import { getNodeColor } from '../../core/constants';
import type { NodeInfo } from '../../core/types';
import { Server, Eye, Layers } from 'lucide-react';

interface NodeCardGridProps {
  nodes: NodeInfo[];
  lastRoutedNodeId?: string;
  onInspectNode: (nodeId: string) => void;
}

export const NodeCardGrid: React.FC<NodeCardGridProps> = ({
  nodes,
  lastRoutedNodeId,
  onInspectNode,
}) => {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {nodes.map((node) => {
        const color = getNodeColor(node.id);
        const isRecentTarget = lastRoutedNodeId === node.id;
        const usedPercent = Math.min(100, Math.round((node.size / node.capacity) * 100));

        return (
          <GlassCard
            key={node.id}
            hoverEffect
            className={`transition-all duration-300 ${
              isRecentTarget ? 'ring-2 ring-cyan-400 shadow-[0_0_20px_rgba(6,182,212,0.25)]' : ''
            }`}
            action={
              <button
                onClick={() => onInspectNode(node.id)}
                className="flex items-center gap-1 px-2.5 py-1 rounded-lg bg-white/5 hover:bg-white/10 text-slate-300 hover:text-white text-xs font-mono border border-white/5 transition-colors cursor-pointer"
                title="View resident keys on this node"
              >
                <Eye className="w-3.5 h-3.5 text-cyan-400" />
                <span>Inspect</span>
              </button>
            }
            title={
              <div className="flex items-center gap-2">
                <span
                  className="w-2.5 h-2.5 rounded-full"
                  style={{ backgroundColor: color.bg }}
                />
                <span className="font-mono font-bold text-sm text-white tracking-wide">
                  {node.id}
                </span>
                {isRecentTarget && (
                  <Badge variant="cyan" size="sm" pulse>
                    ACTIVE
                  </Badge>
                )}
              </div>
            }
            subtitle={node.address}
            icon={<Server className="w-4 h-4" style={{ color: color.bg }} />}
          >
            <div className="space-y-3 font-mono text-xs pt-1">
              {/* Policy & Status */}
              <div className="flex items-center justify-between">
                <span className="text-slate-400 text-[11px] flex items-center gap-1">
                  <Layers className="w-3 h-3 text-slate-400" />
                  Policy
                </span>
                <span
                  className="px-2 py-0.5 rounded-full text-[11px] font-bold border"
                  style={{
                    backgroundColor: `${color.bg}15`,
                    borderColor: `${color.bg}30`,
                    color: color.bg,
                  }}
                >
                  {node.policy}
                </span>
              </div>

              {/* Memory Capacity Bar */}
              <div className="space-y-1">
                <div className="flex items-center justify-between text-[11px]">
                  <span className="text-slate-400">Capacity Usage</span>
                  <span className="text-slate-200 font-bold">
                    {node.size} / {node.capacity} slots ({usedPercent}%)
                  </span>
                </div>
                <div className="w-full h-2 rounded-full bg-slate-900 border border-white/5 overflow-hidden">
                  <div
                    className="h-full rounded-full transition-all duration-500"
                    style={{
                      width: `${usedPercent}%`,
                      backgroundColor: color.bg,
                    }}
                  />
                </div>
              </div>
            </div>
          </GlassCard>
        );
      })}
    </div>
  );
};
