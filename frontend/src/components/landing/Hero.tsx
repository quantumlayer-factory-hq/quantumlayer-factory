'use client';

import { motion } from 'framer-motion';
import Link from 'next/link';
import { ArrowRight, Sparkles, Cpu, Network, Zap } from 'lucide-react';
import { GlassButton } from '@/components/glass/GlassButton';

export function Hero() {
  return (
    <section className="relative min-h-screen flex items-center justify-center overflow-hidden">
      {/* Background Elements */}
      <div className="absolute inset-0">
        {/* Animated Grid */}
        <div className="absolute inset-0 opacity-20">
          <div className="h-full w-full bg-grid-white/[0.02] bg-[size:50px_50px]" />
        </div>

        {/* Floating Particles */}
        {Array.from({ length: 20 }).map((_, i) => (
          <motion.div
            key={i}
            className="absolute w-1 h-1 bg-quantum-primary/30 rounded-full"
            animate={{
              x: [0, 100, 0],
              y: [0, -100, 0],
              opacity: [0.3, 0.8, 0.3],
            }}
            transition={{
              duration: 10 + i * 2,
              repeat: Infinity,
              ease: 'easeInOut',
            }}
            style={{
              left: `${10 + (i * 5) % 80}%`,
              top: `${20 + (i * 3) % 60}%`,
            }}
          />
        ))}
      </div>

      <div className="relative z-10 max-w-7xl mx-auto px-6 py-20">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-16 items-center">
          {/* Left Column - Content */}
          <motion.div
            initial={{ opacity: 0, x: -50 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ duration: 0.8 }}
            className="space-y-8"
          >
            {/* Badge */}
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.2 }}
              className="flex items-center gap-2 px-4 py-2 rounded-full bg-quantum-primary/10 border border-quantum-primary/20 w-fit"
            >
              <Sparkles className="w-4 h-4 text-quantum-primary" />
              <span className="text-sm font-medium text-quantum-primary">
                Production-Ready Multi-Agent Architecture
              </span>
            </motion.div>

            {/* Main Heading */}
            <div className="space-y-4">
              <motion.h1
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.3 }}
                className="text-5xl lg:text-7xl font-bold leading-tight"
              >
                Transform{' '}
                <span className="text-gradient">Natural Language</span>{' '}
                Into Production Apps
              </motion.h1>

              <motion.p
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.4 }}
                className="text-xl text-text-secondary leading-relaxed max-w-lg"
              >
                QuantumLayer Factory orchestrates 7 specialized AI agents to generate
                complete, production-ready applications from simple descriptions.
              </motion.p>
            </div>

            {/* Stats */}
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.5 }}
              className="grid grid-cols-3 gap-6 py-6"
            >
              <div className="text-center">
                <div className="text-3xl font-bold text-quantum-primary">7</div>
                <div className="text-sm text-text-muted">AI Agents</div>
              </div>
              <div className="text-center">
                <div className="text-3xl font-bold text-quantum-success">12x</div>
                <div className="text-sm text-text-muted">Faster</div>
              </div>
              <div className="text-center">
                <div className="text-3xl font-bold text-quantum-warning">99%</div>
                <div className="text-sm text-text-muted">Accurate</div>
              </div>
            </motion.div>

            {/* CTA Buttons */}
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.6 }}
              className="flex flex-col sm:flex-row gap-4"
            >
              <Link href="/playground">
                <GlassButton size="lg" className="group px-8 py-4 bg-quantum-primary/20 hover:bg-quantum-primary/30 border-quantum-primary/50">
                  <span className="flex items-center gap-2">
                    Try Interactive Demo
                    <ArrowRight className="w-5 h-5 group-hover:translate-x-1 transition-transform" />
                  </span>
                </GlassButton>
              </Link>

              <Link href="/docs">
                <GlassButton variant="secondary" size="lg" className="px-8 py-4">
                  View Architecture
                </GlassButton>
              </Link>
            </motion.div>

            {/* Social Proof */}
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.7 }}
              className="flex items-center gap-6 pt-6 border-t border-white/5"
            >
              <span className="text-sm text-text-muted">Trusted by enterprise teams</span>
              <div className="flex items-center gap-4">
                {['Tech', 'Finance', 'Healthcare', 'E-commerce'].map((industry) => (
                  <div key={industry} className="px-3 py-1 rounded-full bg-white/5 text-xs text-text-muted">
                    {industry}
                  </div>
                ))}
              </div>
            </motion.div>
          </motion.div>

          {/* Right Column - Agent Network Visualization */}
          <motion.div
            initial={{ opacity: 0, x: 50 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ duration: 0.8, delay: 0.2 }}
            className="relative"
          >
            <div className="relative aspect-square max-w-lg mx-auto">
              {/* Central Hub */}
              <motion.div
                animate={{ rotate: 360 }}
                transition={{ duration: 20, repeat: Infinity, ease: 'linear' }}
                className="absolute inset-0 flex items-center justify-center"
              >
                <div className="w-24 h-24 rounded-full bg-gradient-to-r from-quantum-primary to-quantum-purple flex items-center justify-center shadow-2xl shadow-quantum-primary/20">
                  <Cpu className="w-12 h-12 text-white" />
                </div>
              </motion.div>

              {/* Agent Nodes */}
              {[
                { name: 'Backend', angle: 0, color: 'bg-blue-500', icon: '🔧' },
                { name: 'Frontend', angle: 51, color: 'bg-green-500', icon: '🎨' },
                { name: 'Database', angle: 102, color: 'bg-yellow-500', icon: '🗄️' },
                { name: 'API', angle: 153, color: 'bg-purple-500', icon: '⚡' },
                { name: 'Test', angle: 204, color: 'bg-red-500', icon: '✅' },
                { name: 'DevOps', angle: 255, color: 'bg-indigo-500', icon: '⚙️' },
                { name: 'Security', angle: 306, color: 'bg-pink-500', icon: '🛡️' },
              ].map((agent, index) => {
                const radius = 180;
                const x = Math.cos((agent.angle * Math.PI) / 180) * radius;
                const y = Math.sin((agent.angle * Math.PI) / 180) * radius;

                return (
                  <motion.div
                    key={agent.name}
                    initial={{ opacity: 0, scale: 0 }}
                    animate={{ opacity: 1, scale: 1 }}
                    transition={{ delay: 0.5 + index * 0.1 }}
                    className="absolute"
                    style={{
                      left: '50%',
                      top: '50%',
                      transform: `translate(calc(-50% + ${x}px), calc(-50% + ${y}px))`,
                    }}
                  >
                    <div className="relative group">
                      {/* Connection Line */}
                      <motion.div
                        initial={{ pathLength: 0 }}
                        animate={{ pathLength: 1 }}
                        transition={{ delay: 1 + index * 0.1, duration: 0.5 }}
                        className="absolute inset-0"
                      >
                        <svg
                          className="absolute top-1/2 left-1/2 w-48 h-48 -translate-x-1/2 -translate-y-1/2"
                          viewBox="0 0 200 200"
                        >
                          <line
                            x1={100 - x / 2}
                            y1={100 - y / 2}
                            x2="100"
                            y2="100"
                            stroke="rgba(74, 158, 255, 0.3)"
                            strokeWidth="1"
                            strokeDasharray="2 2"
                          />
                        </svg>
                      </motion.div>

                      {/* Agent Node */}
                      <motion.div
                        whileHover={{ scale: 1.1 }}
                        className={`w-16 h-16 rounded-full ${agent.color} bg-opacity-20 border-2 border-current flex items-center justify-center backdrop-blur-sm cursor-pointer group-hover:shadow-lg transition-all`}
                      >
                        <span className="text-2xl">{agent.icon}</span>
                      </motion.div>

                      {/* Agent Label */}
                      <div className="absolute -bottom-8 left-1/2 -translate-x-1/2 text-xs font-medium text-text-muted whitespace-nowrap">
                        {agent.name}
                      </div>

                      {/* Pulse Effect */}
                      <motion.div
                        animate={{
                          scale: [1, 1.2, 1],
                          opacity: [0.5, 0, 0.5],
                        }}
                        transition={{
                          duration: 2,
                          repeat: Infinity,
                          delay: index * 0.3,
                        }}
                        className={`absolute inset-0 w-16 h-16 rounded-full ${agent.color} bg-opacity-20`}
                      />
                    </div>
                  </motion.div>
                );
              })}
            </div>
          </motion.div>
        </div>
      </div>
    </section>
  );
}