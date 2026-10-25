import React from 'react';
import { motion } from 'framer-motion';
import { Badge } from '../shared/Badge';
import { CodeBlock } from '../shared/CodeBlock';
import { getNodeColor } from '../../core/constants';
import type { RequestLog } from '../../core/types';
import { ArrowRight, Server, Terminal, Zap, CheckCircle2, XCircle } from 'lucide-react';

interface ResponsePanelProps {
  log: RequestLog | null;
  isExecuting?: boolean;
}

export const ResponsePanel: React.FC<ResponsePanelProps> = ({ log, isExecuting }) => {
  if (isExecuting) {
    return (
      <div className="h-60 flex flex-col items-center justify-center p-6 text-center border border-dashed border-white/10 rounded-2xl bg-white/[0.01]">
        <Zap className="w-8 h-8 text-cyan-400 mb-2 animate-bounce" />
        <p className="text-sm font-medium text-slate-200">Proxying Request via Consistent Hash Ring...</p>
        <p className="text-xs text-slate-400 mt-1 font-mono">Routing key to target backend node...</p>
      </div>
    );
  }

  if (!log) {
    return (
      <div className="h-60 flex flex-col items-center justify-center p-6 text-center border border-dashed border-white/10 rounded-2xl bg-white/[0.01]">
        <Terminal className="w-8 h-8 text-slate-500 mb-2 opacity-50" />
        <p className="text-sm font-medium text-slate-300">Awaiting Request Execution</p>
        <p className="text-xs text-slate-500 mt-1 max-w-xs font-mono">
          Execute a GET, PUT, or DELETE request above to inspect the status code, routing trace, and response payload.
        </p>
      </div>
    );
  }

  const isSuccess = log.status >= 200 && log.status < 300;
  const isMiss = log.status === 404;
  const nodeColor = log.routedToNodeId ? getNodeColor(log.routedToNodeId) : null;

  // Format response body JSON
  let formattedBody = '';
  if (log.value) {
    try {
      formattedBody = JSON.stringify(JSON.parse(log.value), null, 2);
    } catch {
      formattedBody = JSON.stringify({ key: log.key, value: log.value }, null, 2);
    }
  } else if (log.statusText) {
    formattedBody = JSON.stringify(
      {
        status: log.status,
        message: log.statusText,
        key: log.key,
        hit: log.isHit ?? false,
      },
      null,
      2
    );
  }

  return (
    <motion.div
      key={log.id}
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2 }}
      className="space-y-4 font-mono text-xs"
    >
      {/* Top Status Header */}
      <div className="flex flex-wrap items-center justify-between gap-3 pb-3 border-b border-white/10">
        <div className="flex items-center gap-2">
          {isSuccess ? (
            <Badge variant="hit" size="md">
              <CheckCircle2 className="w-3.5 h-3.5" />
              <span>{log.status} {log.method === 'GET' ? 'CACHE HIT' : 'SUCCESS'}</span>
            </Badge>
          ) : isMiss ? (
            <Badge variant="miss" size="md">
              <XCircle className="w-3.5 h-3.5" />
              <span>404 CACHE MISS</span>
            </Badge>
          ) : (
            <Badge variant="evicted" size="md">
              <span>{log.status} ERROR</span>
            </Badge>
          )}

          <span className="px-2.5 py-0.5 rounded-full bg-slate-800 text-slate-300 text-[11px] border border-white/5">
            {log.latencyMs}ms
          </span>
        </div>

        <span className="text-[11px] text-slate-400">
          {new Date(log.timestamp).toLocaleTimeString()}
        </span>
      </div>

      {/* Network Proxy Hop Trace */}
      <div className="p-3.5 rounded-xl bg-slate-900/80 border border-white/10 space-y-2">
        <span className="text-[10px] uppercase text-slate-400 font-semibold tracking-wider">
          Request Routing Hops
        </span>
        <div className="flex items-center gap-2 overflow-x-auto py-1 text-xs">
          {/* Hop 1: Client */}
          <div className="px-2.5 py-1 rounded-lg bg-white/5 border border-white/10 text-slate-300 flex items-center gap-1.5 whitespace-nowrap">
            <span>Client</span>
          </div>

          <ArrowRight className="w-3.5 h-3.5 text-slate-500 shrink-0" />

          {/* Hop 2: Reverse Proxy Router */}
          <div className="px-2.5 py-1 rounded-lg bg-cyan-500/10 border border-cyan-500/30 text-cyan-300 flex items-center gap-1.5 whitespace-nowrap">
            <Zap className="w-3.5 h-3.5" />
            <span>Router (:9000)</span>
          </div>

          <ArrowRight className="w-3.5 h-3.5 text-slate-500 shrink-0" />

          {/* Hop 3: Destination Node */}
          <div
            className="px-2.5 py-1 rounded-lg border flex items-center gap-1.5 whitespace-nowrap font-bold"
            style={{
              backgroundColor: nodeColor ? `${nodeColor.bg}15` : 'rgba(255,255,255,0.05)',
              borderColor: nodeColor ? `${nodeColor.bg}40` : 'rgba(255,255,255,0.1)',
              color: nodeColor ? nodeColor.bg : '#ffffff',
            }}
          >
            <Server className="w-3.5 h-3.5" />
            <span>{log.routedToNodeId || 'Backend Node'} ({log.routedToAddress || ':8080'})</span>
          </div>
        </div>
      </div>

      {/* Response Payload */}
      <div>
        <span className="text-[10px] uppercase text-slate-400 font-semibold tracking-wider block mb-1">
          Response Body
        </span>
        <CodeBlock code={formattedBody} language="json" />
      </div>
    </motion.div>
  );
};
