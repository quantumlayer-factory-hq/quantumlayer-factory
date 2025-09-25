'use client';

import { useState } from 'react';
import { motion } from 'framer-motion';
import { Play, Github, ExternalLink, Star, GitFork, Clock, Code, Database, Server } from 'lucide-react';
import { GlassPanel } from '@/components/glass/GlassPanel';
import { GlassButton } from '@/components/glass/GlassButton';

interface Project {
  id: string;
  title: string;
  description: string;
  tags: string[];
  language: string;
  framework: string;
  database?: string;
  generationTime: string;
  filesCount: number;
  stars: number;
  forks: number;
  thumbnail: string;
  demoUrl?: string;
  githubUrl?: string;
  featured?: boolean;
}

const projects: Project[] = [
  {
    id: '1',
    title: 'E-Commerce REST API',
    description: 'Complete RESTful API with authentication, product catalog, shopping cart, and payment processing using Stripe integration.',
    tags: ['API', 'Authentication', 'Payments', 'Database'],
    language: 'Python',
    framework: 'FastAPI',
    database: 'PostgreSQL',
    generationTime: '24.3s',
    filesCount: 18,
    stars: 142,
    forks: 23,
    thumbnail: '/api/placeholder/400/250',
    demoUrl: 'https://demo.example.com',
    githubUrl: 'https://github.com/example/ecommerce-api',
    featured: true
  },
  {
    id: '2',
    title: 'React Dashboard',
    description: 'Modern admin dashboard with real-time analytics, user management, and responsive design using Tailwind CSS.',
    tags: ['Dashboard', 'Analytics', 'Real-time', 'Responsive'],
    language: 'TypeScript',
    framework: 'React',
    generationTime: '31.7s',
    filesCount: 24,
    stars: 98,
    forks: 15,
    thumbnail: '/api/placeholder/400/250',
    demoUrl: 'https://dashboard.example.com',
    featured: true
  },
  {
    id: '3',
    title: 'Microservices Architecture',
    description: 'Scalable microservices setup with API Gateway, service discovery, and containerized deployment.',
    tags: ['Microservices', 'Docker', 'API Gateway', 'Kubernetes'],
    language: 'Go',
    framework: 'Gin',
    database: 'MongoDB',
    generationTime: '42.1s',
    filesCount: 35,
    stars: 87,
    forks: 12,
    thumbnail: '/api/placeholder/400/250',
    githubUrl: 'https://github.com/example/microservices'
  },
  {
    id: '4',
    title: 'Chat Application',
    description: 'Real-time chat application with WebSocket support, user authentication, and message history.',
    tags: ['WebSocket', 'Real-time', 'Chat', 'Authentication'],
    language: 'JavaScript',
    framework: 'Express',
    database: 'Redis',
    generationTime: '28.9s',
    filesCount: 16,
    stars: 156,
    forks: 31,
    thumbnail: '/api/placeholder/400/250'
  },
  {
    id: '5',
    title: 'Task Management System',
    description: 'Comprehensive task tracking with project management features, team collaboration, and deadline tracking.',
    tags: ['Project Management', 'Collaboration', 'Tasks', 'Teams'],
    language: 'Python',
    framework: 'Django',
    database: 'PostgreSQL',
    generationTime: '36.4s',
    filesCount: 28,
    stars: 73,
    forks: 9,
    thumbnail: '/api/placeholder/400/250'
  },
  {
    id: '6',
    title: 'Blockchain Voting System',
    description: 'Decentralized voting platform with smart contracts, voter authentication, and transparent results.',
    tags: ['Blockchain', 'Smart Contracts', 'Voting', 'Ethereum'],
    language: 'Solidity',
    framework: 'Hardhat',
    generationTime: '45.2s',
    filesCount: 22,
    stars: 201,
    forks: 47,
    thumbnail: '/api/placeholder/400/250',
    featured: true
  }
];

const languageColors: Record<string, string> = {
  Python: 'bg-blue-500',
  TypeScript: 'bg-blue-600',
  JavaScript: 'bg-yellow-500',
  Go: 'bg-cyan-500',
  Solidity: 'bg-purple-500',
  Rust: 'bg-orange-500',
  Java: 'bg-red-500'
};

