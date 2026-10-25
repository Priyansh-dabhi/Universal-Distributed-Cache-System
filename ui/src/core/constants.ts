export const APP_CONFIG = {
  name: 'Distributed Cache Studio',
  version: '1.0.0',
  githubUrl: 'https://github.com/Priyansh-dabhi/Universal-Distributed-Cache-System',
  defaultRouterUrl: 'http://127.0.0.1:9000',
  defaultReplicas: 50,
};

export const NODE_COLORS: Record<string, { bg: string; border: string; glow: string; text: string }> = {
  'node-1': {
    bg: '#06b6d4',      // Cyan
    border: 'border-cyan-500/50',
    glow: 'rgba(6, 182, 212, 0.4)',
    text: 'text-cyan-400',
  },
  'node-2': {
    bg: '#8b5cf6',      // Violet
    border: 'border-violet-500/50',
    glow: 'rgba(139, 92, 246, 0.4)',
    text: 'text-violet-400',
  },
  'node-3': {
    bg: '#10b981',      // Emerald
    border: 'border-emerald-500/50',
    glow: 'rgba(16, 185, 129, 0.4)',
    text: 'text-emerald-400',
  },
  'node-4': {
    bg: '#f59e0b',      // Amber
    border: 'border-amber-500/50',
    glow: 'rgba(245, 158, 11, 0.4)',
    text: 'text-amber-400',
  },
  'node-5': {
    bg: '#ec4899',      // Pink
    border: 'border-pink-500/50',
    glow: 'rgba(236, 72, 153, 0.4)',
    text: 'text-pink-400',
  },
};

export const COLOR_PALETTE = [
  '#06b6d4', '#8b5cf6', '#10b981', '#f59e0b', '#ec4899', '#3b82f6', '#14b8a6', '#f97316'
];

export function getNodeColor(nodeId: string, index = 0): { bg: string; border: string; glow: string; text: string } {
  if (NODE_COLORS[nodeId]) {
    return NODE_COLORS[nodeId];
  }
  const color = COLOR_PALETTE[index % COLOR_PALETTE.length];
  return {
    bg: color,
    border: 'border-slate-500/50',
    glow: 'rgba(100, 116, 139, 0.4)',
    text: 'text-slate-300',
  };
}

export const TABS = [
  { id: 'ring', label: 'Consistent Hash Ring', path: '/ring', icon: 'CircleDot' },
  { id: 'playground', label: 'CRUD Studio', path: '/playground', icon: 'Terminal' },
  { id: 'eviction', label: 'Eviction Engines', path: '/eviction', icon: 'Layers' },
  { id: 'metrics', label: 'Live Telemetry', path: '/metrics', icon: 'Activity' },
  { id: 'learn', label: 'Architecture Docs', path: '/learn', icon: 'BookOpen' },
] as const;

export type TabId = typeof TABS[number]['id'];
