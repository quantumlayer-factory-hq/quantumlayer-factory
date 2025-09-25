'use client';

import { motion } from 'framer-motion';
import {
  Server,
  Globe,
  Database,
  Code,
  TestTube,
  Shield,
  FileText,
  Cpu
} from 'lucide-react';

export type AgentType = 'backend' | 'frontend' | 'database' | 'api' | 'test' | 'devops' | 'security' | 'docs';
export type AgentStatus = 'pending' | 'active' | 'completed' | 'failed';

interface AgentNodeProps {
  type: AgentType;
  status: AgentStatus;
  progress: number;
  files: number;
  duration: string;
  className?: string;
}

const agentConfig = {
  backend: { icon: Server, label: 'Backend', color: '#4A9EFF' },
  frontend: { icon: Globe, label: 'Frontend', color: '#10B981' },
  database: { icon: Database, label: 'Database', color: '#F59E0B' },
  api: { icon: Code, label: 'API', color: '#8B5CF6' },
  test: { icon: TestTube, label: 'Test', color: '#EF4444' },
  devops: { icon: Cpu, label: 'DevOps', color: '#6B7280' },
  security: { icon: Shield, label: 'Security', color: '#DC2626' },
  docs: { icon: FileText, label: 'Docs', color: '#059669' }
};

export function AgentNode({
  type,
  status,
  progress,
  files,
  duration,
  className = ''
}: AgentNodeProps) {
  const config = agentConfig[type];
  const Icon = config.icon;

  const statusClasses = {
    pending: 'agent-node pending',
    active: 'agent-node active',
    completed: 'agent-node completed',
    failed: 'agent-node opacity-50 border-quantum-danger/50'
  };

  return (
    <motion.div
      initial={{ scale: 0.8, opacity: 0 }}
      animate={{ scale: 1, opacity: 1 }}
      transition={{ duration: 0.3, delay: Math.random() * 0.2 }}
      className={`${statusClasses[status]} ${className}`}
    >
      {/* Agent Icon & Status */}
      <div className="flex items-center gap-3 mb-4">
        <div className="relative">
          <div className={`
            w-10 h-10 rounded-xl flex items-center justify-center
            ${status === 'active' ? 'bg-quantum-primary/20 animate-pulse-glow' : ''}
            ${status === 'completed' ? 'bg-quantum-success/20' : ''}
            ${status === 'pending' ? 'bg-white/5' : ''}
            ${status === 'failed' ? 'bg-quantum-danger/20' : ''}
          `}>
            <Icon
              className={`w-5 h-5 ${
                status === 'active' ? 'text-quantum-primary' :
                status === 'completed' ? 'text-quantum-success' :
                status === 'failed' ? 'text-quantum-danger' :
                'text-text-muted'
              }`}
            />
          </div>

          {/* Status indicator dot */}
          <div className={`
            absolute -top-1 -right-1 w-3 h-3 rounded-full border-2 border-bg-elevated
            ${status === 'active' ? 'bg-quantum-success animate-pulse' : ''}
            ${status === 'completed' ? 'bg-quantum-primary' : ''}
            ${status === 'pending' ? 'bg-text-muted' : ''}
            ${status === 'failed' ? 'bg-quantum-danger' : ''}
          `} />
        </div>

        <div className="flex-1">
          <h3 className="text-sm font-medium text-text-primary">{config.label}</h3>
          <span className="text-xs text-text-muted capitalize">{status}</span>
        </div>

        <div className="text-right">
          <div className="text-sm font-mono text-text-primary">{progress}%</div>
          <div className="text-xs text-text-muted">{duration}</div>
        </div>
      </div>

      {/* Progress Bar */}
      <div className="progress-bar mb-3">
        <motion.div
          className="progress-fill"
          initial={{ width: 0 }}
          animate={{ width: `${progress}%` }}
          transition={{ duration: 0.5, delay: 0.2 }}
          style={{ backgroundColor: config.color + '40' }}
        />
      </div>

      {/* Stats */}
      <div className="flex justify-between text-xs">
        <span className="text-text-muted">
          {files} file{files !== 1 ? 's' : ''}
        </span>
        <span className="text-text-muted">
          {Math.round(progress * 0.1 * files)} LOC
        </span>
      </div>
    </motion.div>
  );
}