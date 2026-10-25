import React, { useState } from 'react';
import { Database, Radio, Cpu, ExternalLink, Code2, Settings, RefreshCw, CheckCircle2, XCircle } from 'lucide-react';
import { APP_CONFIG } from '../../core/constants';
import { useBackend } from '../../hooks/useBackend';
import { motion, AnimatePresence } from 'framer-motion';

export const Header: React.FC = () => {
  const { mode, setMode, isConnected, isChecking, backendUrl, setBackendUrl, checkConnection } = useBackend();
  const [showSettings, setShowSettings] = useState(false);
  const [inputUrl, setInputUrl] = useState(backendUrl);
  const [testing, setTesting] = useState(false);

  const handleTestConnection = async () => {
    setTesting(true);
    setBackendUrl(inputUrl);
    await checkConnection();
    setTesting(false);
  };

  return (
    <header className="sticky top-0 z-50 glass-panel border-b border-white/5 backdrop-blur-xl">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
        {/* Brand */}
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-gradient-to-tr from-cyan-600 via-cyan-500 to-emerald-400 p-0.5 flex items-center justify-center shadow-lg shadow-cyan-500/20">
            <div className="w-full h-full bg-slate-950 rounded-[10px] flex items-center justify-center">
              <Database className="w-5 h-5 text-cyan-400" />
            </div>
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h1 className="font-bold text-base tracking-tight text-white flex items-center gap-2">
                Universal Distributed Cache
                <span className="text-[10px] uppercase font-mono px-2 py-0.5 rounded-full bg-cyan-500/10 text-cyan-400 border border-cyan-500/20">
                  Studio
                </span>
              </h1>
            </div>
            <p className="text-xs text-slate-400 font-mono hidden sm:block">
              Consistent Hashing &bull; Virtual Nodes &bull; LRU / LFU / 2Q Eviction
            </p>
          </div>
        </div>

        {/* Controls & Links */}
        <div className="flex items-center gap-2.5">
          {/* Mode Switcher Pill */}
          <button
            onClick={() => setMode(mode === 'live' ? 'sim' : 'live')}
            className={`flex items-center gap-2 px-3 py-1.5 rounded-full text-xs font-medium border transition-all duration-200 cursor-pointer ${
              mode === 'live'
                ? 'bg-emerald-500/10 hover:bg-emerald-500/20 border-emerald-500/30 text-emerald-300'
                : 'bg-cyan-500/10 hover:bg-cyan-500/20 border-cyan-500/30 text-cyan-300'
            }`}
            title="Toggle between Live Go Cluster and In-Browser Simulator"
          >
            {mode === 'live' ? (
              <>
                <span className="relative flex h-2 w-2">
                  <span className={`animate-ping absolute inline-flex h-full w-full rounded-full opacity-75 ${isConnected ? 'bg-emerald-400' : 'bg-rose-400'}`} />
                  <span className={`relative inline-flex rounded-full h-2 w-2 ${isConnected ? 'bg-emerald-500' : 'bg-rose-500'}`} />
                </span>
                <Radio className="w-3.5 h-3.5 text-emerald-400" />
                <span>Live Cluster ({isConnected ? ':9000' : 'Offline'})</span>
              </>
            ) : (
              <>
                <Cpu className="w-3.5 h-3.5 text-cyan-400" />
                <span>Web Simulator</span>
              </>
            )}
          </button>

          {/* Connection Settings Toggle */}
          <button
            onClick={() => setShowSettings(!showSettings)}
            className="p-2 rounded-xl text-slate-400 hover:text-white bg-white/5 hover:bg-white/10 border border-white/5 transition-colors cursor-pointer"
            title="Cluster Connection Settings"
          >
            <Settings className="w-4 h-4" />
          </button>

          {/* GitHub Repo */}
          <a
            href={APP_CONFIG.githubUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="p-2 rounded-xl text-slate-400 hover:text-white bg-white/5 hover:bg-white/10 border border-white/5 transition-colors flex items-center gap-1.5 text-xs"
            title="View Go source code on GitHub"
          >
            <Code2 className="w-4 h-4" />
            <span className="hidden md:inline">Source</span>
            <ExternalLink className="w-3 h-3 opacity-60" />
          </a>
        </div>
      </div>

      {/* Settings Dropdown Dialog */}
      <AnimatePresence>
        {showSettings && (
          <motion.div
            initial={{ opacity: 0, y: -8 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -8 }}
            transition={{ duration: 0.15 }}
            className="border-t border-white/10 bg-slate-950/95 backdrop-blur-2xl px-4 py-4 max-w-7xl mx-auto shadow-2xl"
          >
            <div className="max-w-2xl mx-auto flex flex-col sm:flex-row items-center gap-3">
              <div className="flex-1 w-full">
                <label className="block text-xs font-mono text-slate-400 mb-1">
                  Go Distributed Router Base URL (default: http://127.0.0.1:9000)
                </label>
                <div className="relative">
                  <input
                    type="text"
                    value={inputUrl}
                    onChange={(e) => setInputUrl(e.target.value)}
                    placeholder="http://127.0.0.1:9000"
                    className="w-full bg-slate-900 border border-white/10 rounded-xl px-3.5 py-2 text-xs font-mono text-slate-200 focus:outline-none focus:border-cyan-500/50"
                  />
                  <div className="absolute right-3 top-2.5">
                    {isConnected ? (
                      <CheckCircle2 className="w-4 h-4 text-emerald-400" />
                    ) : (
                      <XCircle className="w-4 h-4 text-rose-400" />
                    )}
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-2 self-end sm:self-auto pt-4 sm:pt-4">
                <button
                  onClick={handleTestConnection}
                  disabled={testing || isChecking}
                  className="px-3.5 py-2 rounded-xl bg-cyan-500/10 hover:bg-cyan-500/20 border border-cyan-500/30 text-cyan-300 text-xs font-medium flex items-center gap-1.5 transition-colors cursor-pointer disabled:opacity-50"
                >
                  <RefreshCw className={`w-3.5 h-3.5 ${testing ? 'animate-spin' : ''}`} />
                  <span>Test Connection</span>
                </button>
                <button
                  onClick={() => setShowSettings(false)}
                  className="px-3 py-2 rounded-xl bg-white/5 hover:bg-white/10 text-slate-300 text-xs transition-colors cursor-pointer"
                >
                  Close
                </button>
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </header>
  );
};
