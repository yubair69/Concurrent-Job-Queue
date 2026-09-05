import React from 'react'
import { motion } from 'framer-motion'

export const Header: React.FC = () => {
  return (
    <motion.header
      initial={{ y: -20, opacity: 0 }}
      animate={{ y: 0, opacity: 1 }}
      transition={{ duration: 0.3, delay: 0.05 }}
      className="fixed top-0 left-0 right-0 z-50 border-b border-border bg-bg/80 backdrop-blur-sm"
    >
      <div className="max-w-7xl mx-auto px-6 py-3 flex items-center justify-between">
        <div className="flex items-center space-x-3">
          <div className="relative">
            <div className="w-6 h-6 border-2 border-accent rounded-sm" />
            <div className="absolute inset-0.5 grid grid-cols-2 gap-0.5">
              <div className="bg-accent rounded-sm animate-pulse-slow" />
              <div className="bg-accent rounded-sm animate-pulse-slow delay-75" />
              <div className="bg-accent rounded-sm animate-pulse-slow delay-150" />
              <div className="bg-accent rounded-sm animate-pulse-slow delay-300" />
            </div>
          </div>
          <span className="font-mono text-sm font-bold text-text">
            Go<span className="text-accent">Task</span>
          </span>
        </div>

        <nav className="flex items-center space-x-8 text-sm font-mono">
          <a href="#system" className="text-textMuted hover:text-text transition-colors relative group">
            System
            <span className="absolute -bottom-1 left-0 w-0 h-px bg-accent transition-all group-hover:w-full" />
          </a>
          <a href="#concurrency" className="text-textMuted hover:text-text transition-colors relative group">
            Concurrency
            <span className="absolute -bottom-1 left-0 w-0 h-px bg-accent transition-all group-hover:w-full" />
          </a>
          <a href="#jobs" className="text-textMuted hover:text-text transition-colors relative group">
            Jobs
            <span className="absolute -bottom-1 left-0 w-0 h-px bg-accent transition-all group-hover:w-full" />
          </a>
          <a href="#terminal" className="text-textMuted hover:text-text transition-colors relative group">
            CLI
            <span className="absolute -bottom-1 left-0 w-0 h-px bg-accent transition-all group-hover:w-full" />
          </a>
          <a
            href="https://github.com/gotask/gotask"
            target="_blank"
            rel="noopener noreferrer"
            className="text-textMuted hover:text-accent transition-colors"
          >
            GitHub
          </a>
        </nav>
      </div>
    </motion.header>
  )
}
