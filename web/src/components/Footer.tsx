import React from 'react'
import { motion } from 'framer-motion'

export const Footer: React.FC = () => {
  return (
    <motion.footer
      initial={{ y: 30, opacity: 0 }}
      whileInView={{ y: 0, opacity: 1 }}
      viewport={{ once: true }}
      className="border-t border-border py-12 mt-20"
    >
      <div className="max-w-7xl mx-auto px-6">
        <div className="grid md:grid-cols-3 gap-8">
          <div>
            <div className="flex items-center space-x-2 mb-3">
              <div className="w-5 h-5 border-2 border-accent rounded-sm flex items-center justify-center">
                <div className="w-2 h-2 bg-accent animate-pulse-slow" />
              </div>
              <span className="font-mono text-sm font-bold">
                Go<span className="text-accent">Task</span>
              </span>
            </div>
            <p className="text-sm text-textMuted">
              A concurrent background job processing engine for Go.
            </p>
          </div>

          <div className="space-y-2">
            <h4 className="text-xs font-mono text-textMuted">Navigation</h4>
            <div className="space-y-1">
              <a href="#system" className="block text-sm text-textMuted hover:text-text transition-colors">Live System</a>
              <a href="#concurrency" className="block text-sm text-textMuted hover:text-text transition-colors">Concurrency</a>
              <a href="#jobs" className="block text-sm text-textMuted hover:text-text transition-colors">Jobs</a>
              <a href="#terminal" className="block text-sm text-textMuted hover:text-text transition-colors">CLI</a>
            </div>
          </div>

          <div className="space-y-2">
            <h4 className="text-xs font-mono text-textMuted">Technology</h4>
            <div className="space-y-1">
              <div className="text-sm text-textMuted">Go · React · TypeScript · Tailwind</div>
              <div className="text-sm text-textDim">SQLite · Goroutine · Priority Queue</div>
            </div>
          </div>
        </div>

        <div className="border-t border-border mt-8 pt-4 flex items-center justify-between text-xs text-textDim font-mono">
          <div>GoTask 1.0.0</div>
          <div className="flex gap-2">
            <a
              href="https://github.com/gotask/gotask"
              target="_blank"
              rel="noopener noreferrer"
              className="hover:text-accent transition-colors"
            >
              GitHub
            </a>
            <span>•</span>
            <span>MIT License</span>
          </div>
        </div>
      </div>
    </motion.footer>
  )
}
