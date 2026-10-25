import React from 'react';
import { NavLink, useLocation } from 'react-router-dom';
import { motion } from 'framer-motion';
import { CircleDot, Terminal, Layers, Activity, BookOpen } from 'lucide-react';
import { TABS } from '../../core/constants';

const ICONS = {
  CircleDot,
  Terminal,
  Layers,
  Activity,
  BookOpen,
};

export const TabBar: React.FC = () => {
  const location = useLocation();

  return (
    <nav className="border-b border-white/5 bg-slate-950/40 backdrop-blur-md sticky top-16 z-40">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center space-x-1 sm:space-x-2 py-2 overflow-x-auto no-scrollbar">
          {TABS.map((tab) => {
            const Icon = ICONS[tab.icon as keyof typeof ICONS];
            const isActive = location.pathname.startsWith(tab.path);

            return (
              <NavLink
                key={tab.id}
                to={tab.path}
                className={`relative flex items-center gap-2 px-4 py-2 rounded-xl text-xs sm:text-sm font-medium transition-colors duration-150 whitespace-nowrap cursor-pointer ${
                  isActive ? 'text-cyan-300' : 'text-slate-400 hover:text-slate-200 hover:bg-white/5'
                }`}
              >
                {isActive && (
                  <motion.div
                    layoutId="activeTabIndicator"
                    className="absolute inset-0 rounded-xl bg-cyan-500/10 border border-cyan-500/30 shadow-[0_0_15px_rgba(6,182,212,0.15)]"
                    transition={{ type: 'spring', stiffness: 380, damping: 30 }}
                  />
                )}
                <span className="relative z-10 flex items-center gap-2">
                  {Icon && <Icon className="w-4 h-4" />}
                  <span>{tab.label}</span>
                </span>
              </NavLink>
            );
          })}
        </div>
      </div>
    </nav>
  );
};
