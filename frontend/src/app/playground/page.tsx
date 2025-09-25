'use client';

import { useState, useEffect } from 'react';
import { Play, Download, Copy, Terminal, FileText, Loader, CheckCircle, Clock, AlertCircle, Folder, FolderOpen } from 'lucide-react';
import { GlassPanel } from '@/components/glass/GlassPanel';
import { GlassButton } from '@/components/glass/GlassButton';
import { CodeViewer } from '@/components/code/CodeViewer';
import { motion, AnimatePresence } from 'framer-motion';

interface Agent {
  type: string;
  status: 'pending' | 'active' | 'completed' | 'failed';
  progress: number;
  files: number;
  duration: string;
}

interface GenerationStatus {
  workflowId: string;
  status: string;
  progress: number;
  agents: Agent[];
  files: number;
  metrics: {
    totalFiles: number;
    totalDuration: string;
    tokensUsed: number;
    cost: string;
  };
  realBackend?: boolean;
  rawOutput?: string;
}

const agentIcons: Record<string, string> = {
  backend: '🔧',
  frontend: '🎨',
  database: '🗄️',
  api: '⚡',
  test: '✅',
  devops: '⚙️',
  security: '🛡️'
};

const agentColors: Record<string, string> = {
  backend: 'bg-blue-500',
  frontend: 'bg-green-500',
  database: 'bg-yellow-500',
  api: 'bg-purple-500',
  test: 'bg-red-500',
  devops: 'bg-indigo-500',
  security: 'bg-pink-500'
};

interface FileTreeItem {
  name: string;
  path: string;
  type: 'file' | 'folder';
  children?: FileTreeItem[];
}

const buildFileTree = (files: string[]): FileTreeItem[] => {
  const root: FileTreeItem[] = [];
  const lookup: Record<string, FileTreeItem> = {};

  files.forEach(filePath => {
    const parts = filePath.split('/');
    let currentPath = '';

    parts.forEach((part, index) => {
      const parentPath = currentPath;
      currentPath = currentPath ? `${currentPath}/${part}` : part;

      if (!lookup[currentPath]) {
        const item: FileTreeItem = {
          name: part,
          path: currentPath,
          type: index === parts.length - 1 ? 'file' : 'folder',
          children: []
        };

        lookup[currentPath] = item;

        if (parentPath && lookup[parentPath]) {
          lookup[parentPath].children!.push(item);
        } else {
          root.push(item);
        }
      }
    });
  });

  // Sort items: folders first, then files, both alphabetically
  const sortItems = (items: FileTreeItem[]): FileTreeItem[] => {
    return items.sort((a, b) => {
      if (a.type !== b.type) {
        return a.type === 'folder' ? -1 : 1;
      }
      return a.name.localeCompare(b.name);
    }).map(item => ({
      ...item,
      children: item.children ? sortItems(item.children) : []
    }));
  };

  return sortItems(root);
};

const getFileIcon = (filename: string, isFolder: boolean = false) => {
  if (isFolder) return '📁';

  const name = filename.toLowerCase();
  const ext = name.split('.').pop();

  if (name.includes('readme')) return '📖';
  if (name.includes('dockerfile')) return '🐳';
  if (name.includes('makefile')) return '🔧';

  const iconMap: Record<string, string> = {
    'js': '🟨', 'jsx': '⚛️', 'ts': '🔷', 'tsx': '⚛️',
    'py': '🐍', 'go': '🐹', 'rs': '🦀', 'java': '☕',
    'sql': '🗄️', 'json': '📋', 'yaml': '⚙️', 'yml': '⚙️',
    'md': '📝', 'css': '🎨', 'html': '🌐', 'xml': '📄',
    'env': '🔐', 'txt': '📄', 'png': '🖼️', 'jpg': '🖼️',
    'gif': '🖼️', 'svg': '🎨', 'ico': '🖼️'
  };

  return iconMap[ext || ''] || '📄';
};

