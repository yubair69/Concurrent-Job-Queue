import React, { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Job, JobStatus, JOB_TYPES } from '@/types'
import { useSystem } from '@/hooks/useSystem'

const STATUS_CONFIG: Record<JobStatus, { color: string; label: string; icon: string }> = {
  queued: { color: 'text-yellow-400', label: 'Queued', icon: '○' },
  running: { color: 'text-blue-400', label: 'Running', icon: '◐' },
  succeeded: { color: 'text-green-400', label: 'Succeeded', icon: '✓' },
  failed: { color: 'text-red-400', label: 'Failed', icon: '✗' },
  cancelled: { color: 'text-textDim', label: 'Cancelled', icon: '⊘' },
}

const JOB_TYPE_DESCRIPTIONS: Record<string, string> = {
  echo: 'Echo back the payload',
  sleep: 'Sleep for a configurable duration (simulates long work)',
  email: 'Simulate sending an email notification',
  'image-processing': 'Process an uploaded image (CPU-intensive)',
  'data-sync': 'Synchronize data across systems (I/O-bound)',
}

export const JobExplorer: React.FC = () => {
  const {
    jobs,
    workers,
    enqueueJob,
    cancelJob,
    queueDepth,
    activeWorkers,
    totalProcessed,
  } = useSystem(4)
  const [selectedType, setSelectedType] = useState('echo')
  const [priority, setPriority] = useState(3)
  const [payload, setPayload] = useState('{"message": "hello world"}')
  const [copiedId, setCopiedId] = useState<string | null>(null)

  const handleEnqueue = () => {
    try {
      const parsedPayload = payload ? JSON.parse(payload) : {}
      enqueueJob(selectedType, priority, parsedPayload)
      setPayload('{"message": "hello world"}')
    } catch (e) {
      console.error('Invalid JSON payload', e)
    }
  }

  const handleCancel = (jobId: string) => {
    cancelJob(jobId)
  }

  const handleCopyId = (jobId: string) => {
    navigator.clipboard.writeText(jobId)
    setCopiedId(jobId)
    setTimeout(() => setCopiedId(null), 2000)
  }

  const runningJobs = jobs.filter((j) => j.status === 'running')
  const succeededJobs = jobs.filter((j) => j.status === 'succeeded')
  const failedJobs = jobs.filter((j) => j.status === 'failed')

  return (
    <section id="jobs" className="py-20 bg-bg">
      <div className="max-w-7xl mx-auto px-6">
        <motion.div
          initial={{ y: 30, opacity: 0 }}
          whileInView={{ y: 0, opacity: 1 }}
          viewport={{ once: true }}
          className="mb-12"
        >
          <h2 className="text-3xl font-bold mb-2 font-mono">Job Explorer</h2>
          <p className="text-textMuted max-w-2xl">
            Submit jobs with custom payloads and watch them flow through the pipeline.
            Jobs are enqueued, dispatched to workers, executed, and their status
            persists to SQLite with automatic retry on failure.
          </p>
        </motion.div>

        <div className="grid lg:grid-cols-5 gap-6">
          <motion.div
            initial={{ y: 20, opacity: 0 }}
            whileInView={{ y: 0, opacity: 1 }}
            viewport={{ once: true }}
            className="lg:col-span-1 bg-surfaceAlt border border-border rounded-xl p-6 space-y-4"
          >
            <h3 className="text-sm font-mono text-textMuted">Submit Job</h3>

            <div>
              <label className="text-xs font-mono text-textMuted">Job Type</label>
              <select
                value={selectedType}
                onChange={(e) => setSelectedType(e.target.value)}
                className="w-full mt-1 px-2 py-1.5 bg-bg border border-border rounded text-sm font-mono focus:outline-none focus:border-accent"
              >
                {Object.entries(JOB_TYPES).map(([key, config]) => (
                  <option key={key} value={key}>
                    {config.name}
                  </option>
                ))}
              </select>
              <p className="mt-1 text-xs text-textDim">
                {JOB_TYPE_DESCRIPTIONS[selectedType]}
              </p>
            </div>

            <div>
              <label className="text-xs font-mono text-textMuted">Priority</label>
              <select
                value={priority}
                onChange={(e) => setPriority(parseInt(e.target.value))}
                className="w-full mt-1 px-2 py-1.5 bg-bg border border-border rounded text-sm font-mono focus:outline-none focus:border-accent"
              >
                <option value={5}>5 - Critical</option>
                <option value={3}>3 - Normal</option>
                <option value={1}>1 - Low</option>
              </select>
            </div>

            <div>
              <label className="text-xs font-mono text-textMuted">Payload (JSON)</label>
              <textarea
                value={payload}
                onChange={(e) => setPayload(e.target.value)}
                className="w-full mt-1 px-2 py-1.5 bg-bg border border-border rounded text-xs font-mono focus:outline-none focus:border-accent resize-none"
                rows={4}
              />
            </div>

            <motion.button
              whileHover={{ scale: 1.01 }}
              whileTap={{ scale: 0.99 }}
              onClick={handleEnqueue}
              className="w-full px-3 py-2 bg-accent text-bg font-mono font-bold text-xs rounded hover:bg-accentHover transition-colors shadow-lg shadow-accent/20"
            >
              Enqueue Job
            </motion.button>
          </motion.div>

          <div className="lg:col-span-4 space-y-4">
            <div className="grid grid-cols-4 gap-3 mb-4">
              <div className="bg-surfaceAlt border border-border rounded-lg p-3">
                <div className="text-xs text-textMuted font-mono">Queue Depth</div>
                <div className="text-2xl font-bold text-yellow-400">{queueDepth}</div>
              </div>
              <div className="bg-surfaceAlt border border-border rounded-lg p-3">
                <div className="text-xs text-textMuted font-mono">Running</div>
                <div className="text-2xl font-bold text-blue-400">{activeWorkers}</div>
              </div>
              <div className="bg-surfaceAlt border border-border rounded-lg p-3">
                <div className="text-xs text-textMuted font-mono">Succeeded</div>
                <div className="text-2xl font-bold text-green-400">{succeededJobs.length}</div>
              </div>
              <div className="bg-surfaceAlt border border-border rounded-lg p-3">
                <div className="text-xs text-textMuted font-mono">Failed</div>
                <div className="text-2xl font-bold text-red-400">{failedJobs.length}</div>
              </div>
            </div>

            <div className="bg-surfaceAlt border border-border rounded-xl overflow-hidden">
              <table className="w-full text-sm font-mono">
                <thead>
                  <tr className="border-b border-border">
                    <th className="text-left py-2 px-3 text-xs text-textMuted">ID</th>
                    <th className="text-left py-2 px-3 text-xs text-textMuted">Type</th>
                    <th className="text-left py-2 px-3 text-xs text-textMuted">Priority</th>
                    <th className="text-left py-2 px-3 text-xs text-textMuted">Status</th>
                    <th className="text-left py-2 px-3 text-xs text-textMuted">Worker</th>
                    <th className="text-left py-2 px-3 text-xs text-textMuted">Duration</th>
                    <th className="text-left py-2 px-3 text-xs text-textMuted">Attempts</th>
                    <th className="text-left py-2 px-3 text-xs text-textMuted">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  <AnimatePresence>
                    {jobs.slice().reverse().map((job) => (
                      <motion.tr
                        key={job.id}
                        initial={{ opacity: 0, y: 10 }}
                        animate={{ opacity: 1, y: 0 }}
                        exit={{ opacity: 0, y: -10 }}
                        className="border-b border-border/30 hover:bg-bgAlt/20"
                      >
                        <td className="py-2 px-3">
                          <button
                            onClick={() => handleCopyId(job.id)}
                            className="text-textDim hover:text-text font-mono transition-colors"
                          >
                            #{job.id.slice(0, 6)}
                          </button>
                          {copiedId === job.id && (
                            <span className="ml-1 text-xs text-green">copied</span>
                          )}
                        </td>
                        <td className="py-2 px-3">
                          <span className="text-cyan">{job.type}</span>
                        </td>
                        <td className="py-2 px-3">
                          <span className={job.priority >= 3 ? 'text-accent' : 'text-textDim'}>
                            {job.priority}
                          </span>
                        </td>
                        <td className="py-2 px-3">
                          <span className={`status-pill ${STATUS_CONFIG[job.status].color.replace('text-', 'bg-')}/20 border ${STATUS_CONFIG[job.status].color.replace('text-', 'border-')}/30`}>
                            {STATUS_CONFIG[job.status].icon} {STATUS_CONFIG[job.status].label}
                          </span>
                        </td>
                        <td className="py-2 px-3 text-textDim">
                          {job.workerId ? `Worker ${job.workerId}` : '-'}
                        </td>
                        <td className="py-2 px-3 text-textDim">
                          {job.durationMs ? `${job.durationMs}ms` : '-'}
                        </td>
                        <td className="py-2 px-3 text-textDim">
                          {job.attempts}/{job.maxAttempts}
                        </td>
                        <td className="py-2 px-3">
                          {job.status === 'queued' && (
                            <motion.button
                              whileHover={{ scale: 1.1 }}
                              whileTap={{ scale: 0.9 }}
                              onClick={() => handleCancel(job.id)}
                              className="px-2 py-1 text-xs bg-red/20 border border-red/30 rounded text-red hover:bg-red/30 transition-colors"
                            >
                              Cancel
                            </motion.button>
                          )}
                        </td>
                      </motion.tr>
                    ))}
                  </AnimatePresence>
                </tbody>
              </table>
              {jobs.length === 0 && (
                <div className="text-center py-12 text-textDim font-mono text-sm">
                  No jobs yet. Submit one using the form.
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
