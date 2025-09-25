'use client';

import { useState } from 'react';
import { motion } from 'framer-motion';
import { Search, Book, ChevronRight, Code2, Server, Database, Shield, Zap, Play, Download } from 'lucide-react';
import { GlassPanel } from '@/components/glass/GlassPanel';
import { GlassButton } from '@/components/glass/GlassButton';

interface DocSection {
  id: string;
  title: string;
  description: string;
  icon: any;
  color: string;
  articles: DocArticle[];
}

interface DocArticle {
  id: string;
  title: string;
  description: string;
  readTime: string;
  difficulty: 'Beginner' | 'Intermediate' | 'Advanced';
}

const docSections: DocSection[] = [
  {
    id: 'getting-started',
    title: 'Getting Started',
    description: 'Quick setup and your first application',
    icon: Play,
    color: 'bg-green-500',
    articles: [
      {
        id: 'quickstart',
        title: 'Quickstart Guide',
        description: 'Generate your first application in under 5 minutes',
        readTime: '5 min',
        difficulty: 'Beginner'
      },
      {
        id: 'installation',
        title: 'Installation & Setup',
        description: 'Install QuantumLayer Factory and configure your environment',
        readTime: '10 min',
        difficulty: 'Beginner'
      },
      {
        id: 'first-project',
        title: 'Your First Project',
        description: 'Step-by-step guide to creating a complete application',
        readTime: '15 min',
        difficulty: 'Beginner'
      }
    ]
  },
  {
    id: 'architecture',
    title: 'Architecture',
    description: 'Understanding the multi-agent system',
    icon: Server,
    color: 'bg-blue-500',
    articles: [
      {
        id: 'overview',
        title: 'System Overview',
        description: 'High-level architecture and component interaction',
        readTime: '8 min',
        difficulty: 'Intermediate'
      },
      {
        id: 'agents',
        title: 'Agent System',
        description: 'Deep dive into the 7 specialized agents',
        readTime: '12 min',
        difficulty: 'Intermediate'
      },
      {
        id: 'temporal',
        title: 'Temporal Workflows',
        description: 'How workflows orchestrate agent execution',
        readTime: '15 min',
        difficulty: 'Advanced'
      },
      {
        id: 'llm-integration',
        title: 'LLM Integration',
        description: 'Multi-provider AI with failover and caching',
        readTime: '10 min',
        difficulty: 'Intermediate'
      }
    ]
  },
  {
    id: 'development',
    title: 'Development',
    description: 'Building and customizing applications',
    icon: Code2,
    color: 'bg-purple-500',
    articles: [
      {
        id: 'prompts',
        title: 'Writing Effective Prompts',
        description: 'Best practices for describing your application',
        readTime: '12 min',
        difficulty: 'Beginner'
      },
      {
        id: 'customization',
        title: 'Customizing Output',
        description: 'Configure agents and modify generation behavior',
        readTime: '18 min',
        difficulty: 'Intermediate'
      },
      {
        id: 'templates',
        title: 'Custom Templates',
        description: 'Create reusable templates for your team',
        readTime: '20 min',
        difficulty: 'Advanced'
      }
    ]
  },
  {
    id: 'deployment',
    title: 'Deployment',
    description: 'Production deployment strategies',
    icon: Shield,
    color: 'bg-orange-500',
    articles: [
      {
        id: 'docker',
        title: 'Docker Deployment',
        description: 'Containerize and deploy your applications',
        readTime: '15 min',
        difficulty: 'Intermediate'
      },
      {
        id: 'kubernetes',
        title: 'Kubernetes',
        description: 'Scalable deployment with orchestration',
        readTime: '25 min',
        difficulty: 'Advanced'
      },
      {
        id: 'monitoring',
        title: 'Monitoring & Observability',
        description: 'Production monitoring and debugging',
        readTime: '20 min',
        difficulty: 'Advanced'
      }
    ]
  }
];

