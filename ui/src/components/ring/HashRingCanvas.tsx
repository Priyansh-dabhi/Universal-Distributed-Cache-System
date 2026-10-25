import React, { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { getNodeColor } from '../../core/constants';
import type { RingNode, RingPosition, RouteResult } from '../../core/types';

interface HashRingCanvasProps {
  nodes: RingNode[];
  positions: RingPosition[];
  activeRoute: RouteResult | null;
  showVirtualNodes: boolean;
  onSelectNode?: (nodeId: string) => void;
}

export const HashRingCanvas: React.FC<HashRingCanvasProps> = ({
  nodes,
  positions,
  activeRoute,
  showVirtualNodes,
  onSelectNode,
}) => {
  const [hoveredToken, setHoveredToken] = useState<RingPosition | null>(null);

  // SVG Geometry
  const size = 520;
  const center = size / 2;
  const radius = 190;
  const trackWidth = 4;

  // Convert angle (0 to 360 deg, where 0 is at 12 o'clock, clockwise) to SVG coordinates
  const polarToCartesian = (angleDeg: number, r: number = radius) => {
    const angleRad = ((angleDeg - 90) * Math.PI) / 180;
    return {
      x: center + r * Math.cos(angleRad),
      y: center + r * Math.sin(angleRad),
    };
  };

  // Group positions by physical node to compute physical node anchor positions
  const physicalNodeAverages = nodes.map((node) => {
    const nodePositions = positions.filter((p) => p.nodeId === node.id);
    const avgAngle =
      nodePositions.length > 0
        ? nodePositions.reduce((acc, curr) => acc + curr.angle, 0) / nodePositions.length
        : 0;
    return {
      ...node,
      angle: avgAngle,
      count: nodePositions.length,
    };
  });

  // Calculate SVG arc path for routing sweep
  const getSweepArcPath = () => {
    if (!activeRoute) return null;

    const startAngle = activeRoute.angle;
    // Target position angle
    const targetAngle = ((activeRoute.targetPosition >>> 0) / 0xffffffff) * 360;

    const startPt = polarToCartesian(startAngle, radius);
    const endPt = polarToCartesian(targetAngle, radius);

    let sweepDiff = targetAngle - startAngle;
    if (sweepDiff < 0) sweepDiff += 360;

    const largeArcFlag = sweepDiff > 180 ? 1 : 0;

    return `M ${startPt.x} ${startPt.y} A ${radius} ${radius} 0 ${largeArcFlag} 1 ${endPt.x} ${endPt.y}`;
  };

  const keyCoord = activeRoute ? polarToCartesian(activeRoute.angle, radius) : null;
  const targetCoord = activeRoute
    ? polarToCartesian(((activeRoute.targetPosition >>> 0) / 0xffffffff) * 360, radius)
    : null;

  return (
    <div className="relative w-full aspect-square max-w-[520px] mx-auto select-none flex items-center justify-center">
      <svg
        viewBox={`0 0 ${size} ${size}`}
        className="w-full h-full overflow-visible drop-shadow-2xl"
      >
        <defs>
          {/* Circular gradient for outer ring track */}
          <linearGradient id="ringTrackGradient" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="#06b6d4" stopOpacity="0.4" />
            <stop offset="33%" stopColor="#8b5cf6" stopOpacity="0.4" />
            <stop offset="66%" stopColor="#10b981" stopOpacity="0.4" />
            <stop offset="100%" stopColor="#f59e0b" stopOpacity="0.4" />
          </linearGradient>

          {/* Radial gradient glow behind ring */}
          <radialGradient id="ringBackdropGlow" cx="50%" cy="50%" r="50%">
            <stop offset="0%" stopColor="#06b6d4" stopOpacity="0.06" />
            <stop offset="70%" stopColor="#8b5cf6" stopOpacity="0.03" />
            <stop offset="100%" stopColor="transparent" stopOpacity="0" />
          </radialGradient>

          {/* Glow filter */}
          <filter id="glowFilter" x="-20%" y="-20%" width="140%" height="140%">
            <feGaussianBlur stdDeviation="3" result="blur" />
            <feComposite in="SourceGraphic" in2="blur" operator="over" />
          </filter>
        </defs>

        {/* Ambient Ring Glow */}
        <circle cx={center} cy={center} r={radius + 35} fill="url(#ringBackdropGlow)" />

        {/* Outer Circular Track */}
        <circle
          cx={center}
          cy={center}
          r={radius}
          fill="none"
          stroke="#1e293b"
          strokeWidth={trackWidth}
          strokeDasharray="4 4"
          className="opacity-60"
        />

        <circle
          cx={center}
          cy={center}
          r={radius}
          fill="none"
          stroke="url(#ringTrackGradient)"
          strokeWidth={trackWidth}
          strokeOpacity={0.8}
        />

        {/* Quadrant Tick Marks & Labels */}
        {[
          { angle: 0, label: '0 / 2³²', hex: '0x00000000' },
          { angle: 90, label: '2³⁰ (25%)', hex: '0x40000000' },
          { angle: 180, label: '2³¹ (50%)', hex: '0x80000000' },
          { angle: 270, label: '3×2³⁰ (75%)', hex: '0xC0000000' },
        ].map((tick) => {
          const ptOuter = polarToCartesian(tick.angle, radius + 12);
          const ptLabel = polarToCartesian(tick.angle, radius + 26);
          const isTop = tick.angle === 0;
          const isBottom = tick.angle === 180;

          return (
            <g key={tick.angle} className="opacity-70">
              <circle cx={ptOuter.x} cy={ptOuter.y} r={2} fill="#64748b" />
              <text
                x={ptLabel.x}
                y={ptLabel.y + (isTop ? -4 : isBottom ? 6 : 3)}
                textAnchor="middle"
                className="fill-slate-500 font-mono text-[9px] font-medium tracking-wider"
              >
                {tick.label}
              </text>
            </g>
          );
        })}

        {/* Clockwise Direction Arrows */}
        {[45, 135, 225, 315].map((angle) => {
          const pt = polarToCartesian(angle, radius);
          return (
            <g
              key={angle}
              transform={`translate(${pt.x}, ${pt.y}) rotate(${angle})`}
              className="opacity-30"
            >
              <path d="M -4 -3 L 0 0 L -4 3" fill="none" stroke="#94a3b8" strokeWidth={1.5} />
            </g>
          );
        })}

        {/* Virtual Node Tokens */}
        {showVirtualNodes &&
          positions.map((pos) => {
            const pt = polarToCartesian(pos.angle, radius);
            const color = getNodeColor(pos.nodeId).bg;
            const isHovered = hoveredToken?.token === pos.token;
            const isTarget = activeRoute?.targetReplicaToken === pos.token;

            return (
              <circle
                key={pos.token}
                cx={pt.x}
                cy={pt.y}
                r={isTarget ? 6 : isHovered ? 4.5 : 2.5}
                fill={color}
                fillOpacity={isTarget ? 1 : isHovered ? 0.9 : 0.65}
                stroke={isTarget ? '#ffffff' : color}
                strokeWidth={isTarget ? 2 : 0}
                filter={isTarget ? 'url(#glowFilter)' : undefined}
                className="transition-all duration-150 cursor-pointer"
                onMouseEnter={() => setHoveredToken(pos)}
                onMouseLeave={() => setHoveredToken(null)}
              />
            );
          })}

        {/* Animated Key Routing Sweep Arc */}
        {activeRoute && getSweepArcPath() && (
          <motion.path
            d={getSweepArcPath()!}
            fill="none"
            stroke="#06b6d4"
            strokeWidth={3.5}
            strokeDasharray="6 4"
            filter="url(#glowFilter)"
            initial={{ pathLength: 0, opacity: 0 }}
            animate={{ pathLength: 1, opacity: 1 }}
            transition={{ duration: 0.6, ease: 'easeOut' }}
          />
        )}

        {/* Key Landing Particle */}
        {activeRoute && keyCoord && (
          <g>
            {/* Ripple wave */}
            <motion.circle
              cx={keyCoord.x}
              cy={keyCoord.y}
              r={12}
              fill="none"
              stroke="#06b6d4"
              strokeWidth={1.5}
              initial={{ scale: 0.5, opacity: 1 }}
              animate={{ scale: 2.2, opacity: 0 }}
              transition={{ repeat: Infinity, duration: 1.4, ease: 'easeOut' }}
            />
            {/* Core dot */}
            <circle
              cx={keyCoord.x}
              cy={keyCoord.y}
              r={6}
              fill="#06b6d4"
              stroke="#ffffff"
              strokeWidth={2}
              filter="url(#glowFilter)"
            />
          </g>
        )}

        {/* Target Node Replica Pulse */}
        {activeRoute && targetCoord && (
          <g>
            <motion.circle
              cx={targetCoord.x}
              cy={targetCoord.y}
              r={14}
              fill="none"
              stroke={getNodeColor(activeRoute.targetNodeId).bg}
              strokeWidth={2}
              initial={{ scale: 0.8, opacity: 1 }}
              animate={{ scale: 2.5, opacity: 0 }}
              transition={{ repeat: Infinity, duration: 1.2, ease: 'easeOut' }}
            />
            <circle
              cx={targetCoord.x}
              cy={targetCoord.y}
              r={8}
              fill={getNodeColor(activeRoute.targetNodeId).bg}
              stroke="#ffffff"
              strokeWidth={2}
              filter="url(#glowFilter)"
            />
          </g>
        )}

        {/* Physical Node Anchors (Large Badges) */}
        {physicalNodeAverages.map((node) => {
          const pt = polarToCartesian(node.angle, radius);
          const color = getNodeColor(node.id).bg;
          const isTargetNode = activeRoute?.targetNodeId === node.id;

          return (
            <g
              key={node.id}
              className="cursor-pointer transition-transform duration-200 hover:scale-110"
              onClick={() => onSelectNode?.(node.id)}
            >
              {/* Outer halo if active */}
              {isTargetNode && (
                <circle
                  cx={pt.x}
                  cy={pt.y}
                  r={22}
                  fill={color}
                  fillOpacity={0.2}
                  className="animate-pulse"
                />
              )}
              {/* Main Badge */}
              <circle
                cx={pt.x}
                cy={pt.y}
                r={16}
                fill="#0f172a"
                stroke={color}
                strokeWidth={isTargetNode ? 3 : 2}
                filter="url(#glowFilter)"
              />
              <text
                x={pt.x}
                y={pt.y + 4}
                textAnchor="middle"
                className="fill-white font-mono text-[10px] font-bold pointer-events-none"
              >
                {node.id.replace('node-', 'N')}
              </text>
            </g>
          );
        })}
      </svg>

      {/* Center Ring Info Plate */}
      <div className="absolute inset-0 flex flex-col items-center justify-center pointer-events-none">
        <div className="w-48 h-48 rounded-full glass-panel flex flex-col items-center justify-center text-center p-3 border border-white/10 shadow-2xl backdrop-blur-md">
          {activeRoute ? (
            <motion.div
              key={activeRoute.key}
              initial={{ scale: 0.9, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              className="space-y-1"
            >
              <span className="text-[10px] uppercase font-mono tracking-wider text-cyan-400 font-semibold">
                Clockwise Route
              </span>
              <p className="font-mono text-sm font-bold text-white truncate max-w-[150px]">
                "{activeRoute.key}"
              </p>
              <div className="pt-1 flex flex-col items-center">
                <span className="text-[10px] font-mono text-slate-400">Target Node</span>
                <span
                  className="text-xs font-mono font-bold px-2 py-0.5 rounded-full mt-0.5 border"
                  style={{
                    color: getNodeColor(activeRoute.targetNodeId).bg,
                    borderColor: `${getNodeColor(activeRoute.targetNodeId).bg}40`,
                    backgroundColor: `${getNodeColor(activeRoute.targetNodeId).bg}15`,
                  }}
                >
                  {activeRoute.targetNodeId}
                </span>
              </div>
              <p className="text-[10px] font-mono text-slate-400 pt-1">
                {activeRoute.angle.toFixed(1)}° &rarr; {((activeRoute.targetPosition >>> 0) / 0xffffffff * 360).toFixed(1)}°
              </p>
            </motion.div>
          ) : (
            <div className="space-y-1.5">
              <span className="text-[10px] uppercase font-mono tracking-widest text-slate-400 font-medium">
                Consistent Hashing
              </span>
              <p className="font-mono text-xl font-bold tracking-tight text-slate-100">
                2³² - 1
              </p>
              <p className="text-[10px] text-slate-400 font-mono">
                {positions.length} Tokens &bull; {nodes.length} Nodes
              </p>
              <div className="pt-1">
                <span className="text-[9px] font-mono text-cyan-400/80 px-2 py-0.5 rounded-full bg-cyan-500/10 border border-cyan-500/20">
                  O(log N) Lookup
                </span>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Hovered Token Tooltip */}
      <AnimatePresence>
        {hoveredToken && (
          <motion.div
            initial={{ opacity: 0, scale: 0.9 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.9 }}
            className="absolute bottom-2 left-1/2 -translate-x-1/2 z-30 px-3 py-1.5 rounded-xl bg-slate-900/90 border border-white/15 text-xs font-mono text-slate-200 shadow-2xl backdrop-blur-md pointer-events-none flex items-center gap-2"
          >
            <span
              className="w-2 h-2 rounded-full"
              style={{ backgroundColor: getNodeColor(hoveredToken.nodeId).bg }}
            />
            <span>Token: <strong>{hoveredToken.token}</strong></span>
            <span className="text-slate-400">|</span>
            <span className="text-cyan-300">0x{(hoveredToken.position >>> 0).toString(16).toUpperCase()}</span>
            <span className="text-slate-400">({hoveredToken.angle.toFixed(1)}°)</span>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};
