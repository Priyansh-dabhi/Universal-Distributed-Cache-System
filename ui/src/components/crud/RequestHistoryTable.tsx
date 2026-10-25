import React from 'react';
import { GlassCard } from '../shared/GlassCard';
import { Badge } from '../shared/Badge';
import { getNodeColor } from '../../core/constants';
import type { RequestLog } from '../../core/types';
import { History, Trash2, ArrowUpRight } from 'lucide-react';

interface RequestHistoryTableProps {
  history: RequestLog[];
  onSelectLog: (log: RequestLog) => void;
  onClearHistory: () => void;
}

export const RequestHistoryTable: React.FC<RequestHistoryTableProps> = ({
  history,
  onSelectLog,
  onClearHistory,
}) => {
  return (
    <GlassCard
      title="Cluster Request Log (Rolling 50 Ops)"
      subtitle="Complete chronological timeline of forwarded proxy operations"
      icon={<History className="w-5 h-5 text-cyan-400" />}
      action={
        history.length > 0 && (
          <button
            onClick={onClearHistory}
            className="flex items-center gap-1 px-2.5 py-1 rounded-lg bg-white/5 hover:bg-rose-500/10 text-slate-400 hover:text-rose-400 text-xs font-mono transition-colors cursor-pointer"
          >
            <Trash2 className="w-3.5 h-3.5" />
            <span>Clear History</span>
          </button>
        )
      }
    >
      {history.length === 0 ? (
        <div className="py-8 text-center text-slate-500 font-mono text-xs">
          No request history recorded yet. Execute operations above to populate telemetry.
        </div>
      ) : (
        <div className="overflow-x-auto no-scrollbar font-mono text-xs">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="border-b border-white/10 text-slate-400 text-[11px]">
                <th className="pb-2 font-medium">Time</th>
                <th className="pb-2 font-medium">Method</th>
                <th className="pb-2 font-medium">Key</th>
                <th className="pb-2 font-medium">Status</th>
                <th className="pb-2 font-medium">Routed Node</th>
                <th className="pb-2 font-medium text-right">Latency</th>
                <th className="pb-2 font-medium text-right">Action</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-white/5">
              {history.map((log) => {
                const nodeColor = log.routedToNodeId ? getNodeColor(log.routedToNodeId) : null;
                const isSuccess = log.status >= 200 && log.status < 300;

                return (
                  <tr
                    key={log.id}
                    className="hover:bg-white/[0.02] transition-colors group cursor-pointer"
                    onClick={() => onSelectLog(log)}
                  >
                    <td className="py-2.5 text-slate-400 text-[11px]">
                      {new Date(log.timestamp).toLocaleTimeString()}
                    </td>
                    <td className="py-2.5">
                      <span
                        className={`px-2 py-0.5 rounded text-[10px] font-bold ${
                          log.method === 'GET'
                            ? 'bg-cyan-500/15 text-cyan-400'
                            : log.method === 'PUT'
                            ? 'bg-emerald-500/15 text-emerald-400'
                            : 'bg-rose-500/15 text-rose-400'
                        }`}
                      >
                        {log.method}
                      </span>
                    </td>
                    <td className="py-2.5 font-bold text-white max-w-[150px] truncate">
                      "{log.key}"
                    </td>
                    <td className="py-2.5">
                      {isSuccess ? (
                        <Badge variant="hit" size="sm">
                          {log.status}
                        </Badge>
                      ) : log.status === 404 ? (
                        <Badge variant="miss" size="sm">
                          404
                        </Badge>
                      ) : (
                        <Badge variant="evicted" size="sm">
                          {log.status}
                        </Badge>
                      )}
                    </td>
                    <td className="py-2.5">
                      {log.routedToNodeId ? (
                        <span
                          className="px-2 py-0.5 rounded-full text-[10px] border font-bold"
                          style={{
                            backgroundColor: `${nodeColor?.bg}15`,
                            borderColor: `${nodeColor?.bg}30`,
                            color: nodeColor?.bg,
                          }}
                        >
                          {log.routedToNodeId}
                        </span>
                      ) : (
                        <span className="text-slate-500">-</span>
                      )}
                    </td>
                    <td className="py-2.5 text-right text-slate-300">
                      {log.latencyMs}ms
                    </td>
                    <td className="py-2.5 text-right">
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          onSelectLog(log);
                        }}
                        className="p-1 rounded text-slate-400 hover:text-white group-hover:bg-white/10 transition-colors"
                        title="View Details"
                      >
                        <ArrowUpRight className="w-3.5 h-3.5" />
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </GlassCard>
  );
};
