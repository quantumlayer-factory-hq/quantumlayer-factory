'use client';

import { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { Copy, Download, ExternalLink, Eye, Code, FileText, Search, Maximize2, Minimize2 } from 'lucide-react';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { GlassButton } from '@/components/glass/GlassButton';

interface CodeViewerProps {
  filename: string;
  content: string;
  onClose?: () => void;
}

interface FileStats {
  lines: number;
  size: string;
  language: string;
}

const getLanguageFromFilename = (filename: string): string => {
  const ext = filename.split('.').pop()?.toLowerCase();
  const languageMap: Record<string, string> = {
    'js': 'javascript',
    'jsx': 'jsx',
    'ts': 'typescript',
    'tsx': 'tsx',
    'py': 'python',
    'go': 'go',
    'rs': 'rust',
    'java': 'java',
    'sql': 'sql',
    'json': 'json',
    'yaml': 'yaml',
    'yml': 'yaml',
    'md': 'markdown',
    'dockerfile': 'dockerfile',
    'sh': 'bash',
    'css': 'css',
    'scss': 'scss',
    'html': 'html',
    'xml': 'xml',
    'toml': 'toml',
    'ini': 'ini',
    'env': 'bash'
  };

  if (filename.toLowerCase().includes('dockerfile')) return 'dockerfile';
  if (filename.toLowerCase().includes('makefile')) return 'makefile';

  return languageMap[ext || ''] || 'text';
};

const getFileIcon = (filename: string) => {
  const ext = filename.split('.').pop()?.toLowerCase();

  if (filename.toLowerCase().includes('readme')) return '📖';
  if (filename.toLowerCase().includes('dockerfile')) return '🐳';
  if (filename.toLowerCase().includes('makefile')) return '🔧';

  const iconMap: Record<string, string> = {
    'js': '🟨',
    'jsx': '⚛️',
    'ts': '🔷',
    'tsx': '⚛️',
    'py': '🐍',
    'go': '🐹',
    'rs': '🦀',
    'java': '☕',
    'sql': '🗄️',
    'json': '📋',
    'yaml': '⚙️',
    'yml': '⚙️',
    'md': '📝',
    'css': '🎨',
    'html': '🌐',
    'xml': '📄',
    'env': '🔐',
    'txt': '📄'
  };

  return iconMap[ext || ''] || '📄';
};

const formatFileSize = (content: string): string => {
  const bytes = new Blob([content]).size;
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
};

export function CodeViewer({ filename, content, onClose }: CodeViewerProps) {
  const [copied, setCopied] = useState(false);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [lineWrap, setLineWrap] = useState(false);
  const [showLineNumbers, setShowLineNumbers] = useState(true);

  const language = getLanguageFromFilename(filename);
  const icon = getFileIcon(filename);

  const stats: FileStats = {
    lines: content.split('\n').length,
    size: formatFileSize(content),
    language: language.charAt(0).toUpperCase() + language.slice(1)
  };

  const copyToClipboard = async () => {
    try {
      await navigator.clipboard.writeText(content);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (error) {
      console.error('Failed to copy:', error);
    }
  };

  const downloadFile = () => {
    const blob = new Blob([content], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  const openInNewTab = () => {
    const blob = new Blob([content], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    window.open(url, '_blank');
    URL.revokeObjectURL(url);
  };

  // Highlight search terms in content
  const highlightedContent = searchTerm ?
    content.replace(new RegExp(`(${searchTerm.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi'), '★$1★') :
    content;

  // Custom syntax highlighter style with search highlighting
  const customStyle = {
    ...vscDarkPlus,
    'pre[class*="language-"]': {
      ...vscDarkPlus['pre[class*="language-"]'],
      background: 'transparent',
      fontSize: '14px',
      lineHeight: '1.5',
      fontFamily: 'Monaco, Consolas, "Courier New", monospace'
    },
    'code[class*="language-"]': {
      ...vscDarkPlus['code[class*="language-"]'],
      background: 'transparent',
      fontSize: '14px',
      fontFamily: 'Monaco, Consolas, "Courier New", monospace'
    }
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -20 }}
      className={`${isFullscreen ? 'fixed inset-0 z-50' : ''}`}
    >
      <div className={`glass-panel ${isFullscreen ? 'h-full' : 'max-h-[600px]'} flex flex-col overflow-hidden`}>
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-white/10 bg-white/5">
          <div className="flex items-center gap-3">
            <span className="text-2xl">{icon}</span>
            <div>
              <h3 className="text-lg font-semibold text-text-primary">{filename}</h3>
              <div className="flex items-center gap-4 text-xs text-text-muted">
                <span>{stats.lines} lines</span>
                <span>{stats.size}</span>
                <span>{stats.language}</span>
              </div>
            </div>
          </div>

          <div className="flex items-center gap-2">
            {/* Search */}
            <div className="relative">
              <Search className="absolute left-2 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted" />
              <input
                type="text"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                placeholder="Search..."
                className="w-32 bg-white/10 border border-white/20 rounded-lg pl-8 pr-3 py-1 text-sm text-text-primary focus:outline-none focus:border-quantum-primary/50"
              />
            </div>

            {/* View Options */}
            <button
              onClick={() => setShowLineNumbers(!showLineNumbers)}
              className={`p-1 rounded text-xs transition-colors ${
                showLineNumbers ? 'text-quantum-primary bg-quantum-primary/10' : 'text-text-muted hover:text-text-primary'
              }`}
              title="Toggle line numbers"
            >
              #
            </button>

            <button
              onClick={() => setLineWrap(!lineWrap)}
              className={`p-1 rounded text-xs transition-colors ${
                lineWrap ? 'text-quantum-primary bg-quantum-primary/10' : 'text-text-muted hover:text-text-primary'
              }`}
              title="Toggle line wrap"
            >
              ⤴
            </button>

            {/* Action Buttons */}
            <GlassButton size="sm" onClick={copyToClipboard} className="px-2 py-1">
              {copied ? '✓' : <Copy className="w-4 h-4" />}
            </GlassButton>

            <GlassButton size="sm" onClick={downloadFile} className="px-2 py-1">
              <Download className="w-4 h-4" />
            </GlassButton>

            <GlassButton size="sm" onClick={openInNewTab} className="px-2 py-1">
              <ExternalLink className="w-4 h-4" />
            </GlassButton>

            <button
              onClick={() => setIsFullscreen(!isFullscreen)}
              className="p-1 text-text-muted hover:text-text-primary transition-colors"
            >
              {isFullscreen ? <Minimize2 className="w-4 h-4" /> : <Maximize2 className="w-4 h-4" />}
            </button>

            {onClose && (
              <button
                onClick={onClose}
                className="p-1 text-text-muted hover:text-text-primary transition-colors ml-2"
              >
                ✕
              </button>
            )}
          </div>
        </div>

        {/* Content */}
        <div className={`flex-1 overflow-auto ${isFullscreen ? 'h-full' : ''}`}>
          <SyntaxHighlighter
            language={language}
            style={customStyle}
            showLineNumbers={showLineNumbers}
            wrapLines={lineWrap}
            wrapLongLines={lineWrap}
            customStyle={{
              background: 'transparent',
              margin: 0,
              padding: '1rem',
              fontSize: '14px',
              minHeight: '100%'
            }}
            codeTagProps={{
              style: {
                background: 'transparent',
                fontFamily: 'Monaco, Consolas, "Courier New", monospace'
              }
            }}
            lineNumberStyle={{
              color: 'rgba(255, 255, 255, 0.3)',
              borderRight: '1px solid rgba(255, 255, 255, 0.1)',
              paddingRight: '1rem',
              marginRight: '1rem'
            }}
          >
            {highlightedContent.replace(/★([^★]+)★/g, (match, p1) =>
              `\x1b[43m\x1b[30m${p1}\x1b[0m`
            )}
          </SyntaxHighlighter>
        </div>

        {/* Footer */}
        {isFullscreen && (
          <div className="flex items-center justify-between px-4 py-2 border-t border-white/10 bg-white/5 text-xs text-text-muted">
            <div className="flex items-center gap-4">
              <span>Language: {stats.language}</span>
              <span>Lines: {stats.lines}</span>
              <span>Size: {stats.size}</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-quantum-success">🟢 Ready</span>
              <span>UTF-8</span>
            </div>
          </div>
        )}

        {/* Copied notification */}
        {copied && (
          <motion.div
            initial={{ opacity: 0, scale: 0.8 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.8 }}
            className="absolute top-16 right-4 bg-quantum-success/20 border border-quantum-success/50 rounded-lg px-3 py-2 text-sm text-quantum-success"
          >
            ✓ Copied to clipboard!
          </motion.div>
        )}
      </div>
    </motion.div>
  );
}