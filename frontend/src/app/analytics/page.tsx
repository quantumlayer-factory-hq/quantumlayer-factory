'use client';

import { useState } from 'react';
import { motion } from 'framer-motion';
import { TrendingUp, Users, Clock, Cpu, Database, Zap, Activity, BarChart3, PieChart, LineChart, Download, RefreshCw } from 'lucide-react';
import { GlassPanel } from '@/components/glass/GlassPanel';
import { GlassButton } from '@/components/glass/GlassButton';

interface MetricCard {
  title: string;
  value: string;
  change: string;
  changeType: 'increase' | 'decrease' | 'neutral';
  icon: any;
  color: string;
}

interface ChartData {
  name: string;
  value: number;
  color: string;
}

const metricsData: MetricCard[] = [
  {
    title: 'Total Generations',
    value: '12,847',
    change: '+23.4%',
    changeType: 'increase',
    icon: Zap,
    color: 'text-quantum-primary'
  },
  {
    title: 'Active Users',
    value: '2,394',
    change: '+18.2%',
    changeType: 'increase',
    icon: Users,
    color: 'text-quantum-success'
  },
  {
    title: 'Avg Generation Time',
    value: '28.4s',
    change: '-12.1%',
    changeType: 'decrease',
    icon: Clock,
    color: 'text-quantum-warning'
  },
  {
    title: 'Success Rate',
    value: '98.7%',
    change: '+2.1%',
    changeType: 'increase',
    icon: Activity,
    color: 'text-quantum-success'
  }
];

const languageData: ChartData[] = [
  { name: 'Python', value: 35, color: '#3B82F6' },
  { name: 'TypeScript', value: 28, color: '#10B981' },
  { name: 'Go', value: 18, color: '#F59E0B' },
  { name: 'Java', value: 12, color: '#EF4444' },
  { name: 'Rust', value: 7, color: '#8B5CF6' }
];

const frameworkData: ChartData[] = [
  { name: 'FastAPI', value: 32, color: '#06B6D4' },
  { name: 'React', value: 24, color: '#3B82F6' },
  { name: 'Express', value: 20, color: '#10B981' },
  { name: 'Spring Boot', value: 15, color: '#F59E0B' },
  { name: 'Gin', value: 9, color: '#EF4444' }
];

const agentPerformance = [
  { name: 'Backend Agent', completions: 3247, avgTime: '24.3s', successRate: 99.2 },
  { name: 'Frontend Agent', completions: 2851, avgTime: '31.7s', successRate: 98.8 },
  { name: 'Database Agent', completions: 3102, avgTime: '18.9s', successRate: 99.5 },
  { name: 'API Agent', completions: 3189, avgTime: '22.1s', successRate: 98.9 },
  { name: 'Test Agent', completions: 2643, avgTime: '35.2s', successRate: 97.8 },
  { name: 'DevOps Agent', completions: 1947, avgTime: '42.6s', successRate: 98.3 },
  { name: 'Security Agent', completions: 1823, avgTime: '28.7s', successRate: 99.1 }
];

const hourlyData = [
  { time: '00:00', generations: 45 },
  { time: '04:00', generations: 23 },
  { time: '08:00', generations: 189 },
  { time: '12:00', generations: 267 },
  { time: '16:00', generations: 234 },
  { time: '20:00', generations: 156 },
];

