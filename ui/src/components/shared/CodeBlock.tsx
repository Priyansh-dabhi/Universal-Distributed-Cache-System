import React, { useState } from 'react';
import { Copy, Check } from 'lucide-react';
import { clsx } from 'clsx';

interface CodeBlockProps {
  code: string;
  language?: string;
  className?: string;
}

export const CodeBlock: React.FC<CodeBlockProps> = ({ code, language = 'json', className }) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  return (
    <div className={clsx('relative rounded-xl overflow-hidden border border-white/10 bg-slate-950/80', className)}>
      <div className="flex items-center justify-between px-3.5 py-1.5 border-b border-white/5 bg-white/[0.02]">
        <span className="font-mono text-[11px] uppercase tracking-wider text-slate-400 font-medium">
          {language}
        </span>
        <button
          onClick={handleCopy}
          className="flex items-center gap-1 text-[11px] text-slate-400 hover:text-white px-2 py-0.5 rounded-md hover:bg-white/10 transition-colors cursor-pointer"
          title="Copy code to clipboard"
        >
          {copied ? (
            <>
              <Check className="w-3.5 h-3.5 text-emerald-400" />
              <span className="text-emerald-400">Copied</span>
            </>
          ) : (
            <>
              <Copy className="w-3.5 h-3.5" />
              <span>Copy</span>
            </>
          )}
        </button>
      </div>
      <pre className="p-3.5 text-xs font-mono text-slate-300 overflow-x-auto leading-relaxed selection:bg-cyan-500/30">
        <code>{code}</code>
      </pre>
    </div>
  );
};
