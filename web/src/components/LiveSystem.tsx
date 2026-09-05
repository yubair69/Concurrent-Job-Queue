import React, { useEffect, useRef } from 'react'
import { motion } from 'framer-motion'
import { useSystem, WorkerState } from '@/hooks/useSystem'

const STAGE_NAMES = ['Queued', 'Dispatched', 'Running', 'Succeeded', 'Failed'] as const

const WORKER_POSITIONS: Array<{ id: number; x: number; y: number }> = [
  { id: 1, x: 760, y: 120 },
  { id: 2, x: 760, y: 220 },
  { id: 3, x: 760, y: 320 },
  { id: 4, x: 760, y: 420 },
]

const QUEUE_POSITIONS: Array<{ id: string; x: number; y: number }> = [
  { id: 'q1', x: 200, y: 220 },
  { id: 'q2', x: 200, y: 270 },
  { id: 'q3', x: 200, y: 320 },
  { id: 'q4', x: 200, y: 370 },
]

const STATUS_COLORS: Record<string, string> = {
  queued: '#fbbf24',
  running: '#60a5fa',
  succeeded: '#4ade80',
  failed: '#f87171',
  cancelled: '#7a7a8a',
}

const WORKER_STATES: Record<WorkerState, { color: string; label: string; dotColor: string }> = {
  idle: { color: '#0f0f15', label: 'Idle', dotColor: '#7a7a8a' },
  running: { color: '#1a1a22', label: 'Running', dotColor: '#00d4ff' },
  stopping: { color: '#1a1212', label: 'Stopping', dotColor: '#fbbf24' },
}