export default function AnalyticsPage() {
  const [timeRange, setTimeRange] = useState('7d');
  const [isLoading, setIsLoading] = useState(false);

  const timeRanges = [
    { value: '1d', label: '24 Hours' },
    { value: '7d', label: '7 Days' },
    { value: '30d', label: '30 Days' },
    { value: '90d', label: '90 Days' }
  ];

  const refreshData = () => {
    setIsLoading(true);
    setTimeout(() => setIsLoading(false), 1500);
  };

  return (
    <div className="min-h-screen pt-20 pb-20">
      <div className="max-w-7xl mx-auto px-6 py-8">
        {/* Header */}
        <div className="flex flex-col md:flex-row md:items-center md:justify-between mb-8">
          <div>
            <motion.h1
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              className="text-4xl font-bold mb-2"
            >
              <span className="text-gradient">Analytics</span> Dashboard
            </motion.h1>
            <motion.p
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.1 }}
              className="text-text-secondary"
            >
              Real-time insights into your QuantumLayer Factory performance
            </motion.p>
          </div>

          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.2 }}
            className="flex items-center gap-4 mt-4 md:mt-0"
          >
            {/* Time Range Selector */}
            <div className="flex bg-white/5 rounded-lg p-1 border border-white/10">
              {timeRanges.map((range) => (
                <button
                  key={range.value}
                  onClick={() => setTimeRange(range.value)}
                  className={`px-3 py-1 rounded-md text-sm font-medium transition-all ${
                    timeRange === range.value
                      ? 'bg-quantum-primary/20 text-quantum-primary'
                      : 'text-text-secondary hover:text-text-primary'
                  }`}
                >
                  {range.label}
                </button>
              ))}
            </div>

            <GlassButton
              onClick={refreshData}
              disabled={isLoading}
              className="px-4 py-2"
            >
              <RefreshCw className={`w-4 h-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
              Refresh
            </GlassButton>

            <GlassButton variant="secondary" className="px-4 py-2">
              <Download className="w-4 h-4 mr-2" />
              Export
            </GlassButton>
          </motion.div>
        </div>

        {/* Key Metrics */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
          {metricsData.map((metric, index) => {
            const Icon = metric.icon;
            return (
              <motion.div
                key={metric.title}
                initial={{ opacity: 0, y: 30 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.3 + index * 0.1 }}
              >
                <GlassPanel className="p-6 hover:border-quantum-primary/30 transition-all duration-300">
                  <div className="flex items-start justify-between">
                    <div>
                      <p className="text-text-muted text-sm mb-1">{metric.title}</p>
                      <h3 className="text-2xl font-bold text-text-primary mb-2">{metric.value}</h3>
                      <div className={`flex items-center text-sm ${
                        metric.changeType === 'increase' ? 'text-quantum-success' :
                        metric.changeType === 'decrease' ? 'text-quantum-warning' : 'text-text-muted'
                      }`}>
                        <TrendingUp className="w-4 h-4 mr-1" />
                        {metric.change}
                      </div>
                    </div>
                    <div className={`p-3 rounded-xl bg-white/5 ${metric.color}`}>
                      <Icon className="w-6 h-6" />
                    </div>
                  </div>
                </GlassPanel>
              </motion.div>
            );
          })}
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 mb-8">
          {/* Language Usage Chart */}
          <motion.div
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.5 }}
          >
            <GlassPanel className="p-6">
              <div className="flex items-center gap-3 mb-6">
                <PieChart className="w-5 h-5 text-quantum-primary" />
                <h2 className="text-xl font-semibold">Language Distribution</h2>
              </div>

              <div className="space-y-4">
                {languageData.map((item, index) => (
                  <div key={item.name} className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div
                        className="w-4 h-4 rounded-full"
                        style={{ backgroundColor: item.color }}
                      />
                      <span className="text-text-primary font-medium">{item.name}</span>
                    </div>
                    <div className="text-right">
                      <div className="text-text-primary font-semibold">{item.value}%</div>
                    </div>
                  </div>
                ))}
              </div>
            </GlassPanel>
          </motion.div>

          {/* Framework Usage Chart */}
          <motion.div
            initial={{ opacity: 0, y: 30 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.6 }}
          >
            <GlassPanel className="p-6">
              <div className="flex items-center gap-3 mb-6">
                <BarChart3 className="w-5 h-5 text-quantum-success" />
                <h2 className="text-xl font-semibold">Framework Usage</h2>
              </div>

              <div className="space-y-4">
                {frameworkData.map((item, index) => (
                  <div key={item.name} className="space-y-2">
                    <div className="flex items-center justify-between">
                      <span className="text-text-primary font-medium">{item.name}</span>
                      <span className="text-text-muted text-sm">{item.value}%</span>
                    </div>
                    <div className="w-full bg-white/10 rounded-full h-2">
                      <motion.div
                        className="h-2 rounded-full"
                        style={{ backgroundColor: item.color }}
                        initial={{ width: 0 }}
                        animate={{ width: `${item.value}%` }}
                        transition={{ delay: 0.7 + index * 0.1, duration: 0.8 }}
                      />
                    </div>
                  </div>
                ))}
              </div>
            </GlassPanel>
          </motion.div>
        </div>

        {/* Agent Performance Table */}
        <motion.div
          initial={{ opacity: 0, y: 30 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.7 }}
          className="mb-8"
        >
          <GlassPanel className="p-6">
            <div className="flex items-center gap-3 mb-6">
              <Cpu className="w-5 h-5 text-quantum-warning" />
              <h2 className="text-xl font-semibold">Agent Performance</h2>
            </div>

            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-white/10">
                    <th className="text-left pb-3 text-text-muted font-medium">Agent</th>
                    <th className="text-right pb-3 text-text-muted font-medium">Completions</th>
                    <th className="text-right pb-3 text-text-muted font-medium">Avg Time</th>
                    <th className="text-right pb-3 text-text-muted font-medium">Success Rate</th>
                  </tr>
                </thead>
                <tbody>
                  {agentPerformance.map((agent, index) => (
                    <motion.tr
                      key={agent.name}
                      initial={{ opacity: 0, x: -20 }}
                      animate={{ opacity: 1, x: 0 }}
                      transition={{ delay: 0.8 + index * 0.05 }}
                      className="border-b border-white/5"
                    >
                      <td className="py-4 text-text-primary font-medium">{agent.name}</td>
                      <td className="py-4 text-right text-text-secondary">{agent.completions.toLocaleString()}</td>
                      <td className="py-4 text-right text-text-secondary">{agent.avgTime}</td>
                      <td className="py-4 text-right">
                        <span className={`font-semibold ${
                          agent.successRate >= 99 ? 'text-quantum-success' :
                          agent.successRate >= 98 ? 'text-quantum-warning' : 'text-red-400'
                        }`}>
                          {agent.successRate}%
                        </span>
                      </td>
                    </motion.tr>
                  ))}
                </tbody>
              </table>
            </div>
          </GlassPanel>
        </motion.div>

        {/* Hourly Generation Activity */}
        <motion.div
          initial={{ opacity: 0, y: 30 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.9 }}
        >
          <GlassPanel className="p-6">
            <div className="flex items-center gap-3 mb-6">
              <LineChart className="w-5 h-5 text-quantum-purple" />
              <h2 className="text-xl font-semibold">Generation Activity (24h)</h2>
            </div>

            <div className="relative h-64">
              <div className="absolute inset-0 flex items-end justify-between gap-2">
                {hourlyData.map((data, index) => {
                  const height = (data.generations / Math.max(...hourlyData.map(d => d.generations))) * 100;
                  return (
                    <div key={data.time} className="flex-1 flex flex-col items-center">
                      <motion.div
                        className="w-full bg-gradient-to-t from-quantum-primary/60 to-quantum-primary/20 rounded-t-lg relative group cursor-pointer"
                        initial={{ height: 0 }}
                        animate={{ height: `${height}%` }}
                        transition={{ delay: 1 + index * 0.1, duration: 0.8 }}
                      >
                        <div className="absolute -top-8 left-1/2 -translate-x-1/2 opacity-0 group-hover:opacity-100 transition-opacity bg-black/80 rounded px-2 py-1 text-xs whitespace-nowrap">
                          {data.generations} generations
                        </div>
                      </motion.div>
                      <div className="text-xs text-text-muted mt-2">{data.time}</div>
                    </div>
                  );
                })}
              </div>
            </div>
          </GlassPanel>
        </motion.div>

        {/* System Health Status */}
        <motion.div
          initial={{ opacity: 0, y: 30 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 1.1 }}
          className="mt-8"
        >
          <GlassPanel className="p-6">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-3">
                <Database className="w-5 h-5 text-quantum-success" />
                <h2 className="text-xl font-semibold">System Health</h2>
              </div>
              <div className="flex items-center gap-2 text-sm text-quantum-success">
                <div className="w-2 h-2 bg-quantum-success rounded-full animate-pulse" />
                All systems operational
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
              <div className="text-center">
                <div className="text-3xl font-bold text-quantum-success mb-2">99.9%</div>
                <div className="text-text-muted text-sm">Uptime</div>
              </div>
              <div className="text-center">
                <div className="text-3xl font-bold text-quantum-primary mb-2">42ms</div>
                <div className="text-text-muted text-sm">Avg Latency</div>
              </div>
              <div className="text-center">
                <div className="text-3xl font-bold text-quantum-warning mb-2">7/7</div>
                <div className="text-text-muted text-sm">Agents Online</div>
              </div>
            </div>
          </GlassPanel>
        </motion.div>
      </div>
    </div>
  );
}