const quickLinks = [
  { title: 'API Reference', description: 'Complete API documentation', icon: Database },
  { title: 'CLI Commands', description: 'Command-line interface guide', icon: Zap },
  { title: 'Configuration', description: 'System configuration options', icon: Server },
  { title: 'Troubleshooting', description: 'Common issues and solutions', icon: Shield }
];

const difficultyColors = {
  'Beginner': 'bg-green-500/20 text-green-400 border-green-500/30',
  'Intermediate': 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30',
  'Advanced': 'bg-red-500/20 text-red-400 border-red-500/30'
};

export default function DocsPage() {
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedSection, setSelectedSection] = useState<string | null>(null);

  const filteredSections = docSections.filter(section =>
    searchTerm === '' ||
    section.title.toLowerCase().includes(searchTerm.toLowerCase()) ||
    section.description.toLowerCase().includes(searchTerm.toLowerCase()) ||
    section.articles.some(article =>
      article.title.toLowerCase().includes(searchTerm.toLowerCase()) ||
      article.description.toLowerCase().includes(searchTerm.toLowerCase())
    )
  );

  return (
    <div className="min-h-screen pt-20 pb-20">
      <div className="max-w-7xl mx-auto px-6 py-8">
        {/* Header */}
        <div className="text-center mb-12">
          <motion.h1
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            className="text-4xl lg:text-5xl font-bold mb-6"
          >
            <span className="text-gradient">Documentation</span>
          </motion.h1>
          <motion.p
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.1 }}
            className="text-xl text-text-secondary max-w-3xl mx-auto"
          >
            Everything you need to know about QuantumLayer Factory.
            From quickstart guides to advanced architecture deep-dives.
          </motion.p>
        </div>

        {/* Search */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.2 }}
          className="mb-8"
        >
          <div className="max-w-2xl mx-auto relative">
            <Search className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-text-muted" />
            <input
              type="text"
              placeholder="Search documentation..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full bg-white/5 border border-white/10 rounded-lg pl-12 pr-4 py-4 text-text-primary placeholder-text-muted focus:outline-none focus:border-quantum-primary/50 backdrop-blur-sm"
            />
          </div>
        </motion.div>

        <div className="grid grid-cols-1 lg:grid-cols-4 gap-8">
          {/* Sidebar - Quick Links */}
          <div className="lg:col-span-1">
            <motion.div
              initial={{ opacity: 0, x: -20 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ delay: 0.3 }}
            >
              <GlassPanel className="p-6 mb-6">
                <h2 className="text-lg font-semibold mb-4 text-text-primary">Quick Links</h2>
                <div className="space-y-3">
                  {quickLinks.map((link, index) => {
                    const Icon = link.icon;
                    return (
                      <motion.button
                        key={link.title}
                        initial={{ opacity: 0, x: -10 }}
                        animate={{ opacity: 1, x: 0 }}
                        transition={{ delay: 0.4 + index * 0.05 }}
                        className="w-full text-left p-3 rounded-lg bg-white/5 hover:bg-white/10 border border-white/10 hover:border-quantum-primary/30 transition-all group"
                      >
                        <div className="flex items-start gap-3">
                          <Icon className="w-5 h-5 text-quantum-primary mt-0.5 group-hover:scale-110 transition-transform" />
                          <div>
                            <div className="font-medium text-text-primary text-sm">{link.title}</div>
                            <div className="text-text-muted text-xs mt-1">{link.description}</div>
                          </div>
                        </div>
                      </motion.button>
                    );
                  })}
                </div>
              </GlassPanel>

              {/* Download Resources */}
              <motion.div
                initial={{ opacity: 0, x: -20 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ delay: 0.5 }}
              >
                <GlassPanel className="p-6">
                  <h3 className="font-semibold mb-4 text-text-primary">Resources</h3>
                  <div className="space-y-3">
                    <GlassButton size="sm" className="w-full text-left justify-start">
                      <Download className="w-4 h-4 mr-2" />
                      PDF Guide
                    </GlassButton>
                    <GlassButton variant="secondary" size="sm" className="w-full text-left justify-start">
                      <Book className="w-4 h-4 mr-2" />
                      Examples
                    </GlassButton>
                  </div>
                </GlassPanel>
              </motion.div>
            </motion.div>
          </div>

          {/* Main Content */}
          <div className="lg:col-span-3">
            <div className="space-y-8">
              {filteredSections.map((section, sectionIndex) => {
                const Icon = section.icon;
                return (
                  <motion.div
                    key={section.id}
                    initial={{ opacity: 0, y: 30 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: 0.3 + sectionIndex * 0.1 }}
                  >
                    <GlassPanel className="overflow-hidden">
                      {/* Section Header */}
                      <div className="p-6 border-b border-white/10">
                        <div className="flex items-center gap-4">
                          <div className={`w-12 h-12 rounded-xl ${section.color} bg-opacity-20 flex items-center justify-center`}>
                            <Icon className="w-6 h-6 text-white" />
                          </div>
                          <div>
                            <h2 className="text-2xl font-bold text-text-primary">{section.title}</h2>
                            <p className="text-text-secondary">{section.description}</p>
                          </div>
                        </div>
                      </div>

                      {/* Articles */}
                      <div className="p-6">
                        <div className="grid gap-4">
                          {section.articles.map((article, articleIndex) => (
                            <motion.button
                              key={article.id}
                              initial={{ opacity: 0, x: -20 }}
                              animate={{ opacity: 1, x: 0 }}
                              transition={{ delay: 0.4 + sectionIndex * 0.1 + articleIndex * 0.05 }}
                              className="w-full text-left p-4 rounded-lg bg-white/5 hover:bg-white/10 border border-white/10 hover:border-quantum-primary/30 transition-all group"
                            >
                              <div className="flex items-center justify-between">
                                <div className="flex-1">
                                  <div className="flex items-center gap-3 mb-2">
                                    <h3 className="font-semibold text-text-primary group-hover:text-quantum-primary transition-colors">
                                      {article.title}
                                    </h3>
                                    <div className={`px-2 py-1 rounded-full text-xs border ${difficultyColors[article.difficulty]}`}>
                                      {article.difficulty}
                                    </div>
                                  </div>
                                  <p className="text-text-secondary text-sm mb-2">{article.description}</p>
                                  <div className="text-text-muted text-xs">{article.readTime} read</div>
                                </div>
                                <ChevronRight className="w-5 h-5 text-text-muted group-hover:text-quantum-primary group-hover:translate-x-1 transition-all" />
                              </div>
                            </motion.button>
                          ))}
                        </div>
                      </div>
                    </GlassPanel>
                  </motion.div>
                );
              })}
            </div>

            {/* Empty State */}
            {filteredSections.length === 0 && (
              <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                className="text-center py-20"
              >
                <div className="text-6xl mb-4">📚</div>
                <h3 className="text-2xl font-bold text-text-primary mb-2">No documentation found</h3>
                <p className="text-text-secondary">Try searching for something else</p>
              </motion.div>
            )}

            {/* Community Section */}
            <motion.div
              initial={{ opacity: 0, y: 30 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.6 }}
              className="mt-12"
            >
              <GlassPanel className="p-8 text-center">
                <h2 className="text-2xl font-bold mb-4">Need More Help?</h2>
                <p className="text-text-secondary mb-6 max-w-2xl mx-auto">
                  Join our community of developers building with QuantumLayer Factory.
                  Get help, share projects, and contribute to the ecosystem.
                </p>
                <div className="flex flex-col sm:flex-row gap-4 justify-center">
                  <GlassButton size="lg" className="px-6 py-3 bg-quantum-primary/20 hover:bg-quantum-primary/30 border-quantum-primary/50">
                    Join Discord Community
                  </GlassButton>
                  <GlassButton variant="secondary" size="lg" className="px-6 py-3">
                    GitHub Discussions
                  </GlassButton>
                </div>
              </GlassPanel>
            </motion.div>
          </div>
        </div>
      </div>
    </div>
  );
}