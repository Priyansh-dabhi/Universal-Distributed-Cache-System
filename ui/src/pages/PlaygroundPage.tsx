import React, { useState } from 'react';
import { useCache } from '../hooks/useCache';
import { useBackend } from '../hooks/useBackend';
import { RequestBar } from '../components/crud/RequestBar';
import type { HttpMethod } from '../components/crud/RequestBar';
import { ResponsePanel } from '../components/crud/ResponsePanel';
import { NodeCardGrid } from '../components/crud/NodeCardGrid';
import { NodeDetailDrawer } from '../components/crud/NodeDetailDrawer';
import { RequestHistoryTable } from '../components/crud/RequestHistoryTable';
import { GlassCard } from '../components/shared/GlassCard';
import { Badge } from '../components/shared/Badge';
import { AnimatedCounter } from '../components/shared/AnimatedCounter';
import { Terminal, Server, Activity, Database } from 'lucide-react';
import type { CacheEntry, NodeInfo, RequestLog } from '../core/types';

export const PlaygroundPage: React.FC = () => {
  const { mode, simRouter } = useBackend();
  const { get, set, deleteKey, history, lastLog, isExecuting, clearHistory, nodes } = useCache();

  const [selectedLog, setSelectedLog] = useState<RequestLog | null>(null);
  const [inspectNodeId, setInspectNodeId] = useState<string | null>(null);

  // Determine current active log to display in response panel
  const activeDisplayLog = selectedLog || lastLog;

  const handleExecute = async (method: HttpMethod, key: string, value?: string, ttl?: string) => {
    setSelectedLog(null);
    if (method === 'GET') {
      await get(key);
    } else if (method === 'PUT') {
      await set(key, value || '', ttl);
    } else if (method === 'DELETE') {
      await deleteKey(key);
    }
  };

  // Inspect node resident keys
  const inspectedNodeInfo: NodeInfo | null = inspectNodeId
    ? nodes.find((n) => n.id === inspectNodeId) || null
    : null;

  let residentKeys: CacheEntry[] = [];
  if (inspectNodeId) {
    if (mode === 'sim') {
      const node = simRouter.getNode(inspectNodeId);
      residentKeys = node ? node.getEntries() : [];
    } else {
      // In live mode, collect known keys routed to this node from history
      residentKeys = history
        .filter((h) => h.routedToNodeId === inspectNodeId && h.isHit && h.value)
        .map((h) => ({
          key: h.key,
          value: h.value || '',
          createdAt: h.timestamp,
        }));
    }
  }

  // Calculate live session hit rate
  const gets = history.filter((h) => h.method === 'GET');
  const hits = gets.filter((h) => h.isHit);
  const sessionHitRate = gets.length > 0 ? Math.round((hits.length / gets.length) * 100) : 0;

  return (
    <div className="space-y-6">
      {/* Page Title & Status Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <div className="flex items-center gap-2.5">
            <h2 className="text-xl sm:text-2xl font-bold text-white tracking-tight flex items-center gap-2">
              <Terminal className="w-6 h-6 text-emerald-400" />
              CRUD Studio & Cluster Control
            </h2>
            <Badge variant={mode === 'live' ? 'live' : 'sim'} pulse>
              {mode === 'live' ? 'Live Cluster' : 'Web Simulator'}
            </Badge>
          </div>
          <p className="text-xs sm:text-sm text-slate-400 mt-1">
            Dispatch GET, PUT, and DELETE operations, view end-to-end proxy routing traces, and inspect per-node resident memory.
          </p>
        </div>

        {/* Quick Session Stats */}
        <div className="flex items-center gap-2 font-mono text-xs">
          <div className="px-3 py-1.5 rounded-xl bg-slate-900/80 border border-white/10 flex items-center gap-2">
            <Server className="w-3.5 h-3.5 text-cyan-400" />
            <span className="text-slate-400">Nodes:</span>
            <span className="font-bold text-white">{nodes.length}</span>
          </div>
          <div className="px-3 py-1.5 rounded-xl bg-slate-900/80 border border-white/10 flex items-center gap-2">
            <Activity className="w-3.5 h-3.5 text-emerald-400" />
            <span className="text-slate-400">Hit Rate:</span>
            <span className="font-bold text-white">{sessionHitRate}%</span>
          </div>
          <div className="px-3 py-1.5 rounded-xl bg-slate-900/80 border border-white/10 flex items-center gap-2">
            <Database className="w-3.5 h-3.5 text-violet-400" />
            <span className="text-slate-400">Total Ops:</span>
            <AnimatedCounter value={history.length} className="font-bold text-white" />
          </div>
        </div>
      </div>

      {/* Main Studio 2-Column Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
        {/* Left Column: Request Dispatcher & Live Response Panel */}
        <div className="lg:col-span-7 space-y-6">
          <GlassCard
            title="HTTP Request Dispatcher"
            subtitle="Forward requests to router (:9000) for consistent hash proxying"
            icon={<Terminal className="w-5 h-5 text-emerald-400" />}
          >
            <RequestBar
              onExecute={handleExecute}
              isExecuting={isExecuting}
              knownKeys={Array.from(new Set(history.map((h) => h.key)))}
            />
          </GlassCard>

          <GlassCard
            title="Real-Time Response Inspector"
            subtitle="Status code, proxy hops, and returned payload"
            icon={<Activity className="w-5 h-5 text-cyan-400" />}
          >
            <ResponsePanel log={activeDisplayLog} isExecuting={isExecuting} />
          </GlassCard>
        </div>

        {/* Right Column: Physical Node Cards & Inspection */}
        <div className="lg:col-span-5 space-y-4">
          <div className="flex items-center justify-between">
            <h3 className="font-bold text-white text-sm tracking-wide flex items-center gap-2">
              <Server className="w-4 h-4 text-cyan-400" />
              <span>Registered Cache Nodes ({nodes.length})</span>
            </h3>
            <span className="text-[11px] font-mono text-slate-400">Click inspect to browse keys</span>
          </div>

          <NodeCardGrid
            nodes={nodes}
            lastRoutedNodeId={lastLog?.routedToNodeId}
            onInspectNode={(nodeId) => setInspectNodeId(nodeId)}
          />
        </div>
      </div>

      {/* Bottom Section: Chronological Request History */}
      <RequestHistoryTable
        history={history}
        onSelectLog={(log) => setSelectedLog(log)}
        onClearHistory={clearHistory}
      />

      {/* Node Resident Keys Drawer */}
      <NodeDetailDrawer
        isOpen={!!inspectNodeId}
        onClose={() => setInspectNodeId(null)}
        node={inspectedNodeInfo}
        entries={residentKeys}
        onGetKey={async (k) => {
          setInspectNodeId(null);
          await get(k);
        }}
        onDeleteKey={async (k) => {
          await deleteKey(k);
        }}
      />
    </div>
  );
};
