'use client';

import { useState, useEffect, useCallback } from 'react';

export type AgentStatus = {
  type: string;
  status: 'pending' | 'active' | 'completed' | 'failed';
  progress: number;
  files: number;
  duration: string;
};

export type GenerationStatus = {
  workflowId: string;
  status: 'idle' | 'starting' | 'running' | 'completed' | 'failed';
  progress: number;
  agents: AgentStatus[];
  files: number;
  metrics: {
    totalFiles: number;
    totalDuration: number;
    tokensUsed: number;
    cost: string;
  };
  error?: string;
};

export function useGeneration() {
  const [status, setStatus] = useState<GenerationStatus>({
    workflowId: '',
    status: 'idle',
    progress: 0,
    agents: [],
    files: 0,
    metrics: {
      totalFiles: 0,
      totalDuration: 0,
      tokensUsed: 0,
      cost: '0.00'
    }
  });

  const [isGenerating, setIsGenerating] = useState(false);

  const startGeneration = useCallback(async (params: {
    brief: string;
    parallel?: boolean;
    provider?: string;
    model?: string;
  }) => {
    if (!params.brief.trim()) return;

    setIsGenerating(true);
    setStatus(prev => ({ ...prev, status: 'starting' }));

    try {
      const response = await fetch('/api/generate', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(params),
      });

      if (!response.ok) {
        throw new Error('Failed to start generation');
      }

      const data = await response.json();

      setStatus(prev => ({
        ...prev,
        workflowId: data.workflowId,
        status: 'running'
      }));

      // Start polling for status updates
      pollStatus(data.workflowId);

    } catch (error) {
      console.error('Generation failed:', error);
      setStatus(prev => ({
        ...prev,
        status: 'failed',
        error: error instanceof Error ? error.message : 'Unknown error'
      }));
      setIsGenerating(false);
    }
  }, []);

  const pollStatus = useCallback((workflowId: string) => {
    const poll = async () => {
      try {
        const response = await fetch(`/api/status/${workflowId}`);
        if (!response.ok) throw new Error('Failed to fetch status');

        const data = await response.json();

        setStatus(prev => ({
          ...prev,
          ...data
        }));

        if (data.status === 'completed' || data.status === 'failed') {
          setIsGenerating(false);
        } else {
          // Continue polling every 2 seconds
          setTimeout(poll, 2000);
        }
      } catch (error) {
        console.error('Status polling failed:', error);
        setStatus(prev => ({
          ...prev,
          status: 'failed',
          error: 'Failed to get status updates'
        }));
        setIsGenerating(false);
      }
    };

    poll();
  }, []);

  return {
    status,
    isGenerating,
    startGeneration,
  };
}