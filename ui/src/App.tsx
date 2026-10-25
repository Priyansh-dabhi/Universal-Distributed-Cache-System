import { lazy, Suspense } from 'react';
import { Routes, Route, Navigate, useLocation } from 'react-router-dom';
import { Toaster } from 'sonner';
import { motion, AnimatePresence } from 'framer-motion';
import { BackendProvider } from './context/BackendContext';
import { Header } from './components/layout/Header';
import { TabBar } from './components/layout/TabBar';

const HashRingPage = lazy(() => import('./pages/HashRingPage').then((m) => ({ default: m.HashRingPage })));
const PlaygroundPage = lazy(() => import('./pages/PlaygroundPage').then((m) => ({ default: m.PlaygroundPage })));
const EvictionPage = lazy(() => import('./pages/EvictionPage').then((m) => ({ default: m.EvictionPage })));
const MetricsPage = lazy(() => import('./pages/MetricsPage').then((m) => ({ default: m.MetricsPage })));
const LearnPage = lazy(() => import('./pages/LearnPage').then((m) => ({ default: m.LearnPage })));

function PageLoadingFallback() {
  return (
    <div className="flex flex-col items-center justify-center min-h-[50vh] gap-3">
      <div className="relative w-10 h-10">
        <div className="absolute inset-0 rounded-full border-2 border-cyan-500/20 animate-ping" />
        <div className="w-10 h-10 rounded-full border-2 border-cyan-400 border-t-transparent animate-spin" />
      </div>
      <span className="font-mono text-xs text-slate-400 tracking-wider uppercase animate-pulse">
        Loading view...
      </span>
    </div>
  );
}

function PageTransitionWrapper({ children }: { children: React.ReactNode }) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -6 }}
      transition={{ duration: 0.2, ease: [0.4, 0, 0.2, 1] }}
    >
      <Suspense fallback={<PageLoadingFallback />}>
        {children}
      </Suspense>
    </motion.div>
  );
}

function AppContent() {
  const location = useLocation();

  return (
    <div className="min-h-screen bg-[#030712] text-slate-100 flex flex-col selection:bg-cyan-500/30 selection:text-cyan-200">
      <Toaster position="top-right" theme="dark" richColors />

      {/* Top Navbar */}
      <Header />

      {/* 5-Tab Bar with animated sliding indicator */}
      <TabBar />

      {/* Main Content Area with fluid page transitions */}
      <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-6">
        <AnimatePresence mode="wait">
          <Routes location={location} key={location.pathname}>
            <Route path="/" element={<Navigate to="/ring" replace />} />
            <Route path="/ring" element={<PageTransitionWrapper><HashRingPage /></PageTransitionWrapper>} />
            <Route path="/playground" element={<PageTransitionWrapper><PlaygroundPage /></PageTransitionWrapper>} />
            <Route path="/eviction" element={<PageTransitionWrapper><EvictionPage /></PageTransitionWrapper>} />
            <Route path="/metrics" element={<PageTransitionWrapper><MetricsPage /></PageTransitionWrapper>} />
            <Route path="/learn" element={<PageTransitionWrapper><LearnPage /></PageTransitionWrapper>} />
            <Route path="*" element={<Navigate to="/ring" replace />} />
          </Routes>
        </AnimatePresence>
      </main>

      {/* Minimal Footer */}
      <footer className="border-t border-white/5 py-4 text-center text-xs text-slate-500 bg-slate-950/30">
        <div className="max-w-7xl mx-auto px-4 flex flex-col sm:flex-row items-center justify-between gap-2">
          <span>Universal Distributed Cache System &bull; Go + React/TypeScript</span>
          <span className="font-mono text-[11px] text-slate-400">
            Consistent Hashing Visualizer &bull; High-Performance Caching
          </span>
        </div>
      </footer>
    </div>
  );
}

export function App() {
  return (
    <BackendProvider>
      <AppContent />
    </BackendProvider>
  );
}

export default App;