export default function GalleryPage() {
  const [selectedFilter, setSelectedFilter] = useState('all');
  const [searchTerm, setSearchTerm] = useState('');

  const filters = ['all', 'featured', 'api', 'frontend', 'blockchain', 'microservices'];

  const filteredProjects = projects.filter(project => {
    const matchesFilter = selectedFilter === 'all' ||
      (selectedFilter === 'featured' && project.featured) ||
      project.tags.some(tag => tag.toLowerCase().includes(selectedFilter));

    const matchesSearch = searchTerm === '' ||
      project.title.toLowerCase().includes(searchTerm.toLowerCase()) ||
      project.description.toLowerCase().includes(searchTerm.toLowerCase()) ||
      project.tags.some(tag => tag.toLowerCase().includes(searchTerm.toLowerCase()));

    return matchesFilter && matchesSearch;
  });

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
            Project <span className="text-gradient">Gallery</span>
          </motion.h1>
          <motion.p
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.1 }}
            className="text-xl text-text-secondary max-w-3xl mx-auto"
          >
            Explore applications built with QuantumLayer Factory. From simple APIs to complex
            microservices architectures, see what our AI agents can create.
          </motion.p>
        </div>

        {/* Filters & Search */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.2 }}
          className="mb-8"
        >
          <GlassPanel className="p-6">
            <div className="flex flex-col md:flex-row gap-6 items-center">
              {/* Search */}
              <div className="flex-1 max-w-md">
                <input
                  type="text"
                  placeholder="Search projects..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="w-full bg-white/5 border border-white/10 rounded-lg px-4 py-2 text-text-primary placeholder-text-muted focus:outline-none focus:border-quantum-primary/50"
                />
              </div>

              {/* Filters */}
              <div className="flex gap-2 flex-wrap">
                {filters.map((filter) => (
                  <button
                    key={filter}
                    onClick={() => setSelectedFilter(filter)}
                    className={`px-4 py-2 rounded-lg text-sm font-medium transition-all ${
                      selectedFilter === filter
                        ? 'bg-quantum-primary/20 text-quantum-primary border border-quantum-primary/50'
                        : 'bg-white/5 text-text-secondary hover:bg-white/10 border border-white/10'
                    }`}
                  >
                    {filter.charAt(0).toUpperCase() + filter.slice(1)}
                  </button>
                ))}
              </div>
            </div>
          </GlassPanel>
        </motion.div>

        {/* Projects Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-8">
          {filteredProjects.map((project, index) => (
            <motion.div
              key={project.id}
              initial={{ opacity: 0, y: 30 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.3 + index * 0.1 }}
              whileHover={{ y: -5 }}
            >
              <GlassPanel className="h-full overflow-hidden group hover:border-quantum-primary/30 transition-all duration-300">
                {/* Thumbnail */}
                <div className="aspect-video bg-gradient-to-br from-quantum-primary/10 to-quantum-purple/10 relative overflow-hidden">
                  {project.featured && (
                    <div className="absolute top-4 left-4 z-10">
                      <div className="bg-quantum-primary/20 backdrop-blur-sm border border-quantum-primary/50 rounded-full px-3 py-1">
                        <Star className="w-4 h-4 text-quantum-primary inline mr-1" />
                        <span className="text-xs font-medium text-quantum-primary">Featured</span>
                      </div>
                    </div>
                  )}

                  {/* Tech Stack Icons */}
                  <div className="absolute bottom-4 left-4 flex items-center gap-2">
                    <div className={`w-8 h-8 rounded-full ${languageColors[project.language]} bg-opacity-20 flex items-center justify-center`}>
                      <Code className="w-4 h-4 text-white" />
                    </div>
                    {project.database && (
                      <div className="w-8 h-8 rounded-full bg-green-500 bg-opacity-20 flex items-center justify-center">
                        <Database className="w-4 h-4 text-green-400" />
                      </div>
                    )}
                    <div className="w-8 h-8 rounded-full bg-purple-500 bg-opacity-20 flex items-center justify-center">
                      <Server className="w-4 h-4 text-purple-400" />
                    </div>
                  </div>
                </div>

                {/* Content */}
                <div className="p-6">
                  {/* Title & Description */}
                  <div className="mb-4">
                    <h3 className="text-xl font-bold text-text-primary mb-2 group-hover:text-quantum-primary transition-colors">
                      {project.title}
                    </h3>
                    <p className="text-text-secondary text-sm leading-relaxed line-clamp-3">
                      {project.description}
                    </p>
                  </div>

                  {/* Tags */}
                  <div className="flex flex-wrap gap-2 mb-4">
                    {project.tags.slice(0, 3).map((tag) => (
                      <span
                        key={tag}
                        className="px-2 py-1 bg-white/5 rounded-md text-xs text-text-muted"
                      >
                        {tag}
                      </span>
                    ))}
                    {project.tags.length > 3 && (
                      <span className="px-2 py-1 text-xs text-text-muted">
                        +{project.tags.length - 3}
                      </span>
                    )}
                  </div>

                  {/* Tech Info */}
                  <div className="grid grid-cols-2 gap-4 mb-4 text-sm">
                    <div>
                      <div className="text-text-muted">Language</div>
                      <div className="text-text-primary font-medium">{project.language}</div>
                    </div>
                    <div>
                      <div className="text-text-muted">Framework</div>
                      <div className="text-text-primary font-medium">{project.framework}</div>
                    </div>
                  </div>

                  {/* Stats */}
                  <div className="flex items-center justify-between text-xs text-text-muted mb-4">
                    <div className="flex items-center gap-4">
                      <span className="flex items-center gap-1">
                        <Clock className="w-3 h-3" />
                        {project.generationTime}
                      </span>
                      <span className="flex items-center gap-1">
                        <Code className="w-3 h-3" />
                        {project.filesCount} files
                      </span>
                    </div>
                    <div className="flex items-center gap-3">
                      <span className="flex items-center gap-1">
                        <Star className="w-3 h-3" />
                        {project.stars}
                      </span>
                      <span className="flex items-center gap-1">
                        <GitFork className="w-3 h-3" />
                        {project.forks}
                      </span>
                    </div>
                  </div>

                  {/* Actions */}
                  <div className="flex gap-2">
                    {project.demoUrl && (
                      <GlassButton size="sm" className="flex-1 text-quantum-primary">
                        <Play className="w-4 h-4 mr-2" />
                        Demo
                      </GlassButton>
                    )}
                    {project.githubUrl && (
                      <GlassButton variant="secondary" size="sm" className="flex-1">
                        <Github className="w-4 h-4 mr-2" />
                        Code
                      </GlassButton>
                    )}
                    <GlassButton variant="secondary" size="sm" className="px-3">
                      <ExternalLink className="w-4 h-4" />
                    </GlassButton>
                  </div>
                </div>
              </GlassPanel>
            </motion.div>
          ))}
        </div>

        {/* Empty State */}
        {filteredProjects.length === 0 && (
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            className="text-center py-20"
          >
            <div className="text-6xl mb-4">🔍</div>
            <h3 className="text-2xl font-bold text-text-primary mb-2">No projects found</h3>
            <p className="text-text-secondary">Try adjusting your search or filter criteria</p>
          </motion.div>
        )}

        {/* CTA Section */}
        <motion.div
          initial={{ opacity: 0, y: 30 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.5 }}
          className="mt-20"
        >
          <GlassPanel className="p-8 text-center">
            <h2 className="text-3xl font-bold mb-4">Ready to Build Your Own?</h2>
            <p className="text-text-secondary mb-6 max-w-2xl mx-auto">
              Join thousands of developers using QuantumLayer Factory to generate
              production-ready applications in seconds.
            </p>
            <GlassButton size="lg" className="px-8 py-4 bg-quantum-primary/20 hover:bg-quantum-primary/30 border-quantum-primary/50">
              <Play className="w-5 h-5 mr-2" />
              Start Building
            </GlassButton>
          </GlassPanel>
        </motion.div>
      </div>
    </div>
  );
}