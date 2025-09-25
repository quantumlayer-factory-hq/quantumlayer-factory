'use client';

import { Navigation } from '@/components/shared/Navigation';
import { Hero } from '@/components/landing/Hero';
import { Architecture } from '@/components/landing/Architecture';

export default function HomePage() {
  return (
    <main className="min-h-screen">
      <Navigation />

      {/* Add padding-top to account for fixed navigation */}
      <div className="pt-20">
        <Hero />
        <Architecture />
      </div>

      {/* Status Footer */}
      <footer className="fixed bottom-0 left-0 right-0 glass-panel rounded-none border-x-0 border-b-0 z-40">
        <div className="max-w-7xl mx-auto px-6 py-3">
          <div className="flex items-center justify-between text-sm">
            <div className="flex items-center gap-4 text-text-secondary">
              <span className="flex items-center gap-2">
                <span className="w-2 h-2 bg-quantum-success rounded-full animate-pulse"></span>
                All Systems Operational
              </span>
              <span>7/7 Agents Online</span>
              <span>Queue: 0</span>
            </div>
            <div className="flex items-center gap-4 text-text-secondary">
              <span>Temporal: Connected</span>
              <span>Latency: 42ms</span>
              <span className="hidden sm:block">Enterprise Ready</span>
            </div>
          </div>
        </div>
      </footer>
    </main>
  );
}