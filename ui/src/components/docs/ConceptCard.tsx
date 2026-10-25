import React, { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { useNavigate } from 'react-router-dom';
import { CodeBlock } from '../shared/CodeBlock';
import type { Concept } from '../../core/docs/concepts-data';
import { ChevronDown, ChevronUp, FileCode, ArrowRight, HelpCircle, AlertCircle, Cpu } from 'lucide-react';

interface ConceptCardProps {
  concept: Concept;
  defaultExpanded?: boolean;
}

export const ConceptCard: React.FC<ConceptCardProps> = ({ concept, defaultExpanded = false }) => {
  const [isExpanded, setIsExpanded] = useState(defaultExpanded);
  const navigate = useNavigate();

  return (
    <div className="rounded-2xl border border-white/10 bg-slate-900/60 overflow-hidden transition-all hover:border-white/20 font-mono text-xs">
      {/* Clickable Header Bar */}
      <div
        onClick={() => setIsExpanded(!isExpanded)}
        className="p-4 flex items-center justify-between cursor-pointer hover:bg-white/[0.02] transition-colors"
      >
        <div className="space-y-1">
          <div className="flex items-center gap-2 flex-wrap">
            <h4 className="font-bold text-white text-sm tracking-wide">
              {concept.title}
            </h4>
            <span className="text-[10px] px-2 py-0.5 rounded-full bg-cyan-500/10 text-cyan-400 border border-cyan-500/20 flex items-center gap-1">
              <FileCode className="w-3 h-3" />
              <span>{concept.sourceFile}</span>
            </span>
          </div>
          <p className="text-xs text-slate-400 font-sans">
            {concept.tagline}
          </p>
        </div>

        <button
          type="button"
          className="p-1.5 rounded-xl bg-white/5 text-slate-400 hover:text-white transition-colors"
        >
          {isExpanded ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
        </button>
      </div>

      {/* Expandable Details Section */}
      <AnimatePresence>
        {isExpanded && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: 'auto', opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.25, ease: 'easeInOut' }}
            className="border-t border-white/5 bg-slate-950/40 p-4 space-y-4"
          >
            {/* 3 Pillars: What, Why, How */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
              {/* What */}
              <div className="p-3 rounded-xl bg-slate-900/80 border border-white/5 space-y-1.5 font-sans">
                <span className="text-[11px] font-mono font-bold text-cyan-400 flex items-center gap-1.5">
                  <HelpCircle className="w-3.5 h-3.5" />
                  What it means
                </span>
                <p className="text-xs text-slate-300 leading-relaxed">
                  {concept.what}
                </p>
              </div>

              {/* Why */}
              <div className="p-3 rounded-xl bg-slate-900/80 border border-white/5 space-y-1.5 font-sans">
                <span className="text-[11px] font-mono font-bold text-amber-400 flex items-center gap-1.5">
                  <AlertCircle className="w-3.5 h-3.5" />
                  Why we need it
                </span>
                <p className="text-xs text-slate-300 leading-relaxed">
                  {concept.why}
                </p>
              </div>

              {/* How */}
              <div className="p-3 rounded-xl bg-slate-900/80 border border-white/5 space-y-1.5 font-sans">
                <span className="text-[11px] font-mono font-bold text-emerald-400 flex items-center gap-1.5">
                  <Cpu className="w-3.5 h-3.5" />
                  How it works in Go
                </span>
                <p className="text-xs text-slate-300 leading-relaxed">
                  {concept.how}
                </p>
              </div>
            </div>

            {/* Code Snippet */}
            <div>
              <span className="text-[11px] font-mono text-slate-400 block mb-1.5">
                Go Implementation Reference:
              </span>
              <CodeBlock code={concept.codeSnippet} language="go" />
            </div>

            {/* Interactive Jump Link */}
            {concept.interactiveLink && (
              <div className="pt-1 flex justify-end">
                <button
                  onClick={() => navigate(concept.interactiveLink!.tabPath)}
                  className="px-3.5 py-1.5 rounded-xl bg-cyan-500/10 hover:bg-cyan-500/20 border border-cyan-500/30 text-cyan-300 text-xs font-semibold flex items-center gap-1.5 transition-colors cursor-pointer"
                >
                  <span>{concept.interactiveLink.label}</span>
                  <ArrowRight className="w-3.5 h-3.5" />
                </button>
              </div>
            )}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};
