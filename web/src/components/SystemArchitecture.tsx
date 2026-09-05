import React from 'react'
import { motion } from 'framer-motion'

const ARCHITECTURE_NODES = [
  {
    id: 'cli',
    title: 'CLI',
    desc: 'Go CLI client for job submission, status checks, and system metrics',
    icon: '>',
    color: 'text-cyan-400',
    file: 'cmd/gotask/main.go',
    details: [
      'HTTP client for API communication',
      'Commands: submit, status, cancel, workers',
      'Default endpoint: localhost:8080',
    ],
  },
  {
    id: 'api',
    title: 'API Server',
    desc: 'HTTP server handling job lifecycle and metrics',
    icon: 'S',
    color: 'text-blue-400',
    file: 'internal/api/server.go',
    details: [
      'Endpoints: POST /jobs, GET /jobs/:id, DELETE /jobs/:id, GET /metrics, GET /health',
      'Request ID tracing with x-request-id',
      'Max payload: 10MB',
    ],
  },
  {
    id: 'worker',
    title: 'Worker Pool',
    desc: 'Concurrent goroutine workers with priority queue dispatch',
    icon: '◐',
    color: 'text-purple-400',
    file: 'internal/worker/pool.go',
    details: [
      'Goroutine per worker consuming from shared priority queue',
      'Context-based cancellation per job',
      'Exponential backoff retry (1s, 2s, 4s, ..., 30s max)',
      'Stats: active workers, total processed',
    ],
  },
  {
    id: 'queue',
    title: 'Priority Queue',
    desc: 'Binary heap priority queue with mutex synchronization',
    icon: 'Q',
    color: 'text-yellow-400',
    file: 'internal/queue/queue.go',
    details: [
      'Priority ordering (higher priority first, FIFO tiebreak)',
      'Bounded capacity with blocking enqueue',
      'Condition variable for producer/consumer sync',
      'Capacity: configurable (default 100)',
    ],
  },
  {
    id: 'storage',
    title: 'SQLite Storage',
    desc: 'Persistent job state in SQLite with transactional updates',
    icon: 'DB',
    color: 'text-green-400',
    file: 'internal/storage/sqlite.go',
    details: [
      'Jobs table with status, attempts, timestamps',
      'Transactional state transitions',
      'Recovery of incomplete jobs on restart',
      'CountByStatus for metrics',
    ],
  },
]

export const SystemArchitecture: React.FC = () => {
  return (
    <section id="architecture" className="py-20">
      <div className="max-w-7xl mx-auto px-6">
        <motion.div
          initial={{ y: 30, opacity: 0 }}
          whileInView={{ y: 0, opacity: 1 }}
          viewport={{ once: true }}
          className="text-center mb-12"
        >
          <h2 className="text-3xl font-bold mb-4 font-mono">System Architecture</h2>
          <p className="text-textMuted max-w-2xl mx-auto">
            GoTask is built as a layered Go service. Jobs are created via CLI or API,
            enqueued into a priority queue, dispatched to goroutine workers, and
            persisted in SQLite for reliability.
          </p>
        </motion.div>

        <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-4 mb-12">
          {ARCHITECTURE_NODES.map((node, i) => (
            <motion.div
              key={node.id}
              initial={{ y: 30, opacity: 0 }}
              whileInView={{ y: 0, opacity: 1 }}
              viewport={{ once: true }}
              transition={{ delay: i * 0.1 }}
              className="bg-surfaceAlt border border-border rounded-xl p-5 hover:border-accent/50 transition-colors group"
              whileHover={{ y: -2 }}
            >
              <div className="flex items-start gap-3 mb-3">
                <div
                  className={`w-8 h-8 rounded border border-border flex items-center justify-center font-mono text-sm font-bold ${node.color.replace('text-', 'bg-')}/20 border-${node.color.replace('text-', '')}/30`}
                >
                  {node.icon}
                </div>
                <div>
                  <h3 className={`font-bold font-mono ${node.color}`}>{node.title}</h3>
                </div>
              </div>
              <p className="text-sm text-textMuted mb-3">{node.desc}</p>
              <div className="text-xs font-mono text-textDim mb-2 break-all">{node.file}</div>
              <div className="space-y-0.5">
                {node.details.map((detail, j) => (
                  <div key={j} className="text-xs text-textDim flex items-center">
                    <span className="text-accent mr-1">▸</span>
                    {detail}
                  </div>
                ))}
              </div>
            </motion.div>
          ))}
        </div>

        <motion.div
          initial={{ y: 30, opacity: 0 }}
          whileInView={{ y: 0, opacity: 1 }}
          viewport={{ once: true }}
          className="bg-surfaceAlt border border-border rounded-xl p-6 overflow-x-auto"
        >
          <h3 className="text-sm font-mono text-textMuted mb-3">Job Lifecycle</h3>
          <div className="font-mono text-xs text-text">
            <div className="flex items-center gap-2">
              <span className="text-cyan">Client</span>
              <span className="text-textDim">→</span>
              <span className="text-blue">API</span>
              <span className="text-textDim">→</span>
              <span className="text-yellow">PriorityQueue</span>
              <span className="text-textDim">→</span>
              <span className="text-purple">WorkerPool</span>
              <span className="text-textDim">→</span>
              <span className="text-green">SQLite</span>
            </div>
            <div className="h-px bg-border my-2" />
            <div className="text-textDim">
              <span className="text-textMuted">States:</span>{' '}
              <span className="text-yellow">queued</span> {' '} → {' '}
              <span className="text-blue">running</span> {' → '}
              <span className="text-green">succeeded</span> {' | '}
              <span className="text-red">failed</span> {' | '}
              <span className="text-textDim">cancelled</span>
            </div>
            <div className="text-textDim mt-1">
              <span className="text-textMuted">Retry:</span> failed → queued (exponential backoff)
            </div>
          </div>
        </motion.div>
      </div>
    </section>
  )
}
