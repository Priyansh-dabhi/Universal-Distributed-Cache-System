import React, { useState } from 'react';
import { ARCHITECTURE_TIERS } from '../core/docs/concepts-data';
import { ConceptCard } from '../components/docs/ConceptCard';
import { TierNavigation } from '../components/docs/TierNavigation';
import { GlassCard } from '../components/shared/GlassCard';
import { Badge } from '../components/shared/Badge';
import { BookOpen, Search, ExternalLink } from 'lucide-react';
import { APP_CONFIG } from '../core/constants';

export const LearnPage: React.FC = () => {
  const [selectedTier, setSelectedTier] = useState<number>(1);
  const [searchQuery, setSearchQuery] = useState<string>('');

  const activeTier = ARCHITECTURE_TIERS.find((t) => t.tierNumber === selectedTier) || ARCHITECTURE_TIERS[0];

  // Search filter across all concepts
  const searchResults = searchQuery.trim()
    ? ARCHITECTURE_TIERS.flatMap((t) => t.concepts).filter(
        (c) =>
          c.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
          c.tagline.toLowerCase().includes(searchQuery.toLowerCase()) ||
          c.what.toLowerCase().includes(searchQuery.toLowerCase()) ||
          c.why.toLowerCase().includes(searchQuery.toLowerCase()) ||
          c.how.toLowerCase().includes(searchQuery.toLowerCase()) ||
          c.sourceFile.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : null;

  return (
    <div className="space-y-6">
      {/* Page Title & Search Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <div className="flex items-center gap-2.5">
            <h2 className="text-xl sm:text-2xl font-bold text-white tracking-tight flex items-center gap-2">
              <BookOpen className="w-6 h-6 text-cyan-400" />
              Architecture Concepts & Reference Guide
            </h2>
            <Badge variant="cyan" size="md">
              10 Topics
            </Badge>
          </div>
          <p className="text-xs sm:text-sm text-slate-400 mt-1">
            A first-principles guide to every distributed pattern, eviction algorithm, and concurrency design implemented in this repository.
          </p>
        </div>

        {/* GitHub Full Doc Link */}
        <a
          href={`${APP_CONFIG.githubUrl}/blob/main/docs/architecture-concepts.md`}
          target="_blank"
          rel="noopener noreferrer"
          className="px-3.5 py-1.5 rounded-xl bg-white/5 hover:bg-white/10 text-slate-300 hover:text-white border border-white/10 text-xs font-mono flex items-center gap-1.5 transition-colors self-start sm:self-auto"
        >
          <span>architecture-concepts.md</span>
          <ExternalLink className="w-3.5 h-3.5 text-cyan-400" />
        </a>
      </div>

      {/* Search Bar */}
      <div className="relative">
        <Search className="w-4 h-4 absolute left-3.5 top-3 text-slate-400" />
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          placeholder="Search any concept (e.g. 'virtual nodes', 'mutex', 'zipfian', 'scan resistance', 'lazy expiration')..."
          className="w-full pl-10 pr-4 py-2.5 bg-slate-900/90 border border-white/10 rounded-xl text-xs font-mono text-white placeholder-slate-500 focus:outline-none focus:border-cyan-500/50"
        />
        {searchQuery && (
          <button
            onClick={() => setSearchQuery('')}
            className="absolute right-3.5 top-2.5 text-xs text-slate-400 hover:text-white font-mono"
          >
            Clear
          </button>
        )}
      </div>

      {/* Main Content Layout */}
      {searchResults ? (
        /* Search Results View */
        <div className="space-y-4">
          <div className="flex items-center justify-between text-xs font-mono text-slate-400 pb-2 border-b border-white/5">
            <span>
              Search Results for: <strong className="text-white">"{searchQuery}"</strong>
            </span>
            <span>{searchResults.length} matches found</span>
          </div>

          {searchResults.length === 0 ? (
            <div className="h-44 flex flex-col items-center justify-center p-6 text-center border border-dashed border-white/10 rounded-2xl text-slate-500 font-mono text-xs">
              No architectural concepts matched your search query. Try searching for "LRU", "Ring", "Mutex", or "TTL".
            </div>
          ) : (
            <div className="space-y-3">
              {searchResults.map((c) => (
                <ConceptCard key={c.id} concept={c} defaultExpanded />
              ))}
            </div>
          )}
        </div>
      ) : (
        /* Tiered Guide Layout */
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
          {/* Left Sidebar: Tier Selector */}
          <div className="lg:col-span-4">
            <GlassCard>
              <TierNavigation
                tiers={ARCHITECTURE_TIERS}
                selectedTier={selectedTier}
                onSelectTier={(t) => setSelectedTier(t)}
              />
            </GlassCard>
          </div>

          {/* Right Panel: Concepts in Active Tier */}
          <div className="lg:col-span-8 space-y-4">
            {/* Active Tier Header */}
            <div className="p-4 rounded-2xl border border-white/10 bg-slate-900/70 space-y-1">
              <div className="flex items-center gap-2">
                <span className="w-5 h-5 rounded-md bg-cyan-500/20 text-cyan-400 flex items-center justify-center font-mono font-bold text-xs border border-cyan-500/30">
                  {activeTier.tierNumber}
                </span>
                <h3 className="font-bold text-white text-base tracking-wide">
                  {activeTier.title}
                </h3>
              </div>
              <p className="text-xs text-slate-400 font-sans leading-relaxed">
                {activeTier.summary}
              </p>
            </div>

            {/* Concept Cards Stack */}
            <div className="space-y-3">
              {activeTier.concepts.map((concept, idx) => (
                <ConceptCard
                  key={concept.id}
                  concept={concept}
                  defaultExpanded={idx === 0}
                />
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