export const LiveSystem: React.FC = () => {
  const { jobs, workers, queueDepth, activeWorkers, totalProcessed, enqueueJob, cancelJob } = useSystem(4)
  const animationRefs = useRef<Map<string, number>>(new Map())
  const animationFrameRef = useRef<number | null>(null)

  const handleEnqueue = () => {
    const types = ['echo', 'sleep', 'email', 'image-processing']
    const type = types[Math.floor(Math.random() * types.length)]
    const priority = [1, 3, 5][Math.floor(Math.random() * 3)]
    enqueueJob(type, priority)
  }

  const handleCancel = (jobId: string) => {
    cancelJob(jobId)
  }

  return (
    <section id="system" className="py-20 bg-bgAlt">
      <div className="max-w-7xl mx-auto px-6">
        <div className="grid lg:grid-cols-5 gap-6 mb-6">
          <div className="lg:col-span-2">
            <h2 className="text-2xl font-bold mb-1 font-mono">Live System</h2>
            <p className="text-textMuted text-sm">
              Priority queue dispatching to {workers.length} goroutine workers. Jobs persist to SQLite with automatic retries.
            </p>
          </div>
          <div className="lg:col-span-3 flex gap-6">
            <div className="flex items-center gap-4 text-sm font-mono">
              <div>
                <span className="text-yellow-400">{queueDepth}</span>
                <span className="text-textDim">queued</span>
              </div>
              <div>
                <span className="text-blue-400">{activeWorkers}</span>
                <span className="text-textDim">running</span>
              </div>
              <div>
                <span className="text-green-400">{totalProcessed}</span>
                <span className="text-textDim">done</span>
              </div>
            </div>
            <motion.button
              whileHover={{ scale: 1.02 }}
              whileTap={{ scale: 0.98 }}
              onClick={handleEnqueue}
              className="ml-auto px-3 py-1.5 bg-accent text-bg font-mono text-xs font-bold rounded hover:bg-accentHover transition-colors shadow-lg shadow-accent/20"
            >
              + Enqueue Job
            </motion.button>
          </div>
        </div>

        <div className="bg-surfaceAlt border border-border rounded-xl p-6 overflow-hidden">
          <svg
            width="100%"
            height="560"
            viewBox="0 0 1000 560"
            className="w-full"
            xmlns="http://www.w3.org/2000/svg"
          >
            <defs>
              <filter id="glow-cyan" x="-50%" y="-50%" width="200%" height="200">
                <feGaussianBlur stdDeviation="3" result="coloredBlur" />
                <feMerge>
                  <feMergeNode in="coloredBlur" />
                  <feMergeNode in="SourceGraphic" />
                </feMerge>
              </filter>
              <filter id="glow-green" x="-50%" y="-50%" width="200%" height="200">
                <feGaussianBlur stdDeviation="4" result="coloredBlur" />
                <feMerge>
                  <feMergeNode in="coloredBlur" />
                  <feMergeNode in="SourceGraphic" />
                </feMerge>
              </filter>
              <filter id="glow-yellow" x="-50%" y="-50%" width="200%" height="200">
                <feGaussianBlur stdDeviation="3" result="coloredBlur" />
                <feMerge>
                  <feMergeNode in="coloredBlur" />
                  <feMergeNode in="SourceGraphic" />
                </feMerge>
              </filter>
            </defs>

            <motion.path
              d="M 240 240 L 320 240"
              stroke="#2a2a3a"
              strokeWidth="2"
              fill="none"
              strokeDasharray="6 3"
            />
            <motion.path
              d="M 420 240 L 560 240"
              stroke="#2a2a3a"
              strokeWidth="2"
              fill="none"
              strokeDasharray="6 3"
            />
            <motion.path
              d="M 640 120 L 700 120"
              stroke="#2a2a3a"
              strokeWidth="2"
              fill="none"
              strokeDasharray="6 3"
            />
            <motion.path
              d="M 420 320 L 560 320"
              stroke="#2a2a3a"
              strokeWidth="2"
              fill="none"
              strokeDasharray="6 3"
            />
            <motion.path
              d="M 640 320 L 700 320"
              stroke="#2a2a3a"
              strokeWidth="2"
              fill="none"
              strokeDasharray="6 3"
            />
            <motion.path
              d="M 420 420 L 560 420"
              stroke="#2a2a3a"
              strokeWidth="2"
              fill="none"
              strokeDasharray="6 3"
            />
            <motion.path
              d="M 640 420 L 700 420"
              stroke="#2a2a3a"
              strokeWidth="2"
              fill="none"
              strokeDasharray="6 3"
            />

            <motion.text x="120" y="230" textAnchor="middle" className="text-xs fill-textMuted font-mono">
              QUEUE
            </motion.text>
            <motion.text x="380" y="230" textAnchor="middle" className="text-xs fill-textMuted font-mono">
              DISPATCH
            </motion.text>
            <motion.text x="660" y="100" textAnchor="middle" className="text-xs fill-textMuted font-mono">
              W1
            </motion.text>
            <motion.text x="660" y="200" textAnchor="middle" className="text-xs fill-textMuted font-mono">
              W2
            </motion.text>
            <motion.text x="660" y="300" textAnchor="middle" className="text-xs fill-textMuted font-mono">
              W3
            </motion.text>
            <motion.text x="660" y="400" textAnchor="middle" className="text-xs fill-textMuted font-mono">
              W4
            </motion.text>

            <motion.rect
              x="100"
              y="140"
              width="280"
              height="200"
              rx="4"
              className="stroke-border fill-surfaceAlt"
            />
            <motion.text x="240" y="130" textAnchor="middle" className="text-xs fill-textMuted font-mono">
              Priority Queue
            </motion.text>

            <motion.rect
              x="360"
              y="160"
              width="120"
              height="160"
              rx="4"
              className="stroke-border fill-surfaceAlt"
            />
            <motion.text x="420" y="150" textAnchor="middle" className="text-xs fill-textMuted font-mono">
              Dispatcher
            </motion.text>

            {WORKER_POSITIONS.map((worker) => {
              const workerData = workers.find((w) => w.id === worker.id)
              const stateClass = workerData
                ? WORKER_STATES[workerData.state] || WORKER_STATES.idle
                : WORKER_STATES.idle
              return (
                <motion.g key={worker.id}>
                  <motion.rect
                    x={worker.x - 50}
                    y={worker.y - 20}
                    width="100"
                    height="40"
                    rx="3"
                    fill={stateClass.color}
                    stroke={workerData?.state === 'running' ? '#00d4ff' : '#2a2a3a'}
                    strokeWidth={workerData?.state === 'running' ? '2' : '1'}
                  >
                    {workerData?.state === 'running' && (
                      <animate
                        attributeName="stroke"
                        values="#00d4ff;#4ade80;#00d4ff"
                        dur="1s"
                        repeatCount="indefinite"
                      />
                    )}
                  </motion.rect>
                  <motion.text
                    x={worker.x}
                    y={worker.y + 26}
                    textAnchor="middle"
                    className="text-xs fill-text font-mono font-bold"
                  >
                    W{worker.id}
                  </motion.text>
                  <motion.text
                    x={worker.x}
                    y={worker.y + 50}
                    textAnchor="middle"
                    className="text-xs fill-textMuted font-mono"
                  >
                    {workerData?.state || 'idle'}
                  </motion.text>
                  <motion.text
                    x={worker.x}
                    y={worker.y + 64}
                    textAnchor="middle"
                    className="text-xs fill-textDim font-mono"
                  >
                    {workerData ? `${workerData.processedCount} processed` : ''}
                  </motion.text>
                  {workerData?.currentJobId && (
                    <motion.text
                      x={worker.x}
                      y={worker.y + 80}
                      textAnchor="middle"
                      className="text-xs fill-accent font-mono"
                    >
                      #{workerData.currentJobId.slice(0, 6)}
                    </motion.text>
                  )}
                </motion.g>
              )
            })}

            {jobs.slice(-8).map((job, idx) => {
              const stageMap = {
                queued: { x: 180, y: 220 },
                dispatched: { x: 420, y: 240 },
                running: { x: 710, y: 120 },
                succeeded: { x: 860, y: 420 },
                failed: { x: 860, y: 480 },
                cancelled: { x: 860, y: 520 },
              }

              const targetStage =
                job.status === 'queued' ? 'queued' : job.status === 'running' ? 'running' : stageMap[job.status]

              let x = 180
              let y = 220 - idx * 24

              if (job.status === 'queued') {
                x = 180
                y = 200 - idx * 22
              } else if (job.status === 'running') {
                if (job.workerId) {
                  const wNum = parseInt(job.workerId)
                  x = 710
                  y = WORKER_POSITIONS[wNum - 1]?.y - 20 || 120
                } else {
                  x = 710
                  y = 120 + idx * 60
                }
              } else if (job.status === 'succeeded') {
                x = 180 + (idx % 4) * 30
                y = 520 + (idx % 4) * 2
              } else if (job.status === 'failed') {
                x = 200 + (idx % 4) * 30
                y = 520 + (idx % 4) * 2
              }

              const color = STATUS_COLORS[job.status] || '#7a7a8a'

              return (
                <motion.g key={job.id}>
                  <motion.rect
                    x={x - 60}
                    y={y - 8}
                    width="120"
                    height="16"
                    rx="2"
                    fill="none"
                    stroke={color}
                    strokeWidth="1"
                    opacity="0.3"
                  />
                  <motion.text
                    x={x - 55}
                    y={y + 5}
                    className="text-xs fill-textDim font-mono"
                  >
                    #{idx + 1}
                  </motion.text>
                  <motion.text
                    x={x + 50}
                    y={y + 5}
                    className="text-xs font-mono"
                  >
                    <tspan fill={color}>{job.type}</tspan>
                    <tspan x={x + 50} dy="14" fill="#7a7a8a">
                      {job.durationMs ? `${job.durationMs}ms` : 'queued'}
                    </tspan>
                  </motion.text>
                </motion.g>
              )
            })}

            <motion.rect
              x="840"
              y="380"
              width="140"
              height="160"
              rx="4"
              className="stroke-border fill-surfaceAlt"
            />
            <motion.text x="910" y="370" textAnchor="middle" className="text-xs fill-textMuted font-mono">
              Results
            </motion.text>

            <motion.text x="910" y="400" textAnchor="middle" className="text-xs font-mono">
              <tspan fill="#4ade80">✓</tspan> {jobs.filter((j) => j.status === 'succeeded').length} succeeded
            </motion.text>
            <motion.text x="910" y="420" textAnchor="middle" className="text-xs font-mono">
              <tspan fill="#f87171">✗</tspan> {jobs.filter((j) => j.status === 'failed').length} failed
            </motion.text>
            <motion.text x="910" y="440" textAnchor="middle" className="text-xs font-mono">
              <tspan fill="#fbbf24">○</tspan> {queueDepth} queued
            </motion.text>
            <motion.text x="910" y="460" textAnchor="middle" className="text-xs font-mono">
              <tspan fill="#60a5fa">◐</tspan> {activeWorkers} running
            </motion.text>
          </svg>

          <div className="mt-4 flex justify-center gap-2">
            <motion.button
              whileHover={{ scale: 1.02 }}
              whileTap={{ scale: 0.98 }}
              onClick={handleEnqueue}
              className="px-3 py-1.5 bg-accent text-bg font-mono text-xs font-bold rounded hover:bg-accentHover transition-colors shadow-lg shadow-accent/20"
            >
              Enqueue Job
            </motion.button>
            <motion.button
              whileHover={{ scale: 1.02 }}
              whileTap={{ scale: 0.98 }}
              onClick={() => window.dispatchEvent(new Event('resetSystem'))}
              className="px-3 py-1.5 border border-border font-mono text-xs rounded hover:border-accent transition-colors"
            >
              Clear
            </motion.button>
          </div>
        </div>
      </div>
    </section>
  )
}
