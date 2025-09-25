'use client';

import { ReactNode, HTMLAttributes } from 'react';
import { motion } from 'framer-motion';

interface GlassPanelProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode;
  elevated?: boolean;
  interactive?: boolean;
}

export function GlassPanel({
  children,
  elevated = false,
  interactive = false,
  className = '',
  ...props
}: GlassPanelProps) {
  const baseClasses = 'glass-panel';
  const elevatedClasses = elevated ? 'shadow-2xl' : '';
  const interactiveClasses = interactive ? 'hover:bg-white/10 cursor-pointer transition-all duration-300' : '';

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5 }}
      className={`${baseClasses} ${elevatedClasses} ${interactiveClasses} ${className}`}
      {...props}
    >
      {children}
    </motion.div>
  );
}