'use client';

import { useState } from 'react';
import Link from 'next/link';
import { motion } from 'framer-motion';
import { Zap, Play, Grid, FileText, BarChart3, Menu, X } from 'lucide-react';

const navItems = [
  { href: '/', label: 'Platform', icon: Zap },
  { href: '/playground', label: 'Playground', icon: Play },
  { href: '/gallery', label: 'Gallery', icon: Grid },
  { href: '/docs', label: 'Docs', icon: FileText },
  { href: '/analytics', label: 'Analytics', icon: BarChart3 },
];

export function Navigation() {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <>
      {/* Desktop Navigation */}
      <nav className="fixed top-0 left-0 right-0 z-50 glass-panel rounded-none border-x-0 border-t-0">
        <div className="max-w-7xl mx-auto px-6 py-4">
          <div className="flex items-center justify-between">
            {/* Logo */}
            <Link href="/" className="flex items-center gap-3 group">
              <div className="w-10 h-10 rounded-xl bg-gradient-to-r from-quantum-primary to-quantum-purple flex items-center justify-center group-hover:scale-105 transition-transform">
                <Zap className="w-6 h-6 text-white" />
              </div>
              <div className="hidden sm:block">
                <div className="text-xl font-bold text-gradient">QuantumLayer</div>
                <div className="text-xs text-text-muted -mt-1">Enterprise Platform</div>
              </div>
            </Link>

            {/* Desktop Menu */}
            <div className="hidden md:flex items-center gap-1">
              {navItems.map((item) => {
                const Icon = item.icon;
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    className="flex items-center gap-2 px-4 py-2 rounded-lg text-text-secondary hover:text-text-primary hover:bg-white/5 transition-all duration-200 group"
                  >
                    <Icon className="w-4 h-4 group-hover:scale-110 transition-transform" />
                    <span className="text-sm font-medium">{item.label}</span>
                  </Link>
                );
              })}
            </div>

            {/* CTA Button */}
            <div className="hidden md:flex items-center gap-4">
              <div className="flex items-center gap-2 px-3 py-1 rounded-full bg-quantum-success/10 text-quantum-success text-xs">
                <div className="w-2 h-2 bg-quantum-success rounded-full animate-pulse" />
                Online
              </div>
              <Link
                href="/playground"
                className="glass-button px-6 py-2 text-sm font-medium text-quantum-primary hover:bg-quantum-primary/10"
              >
                Try Demo
              </Link>
            </div>

            {/* Mobile Menu Button */}
            <button
              onClick={() => setIsOpen(!isOpen)}
              className="md:hidden glass-button p-2"
            >
              {isOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
            </button>
          </div>
        </div>
      </nav>

      {/* Mobile Menu */}
      <motion.div
        initial={false}
        animate={{
          opacity: isOpen ? 1 : 0,
          y: isOpen ? 0 : -20,
          visibility: isOpen ? 'visible' : 'hidden'
        }}
        className="fixed top-20 left-0 right-0 z-40 md:hidden"
      >
        <div className="mx-4 glass-panel p-4">
          <div className="space-y-2">
            {navItems.map((item) => {
              const Icon = item.icon;
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  onClick={() => setIsOpen(false)}
                  className="flex items-center gap-3 px-4 py-3 rounded-lg text-text-secondary hover:text-text-primary hover:bg-white/5 transition-all duration-200"
                >
                  <Icon className="w-5 h-5" />
                  <span className="font-medium">{item.label}</span>
                </Link>
              );
            })}
            <div className="border-t border-white/10 pt-2 mt-2">
              <Link
                href="/playground"
                onClick={() => setIsOpen(false)}
                className="glass-button w-full justify-center py-3 text-quantum-primary font-medium"
              >
                Try Demo
              </Link>
            </div>
          </div>
        </div>
      </motion.div>

      {/* Backdrop */}
      {isOpen && (
        <div
          className="fixed inset-0 z-30 bg-black/20 backdrop-blur-sm md:hidden"
          onClick={() => setIsOpen(false)}
        />
      )}
    </>
  );
}