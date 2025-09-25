'use client';

import { motion } from 'framer-motion';
import { Server, Database, Cloud, Shield, Zap, GitBranch, Code, TestTube } from 'lucide-react';
import { GlassPanel } from '@/components/glass/GlassPanel';

const architectureData = [
  {
    layer: 'Multi-Agent Orchestration',
    description: 'Intelligent agent coordination with Temporal workflows',
    components: [
      { name: 'Backend Agent', icon: Server, description: 'FastAPI, Gin, Express, Spring Boot generation' },
      { name: 'Frontend Agent', icon: Code, description: 'React, Vue, Angular application creation' },
      { name: 'Database Agent', icon: Database, description: 'Schema design and migration generation' },
      { name: 'Test Agent', icon: TestTube, description: 'Unit, integration, and E2E test creation' },
    ]
  },
  {
    layer: 'Core Engine',
    description: 'Domain-agnostic kernel with LLM integration',
    components: [
      { name: 'SOC Parser', icon: GitBranch, description: 'Grammar-enforced LLM output processing' },
      { name: 'IR Compiler', icon: Zap, description: 'Natural language to structured specification' },
      { name: 'Overlay System', icon: Shield, description: 'Domain expertise and compliance injection' },
      { name: 'LLM Router', icon: Cloud, description: 'Multi-provider AI with failover & caching' },
    ]
  }
];

const metrics = [
  { label: 'Generation Speed', value: '< 30s', description: 'Parallel agent execution' },
  { label: 'Code Quality', value: '99%', description: 'Production-ready output' },
  { label: 'Languages', value: '5+', description: 'Python, Go, Node.js, Java, Rust' },
  { label: 'Frameworks', value: '15+', description: 'FastAPI, React, Spring Boot...' },
];

export function Architecture() {
  return (
    <section className="py-20 relative overflow-hidden">
      {/* Section Header */}
      <div className="max-w-7xl mx-auto px-6 mb-16">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          className="text-center max-w-3xl mx-auto"
        >
          <h2 className="text-4xl lg:text-5xl font-bold mb-6">
            Enterprise-Grade <span className="text-gradient">Architecture</span>
          </h2>
          <p className="text-xl text-text-secondary leading-relaxed">
            Built on production-proven technologies with enterprise security,
            scalability, and reliability at its core.
          </p>
        </motion.div>
      </div>

      {/* Architecture Layers */}
      <div className="max-w-7xl mx-auto px-6">
        <div className="space-y-12">
          {architectureData.map((layer, layerIndex) => (
            <motion.div
              key={layer.layer}
              initial={{ opacity: 0, y: 50 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ delay: layerIndex * 0.2 }}
              className="space-y-8"
            >
              {/* Layer Header */}
              <div className="text-center">
                <h3 className="text-2xl font-bold text-quantum-primary mb-2">
                  {layer.layer}
                </h3>
                <p className="text-text-secondary">
                  {layer.description}
                </p>
              </div>

              {/* Components Grid */}
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                {layer.components.map((component, compIndex) => {
                  const Icon = component.icon;
                  return (
                    <motion.div
                      key={component.name}
                      initial={{ opacity: 0, scale: 0.9 }}
                      whileInView={{ opacity: 1, scale: 1 }}
                      viewport={{ once: true }}
                      transition={{ delay: (layerIndex * 0.2) + (compIndex * 0.1) }}
                      whileHover={{ scale: 1.02 }}
                    >
                      <GlassPanel className="p-6 h-full hover:border-quantum-primary/30 transition-all duration-300 group">
                        <div className="text-center space-y-4">
                          <div className="w-16 h-16 mx-auto rounded-xl bg-quantum-primary/10 flex items-center justify-center group-hover:bg-quantum-primary/20 transition-colors">
                            <Icon className="w-8 h-8 text-quantum-primary" />
                          </div>
                          <div>
                            <h4 className="font-semibold text-text-primary mb-2">
                              {component.name}
                            </h4>
                            <p className="text-sm text-text-secondary leading-relaxed">
                              {component.description}
                            </p>
                          </div>
                        </div>
                      </GlassPanel>
                    </motion.div>
                  );
                })}
              </div>

              {/* Connection Lines */}
              {layerIndex < architectureData.length - 1 && (
                <motion.div
                  initial={{ opacity: 0, scaleY: 0 }}
                  whileInView={{ opacity: 1, scaleY: 1 }}
                  viewport={{ once: true }}
                  transition={{ delay: (layerIndex * 0.2) + 0.5 }}
                  className="flex justify-center"
                >
                  <div className="w-px h-16 bg-gradient-to-b from-quantum-primary/50 to-transparent" />
                </motion.div>
              )}
            </motion.div>
          ))}
        </div>
      </div>

      {/* Performance Metrics */}
      <motion.div
        initial={{ opacity: 0, y: 50 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true }}
        className="max-w-7xl mx-auto px-6 mt-20"
      >
        <GlassPanel className="p-8">
          <div className="text-center mb-8">
            <h3 className="text-2xl font-bold mb-2">Performance Metrics</h3>
            <p className="text-text-secondary">Real-world performance data</p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-8">
            {metrics.map((metric, index) => (
              <motion.div
                key={metric.label}
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ delay: index * 0.1 }}
                className="text-center"
              >
                <div className="text-4xl font-bold text-quantum-success mb-2">
                  {metric.value}
                </div>
                <div className="font-medium text-text-primary mb-1">
                  {metric.label}
                </div>
                <div className="text-sm text-text-muted">
                  {metric.description}
                </div>
              </motion.div>
            ))}
          </div>
        </GlassPanel>
      </motion.div>

      {/* Technology Stack */}
      <motion.div
        initial={{ opacity: 0, y: 50 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true }}
        className="max-w-7xl mx-auto px-6 mt-20"
      >
        <div className="text-center mb-12">
          <h3 className="text-2xl font-bold mb-4">Built With Industry Leaders</h3>
          <p className="text-text-secondary">
            Powered by enterprise-grade technologies
          </p>
        </div>

        <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-6">
          {[
            'Temporal', 'PostgreSQL', 'Redis', 'Go', 'TypeScript', 'Docker',
            'Kubernetes', 'AWS Bedrock', 'Azure OpenAI', 'Next.js', 'Tailwind', 'Grafana'
          ].map((tech, index) => (
            <motion.div
              key={tech}
              initial={{ opacity: 0, scale: 0.8 }}
              whileInView={{ opacity: 1, scale: 1 }}
              viewport={{ once: true }}
              transition={{ delay: index * 0.05 }}
              className="glass-panel p-4 text-center hover:border-quantum-primary/30 transition-all duration-300 group cursor-pointer"
            >
              <div className="text-sm font-medium text-text-secondary group-hover:text-text-primary transition-colors">
                {tech}
              </div>
            </motion.div>
          ))}
        </div>
      </motion.div>
    </section>
  );
}