interface FileTreeItemProps {
  item: FileTreeItem;
  level: number;
  selectedFile: string | null;
  onFileSelect: (path: string) => void;
  index: number;
}

function FileTreeItem({ item, level, selectedFile, onFileSelect, index }: FileTreeItemProps) {
  const [isExpanded, setIsExpanded] = useState(level < 2); // Auto-expand first 2 levels

  const handleClick = () => {
    if (item.type === 'folder') {
      setIsExpanded(!isExpanded);
    } else {
      onFileSelect(item.path);
    }
  };

  return (
    <motion.div
      initial={{ opacity: 0, x: -20 }}
      animate={{ opacity: 1, x: 0 }}
      transition={{ delay: index * 0.02 }}
    >
      <button
        onClick={handleClick}
        className={`w-full flex items-center gap-2 px-2 py-1.5 rounded-md transition-all text-left hover:bg-white/5 ${
          selectedFile === item.path
            ? 'bg-quantum-primary/20 text-quantum-primary border-l-2 border-quantum-primary'
            : 'text-text-primary'
        }`}
        style={{ paddingLeft: `${level * 1.5 + 0.5}rem` }}
      >
        {item.type === 'folder' && (
          <span className={`text-xs transition-transform ${isExpanded ? 'rotate-90' : ''}`}>
            ▶
          </span>
        )}
        <span className="text-sm">{getFileIcon(item.name, item.type === 'folder')}</span>
        <span className="text-sm font-medium truncate">{item.name}</span>
        {item.type === 'folder' && item.children && (
          <span className="text-xs text-text-muted ml-auto">
            {item.children.length}
          </span>
        )}
      </button>

      {/* Children */}
      <AnimatePresence>
        {item.type === 'folder' && isExpanded && item.children && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
            transition={{ duration: 0.2 }}
          >
            {item.children.map((child, childIndex) => (
              <FileTreeItem
                key={child.path}
                item={child}
                level={level + 1}
                selectedFile={selectedFile}
                onFileSelect={onFileSelect}
                index={childIndex}
              />
            ))}
          </motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  );
}

export default function PlaygroundPage() {
  const [prompt, setPrompt] = useState('');
  const [workflowId, setWorkflowId] = useState<string | null>(null);
  const [status, setStatus] = useState<GenerationStatus | null>(null);
  const [isGenerating, setIsGenerating] = useState(false);
  const [generatedFiles, setGeneratedFiles] = useState<string[]>([]);
  const [selectedFile, setSelectedFile] = useState<string | null>(null);
  const [fileContent, setFileContent] = useState<string>('');

  // Configuration options
  const [provider, setProvider] = useState('bedrock');
  const [model, setModel] = useState('claude-3-5-sonnet');
  const [parallel, setParallel] = useState(true);
  const [outputDir, setOutputDir] = useState('');
  const [overlays, setOverlays] = useState<string[]>([]);
  const [showAdvanced, setShowAdvanced] = useState(false);

  // Update model when provider changes
  const handleProviderChange = (newProvider: string) => {
    setProvider(newProvider);
    // Reset to default model for the provider
    if (newProvider === 'bedrock') {
      setModel('claude-3-5-sonnet');
    } else if (newProvider === 'azure') {
      setModel('gpt-4o');
    } else {
      setModel('auto');
    }
  };

  // Poll for status updates
  useEffect(() => {
    let interval: NodeJS.Timeout;

    if (workflowId && isGenerating) {
      interval = setInterval(async () => {
        try {
          const response = await fetch(`/api/status/${workflowId}`);
          if (response.ok) {
            const data = await response.json();
            setStatus(data);

            if (data.status === 'completed' || data.status === 'failed') {
              setIsGenerating(false);
              if (data.status === 'completed') {
                loadGeneratedFiles(workflowId);
              }
            }
          }
        } catch (error) {
          console.error('Failed to fetch status:', error);
        }
      }, 2000);
    }

    return () => {
      if (interval) clearInterval(interval);
    };
  }, [workflowId, isGenerating]);

  const loadGeneratedFiles = async (id: string) => {
    try {
      const response = await fetch(`/api/files/${id}`);
      if (response.ok) {
        const files = await response.json();
        setGeneratedFiles(files);
      }
    } catch (error) {
      console.error('Failed to load files:', error);
    }
  };

  const handleGenerate = async () => {
    if (!prompt.trim()) return;

    setIsGenerating(true);
    setStatus(null);
    setGeneratedFiles([]);
    setSelectedFile(null);
    setFileContent('');

    try {
      console.log('Sending request with prompt:', prompt);
      const response = await fetch('/api/generate', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          prompt,
          provider,
          model,
          parallel,
          outputDir: outputDir || undefined,
          overlays: overlays.length > 0 ? overlays : undefined
        }),
      });

      console.log('Response status:', response.status);

      if (response.ok) {
        const data = await response.json();
        console.log('Generation response:', data);
        setWorkflowId(data.workflowId);
      } else {
        const errorData = await response.json().catch(() => ({ error: 'Unknown error' }));
        console.error('Generation failed:', response.status, errorData);
        alert(`Generation failed: ${errorData.error || response.statusText}`);
        setIsGenerating(false);
      }
    } catch (error) {
      console.error('Generation error:', error);
      alert(`Generation error: ${error}`);
      setIsGenerating(false);
    }
  };

  const getStatusIcon = (agentStatus: string) => {
    switch (agentStatus) {
      case 'active':
        return <Loader className="w-4 h-4 animate-spin text-quantum-primary" />;
      case 'completed':
        return <CheckCircle className="w-4 h-4 text-quantum-success" />;
      case 'failed':
        return <AlertCircle className="w-4 h-4 text-red-500" />;
      default:
        return <Clock className="w-4 h-4 text-text-muted" />;
    }
  };

  const viewFile = async (fileName: string) => {
    if (!workflowId) return;

    try {
      const response = await fetch(`/api/files/${workflowId}/${encodeURIComponent(fileName)}`);
      if (response.ok) {
        const content = await response.text();
        setFileContent(content);
        setSelectedFile(fileName);
      }
    } catch (error) {
      console.error('Failed to load file content:', error);
    }
  };

  return (
    <div className="min-h-screen pt-20 pb-20">
      <div className="max-w-7xl mx-auto px-6 py-8">
        {/* Header */}
        <div className="text-center mb-8">
          <h1 className="text-4xl font-bold mb-4">
            Interactive <span className="text-gradient">Playground</span>
          </h1>
          <p className="text-text-secondary max-w-2xl mx-auto">
            Generate complete applications from natural language descriptions.
            Watch as 7 specialized agents work in parallel to create production-ready code.
          </p>
        </div>

        <div className="grid grid-cols-1 xl:grid-cols-2 gap-8">
          {/* Left Column - Input & Generation */}
          <div className="space-y-6">
            {/* Input Section */}
            <GlassPanel className="p-6">
              <div className="flex items-center gap-3 mb-4">
                <Terminal className="w-5 h-5 text-quantum-primary" />
                <h2 className="text-xl font-semibold">Describe Your Application</h2>
              </div>

              <textarea
                value={prompt}
                onChange={(e) => setPrompt(e.target.value)}
                placeholder="Create a RESTful API for a task management system with user authentication, CRUD operations for tasks, and PostgreSQL database integration..."
                className="w-full h-32 bg-white/5 border border-white/10 rounded-lg px-4 py-3 text-text-primary placeholder-text-muted focus:outline-none focus:border-quantum-primary/50 resize-none"
                disabled={isGenerating}
              />

              {/* Configuration Options */}
              <div className="mt-6 space-y-4">
                {/* Basic Options */}
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-text-secondary mb-2">
                      LLM Provider
                    </label>
                    <select
                      value={provider}
                      onChange={(e) => handleProviderChange(e.target.value)}
                      className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-text-primary focus:outline-none focus:border-quantum-primary/50"
                      disabled={isGenerating}
                    >
                      <option value="bedrock">AWS Bedrock</option>
                      <option value="azure">Azure OpenAI</option>
                      <option value="auto">Auto Select</option>
                    </select>
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-text-secondary mb-2">
                      Model
                    </label>
                    <select
                      value={model}
                      onChange={(e) => setModel(e.target.value)}
                      className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-text-primary focus:outline-none focus:border-quantum-primary/50"
                      disabled={isGenerating}
                    >
                      {provider === 'bedrock' && (
                        <>
                          <option value="claude-3-5-sonnet">Claude 3.5 Sonnet</option>
                          <option value="claude-3-sonnet">Claude 3 Sonnet</option>
                          <option value="claude-3-haiku">Claude 3 Haiku</option>
                        </>
                      )}
                      {provider === 'azure' && (
                        <>
                          <option value="gpt-4o">GPT-4o</option>
                          <option value="gpt-4">GPT-4</option>
                          <option value="gpt-3.5-turbo">GPT-3.5 Turbo</option>
                        </>
                      )}
                      {provider === 'auto' && (
                        <option value="auto">Best Available</option>
                      )}
                    </select>
                  </div>

                  <div className="flex items-center gap-4">
                    <div className="flex-1">
                      <label className="block text-sm font-medium text-text-secondary mb-2">
                        Execution Mode
                      </label>
                      <div className="flex items-center gap-3 bg-white/5 rounded-lg p-2 border border-white/10">
                        <button
                          onClick={() => setParallel(false)}
                          disabled={isGenerating}
                          className={`flex-1 px-3 py-1 rounded text-sm font-medium transition-all ${
                            !parallel
                              ? 'bg-quantum-primary/20 text-quantum-primary border border-quantum-primary/50'
                              : 'text-text-secondary hover:text-text-primary'
                          }`}
                        >
                          Sequential
                        </button>
                        <button
                          onClick={() => setParallel(true)}
                          disabled={isGenerating}
                          className={`flex-1 px-3 py-1 rounded text-sm font-medium transition-all ${
                            parallel
                              ? 'bg-quantum-primary/20 text-quantum-primary border border-quantum-primary/50'
                              : 'text-text-secondary hover:text-text-primary'
                          }`}
                        >
                          Parallel
                        </button>
                      </div>
                    </div>
                  </div>
                </div>

                {/* Advanced Options Toggle */}
                <button
                  onClick={() => setShowAdvanced(!showAdvanced)}
                  className="text-sm text-quantum-primary hover:text-quantum-primary/80 transition-colors"
                  disabled={isGenerating}
                >
                  {showAdvanced ? '▼' : '▶'} Advanced Options
                </button>

                {showAdvanced && (
                  <motion.div
                    initial={{ opacity: 0, height: 0 }}
                    animate={{ opacity: 1, height: 'auto' }}
                    exit={{ opacity: 0, height: 0 }}
                    className="space-y-4 pt-4 border-t border-white/10"
                  >
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div>
                        <label className="block text-sm font-medium text-text-secondary mb-2">
                          Output Directory
                        </label>
                        <input
                          type="text"
                          value={outputDir}
                          onChange={(e) => setOutputDir(e.target.value)}
                          placeholder="Custom output directory (optional)"
                          className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-text-primary placeholder-text-muted focus:outline-none focus:border-quantum-primary/50"
                          disabled={isGenerating}
                        />
                      </div>

                      <div>
                        <label className="block text-sm font-medium text-text-secondary mb-2">
                          Overlays
                        </label>
                        <input
                          type="text"
                          value={overlays.join(', ')}
                          onChange={(e) => setOverlays(e.target.value.split(',').map(s => s.trim()).filter(Boolean))}
                          placeholder="e.g., fintech, pci, healthcare"
                          className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-text-primary placeholder-text-muted focus:outline-none focus:border-quantum-primary/50"
                          disabled={isGenerating}
                        />
                        <div className="text-xs text-text-muted mt-1">
                          Comma-separated compliance overlays
                        </div>
                      </div>
                    </div>

                    {/* Overlay suggestions */}
                    <div className="flex flex-wrap gap-2">
                      {['fintech', 'pci', 'healthcare', 'gdpr', 'sox', 'hipaa'].map((overlay) => (
                        <button
                          key={overlay}
                          onClick={() => {
                            if (!overlays.includes(overlay)) {
                              setOverlays([...overlays, overlay]);
                            }
                          }}
                          disabled={isGenerating || overlays.includes(overlay)}
                          className="px-3 py-1 rounded-full bg-white/5 border border-white/10 text-xs text-text-muted hover:text-text-primary hover:border-quantum-primary/30 transition-all disabled:opacity-50"
                        >
                          + {overlay}
                        </button>
                      ))}
                    </div>
                  </motion.div>
                )}
              </div>

              {/* Configuration Summary */}
              <div className="mt-4 p-3 bg-white/5 rounded-lg border border-white/10">
                <div className="flex flex-wrap items-center gap-4 text-sm">
                  <div className="flex items-center gap-2">
                    <div className="w-2 h-2 bg-quantum-primary rounded-full"></div>
                    <span className="text-text-secondary">Provider:</span>
                    <span className="text-text-primary font-medium">
                      {provider === 'bedrock' ? 'AWS Bedrock' : provider === 'azure' ? 'Azure OpenAI' : 'Auto Select'}
                    </span>
                  </div>
                  <div className="flex items-center gap-2">
                    <div className="w-2 h-2 bg-quantum-success rounded-full"></div>
                    <span className="text-text-secondary">Model:</span>
                    <span className="text-text-primary font-medium">{model}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <div className={`w-2 h-2 rounded-full ${parallel ? 'bg-quantum-warning' : 'bg-quantum-purple'}`}></div>
                    <span className="text-text-secondary">Mode:</span>
                    <span className="text-text-primary font-medium">
                      {parallel ? '⚡ Parallel' : '🔄 Sequential'}
                    </span>
                  </div>
                  {overlays.length > 0 && (
                    <div className="flex items-center gap-2">
                      <div className="w-2 h-2 bg-red-500 rounded-full"></div>
                      <span className="text-text-secondary">Overlays:</span>
                      <span className="text-text-primary font-medium">{overlays.join(', ')}</span>
                    </div>
                  )}
                </div>
              </div>

              <div className="flex justify-between items-center mt-6">
                <div className="flex items-center gap-4 text-sm text-text-muted">
                  <span>Supports: Python, Go, Node.js, React, Vue</span>
                  <span className="text-xs px-2 py-1 bg-quantum-success/10 text-quantum-success rounded-full">
                    Real Backend Connected
                  </span>
                </div>
                <GlassButton
                  onClick={handleGenerate}
                  disabled={!prompt.trim() || isGenerating}
                  className="px-6 py-2 bg-quantum-primary/20 hover:bg-quantum-primary/30 border-quantum-primary/50 disabled:opacity-50"
                >
                  <div className="flex items-center gap-2">
                    {isGenerating ? (
                      <Loader className="w-4 h-4 animate-spin" />
                    ) : (
                      <Play className="w-4 h-4" />
                    )}
                    {isGenerating ? 'Generating...' : 'Generate Application'}
                  </div>
                </GlassButton>
              </div>
            </GlassPanel>

            {/* Agent Status */}
            <AnimatePresence>
              {status && (
                <motion.div
                  initial={{ opacity: 0, y: 20 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -20 }}
                >
                  <GlassPanel className="p-6">
                    <div className="flex items-center gap-3 mb-6">
                      <div className="w-3 h-3 bg-quantum-primary rounded-full animate-pulse" />
                      <h2 className="text-xl font-semibold">Agent Orchestration</h2>
                      <div className="ml-auto text-sm text-text-secondary">
                        Progress: {status.progress}%
                      </div>
                    </div>

                    <div className="space-y-4">
                      {status.agents.map((agent) => (
                        <motion.div
                          key={agent.type}
                          initial={{ opacity: 0, x: -20 }}
                          animate={{ opacity: 1, x: 0 }}
                          className="flex items-center gap-4 p-3 rounded-lg bg-white/5"
                        >
                          <div className={`w-10 h-10 rounded-lg ${agentColors[agent.type]} bg-opacity-20 flex items-center justify-center`}>
                            <span className="text-lg">{agentIcons[agent.type]}</span>
                          </div>

                          <div className="flex-1">
                            <div className="flex items-center gap-2 mb-1">
                              <span className="font-medium capitalize">{agent.type} Agent</span>
                              {getStatusIcon(agent.status)}
                            </div>

                            <div className="w-full bg-white/10 rounded-full h-2">
                              <motion.div
                                className="bg-quantum-primary h-2 rounded-full"
                                initial={{ width: 0 }}
                                animate={{ width: `${agent.progress}%` }}
                                transition={{ duration: 0.5 }}
                              />
                            </div>
                          </div>

                          <div className="text-right text-sm text-text-muted">
                            <div>{agent.files} files</div>
                            <div>{agent.duration}</div>
                          </div>
                        </motion.div>
                      ))}
                    </div>

                    {/* Metrics */}
                    {status.metrics && (
                      <div className="mt-6 pt-4 border-t border-white/10">
                        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-center">
                          <div>
                            <div className="text-2xl font-bold text-quantum-primary">{status.metrics.totalFiles}</div>
                            <div className="text-xs text-text-muted">Files Generated</div>
                          </div>
                          <div>
                            <div className="text-2xl font-bold text-quantum-success">{status.metrics.totalDuration}</div>
                            <div className="text-xs text-text-muted">Duration</div>
                          </div>
                          <div>
                            <div className="text-2xl font-bold text-quantum-warning">{status.metrics.tokensUsed.toLocaleString()}</div>
                            <div className="text-xs text-text-muted">Tokens Used</div>
                          </div>
                          <div>
                            <div className="text-2xl font-bold text-text-primary">${status.metrics.cost}</div>
                            <div className="text-xs text-text-muted">Estimated Cost</div>
                          </div>
                        </div>
                      </div>
                    )}
                  </GlassPanel>
                </motion.div>
              )}
            </AnimatePresence>
          </div>

          {/* Right Column - File Explorer & Code View */}
          <div className="space-y-6">
            {/* Enhanced File Explorer */}
            <AnimatePresence>
              {generatedFiles.length > 0 && (
                <motion.div
                  initial={{ opacity: 0, y: 20 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -20 }}
                  className="space-y-6"
                >
                  {/* File Tree */}
                  <GlassPanel className="p-6">
                    <div className="flex items-center gap-3 mb-4">
                      <FolderOpen className="w-5 h-5 text-quantum-success" />
                      <h2 className="text-xl font-semibold">Project Structure</h2>
                      <div className="ml-auto flex gap-2">
                        <div className="text-sm text-text-muted bg-white/5 px-2 py-1 rounded">
                          {generatedFiles.length} files
                        </div>
                        <GlassButton size="sm" className="px-3 py-1">
                          <Download className="w-4 h-4 mr-2" />
                          Download All
                        </GlassButton>
                      </div>
                    </div>

                    <div className="space-y-1 max-h-80 overflow-y-auto">
                      {buildFileTree(generatedFiles).map((item, index) => (
                        <FileTreeItem
                          key={item.path}
                          item={item}
                          level={0}
                          selectedFile={selectedFile}
                          onFileSelect={viewFile}
                          index={index}
                        />
                      ))}
                    </div>
                  </GlassPanel>

                  {/* Enhanced Code Viewer */}
                  {selectedFile && fileContent && (
                    <CodeViewer
                      filename={selectedFile}
                      content={fileContent}
                      onClose={() => {
                        setSelectedFile(null);
                        setFileContent('');
                      }}
                    />
                  )}
                </motion.div>
              )}
            </AnimatePresence>
          </div>
        </div>
      </div>
    </div>
  );
}