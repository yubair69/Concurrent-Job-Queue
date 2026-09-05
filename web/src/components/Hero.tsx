import React, { useEffect, useState } from 'react'
import { motion, useAnimation } from 'framer-motion'

const PIPELINE_STAGES = [
  { id: 'queue', label: 'Queue', color: 'text-yellow-400', bg: 'bg-yellow-900/10', border: 'border-yellow-500/30' },
  { id: 'dispatch', label: 'Dispatch', color: 'text-cyan-400', bg: 'bg-cyan-900/10', border: 'border-cyan-500/30' },
  { id: 'worker', label: 'Worker Pool', color: 'text-blue-400', bg: 'bg-blue-900/10', border: 'border-blue-500/30' },
  { id: 'running', label: 'Running', color: 'text-purple-400', bg: 'bg-purple-900/10', border: 'border-purple-500/30' },
  { id: 'result', label: 'Result', color: 'text-green-400', bg: 'bg-green-900/10', border: 'border-green-500/30' },
]

interface Job {
  id: number
  stage: number
  stageLabel: string
  statusColor: string
  duration: string
}

export const Hero: React.FC = () => {
  const controls = useAnimation()
  const [jobs, setJobs] = useState<Job[]>([])
  const [activeStage, setActiveStage] = useState(0)

  useEffect(() => {
    const interval = setInterval(() => {
      setJobs((prev) => {
        const newJobs: Job[] = prev.map((job) => {
          if (job.stage < PIPELINE_STAGES.length - 1) {
            return {
              ...job,
              stage: job.stage + 1,
              stageLabel: PIPELINE_STAGES[job.stage + 1].label,
              statusColor: PIPELINE_STAGES[job.stage + 1].color,
              duration: `${50 + Math.floor(Math.random() * 200)}ms`,
            }
          }
          return job
        }).filter((job) => job.stage < PIPELINE_STAGES.length)

        if (newJobs.length < 6) {
          const newJob: Job = {
            id: Date.now() + Math.random(),
            stage: 0,
            stageLabel: PIPELINE_STAGES[0].label,
            statusColor: PIPELINE_STAGES[0].color,
            duration: 'queued',
          }
          return [...newJobs, newJob].slice(-8)
        }
        return newJobs
      })
    }, 1500)

    const pulseInterval = setInterval(() => {
      setActiveStage((prev) => (prev + 1) % PIPELINE_STAGES.length)
    }, 500)

    return () => {
      clearInterval(interval)
      clearInterval(pulseInterval)
    }
  }, [])

  return (
    <section className="relative h-screen flex items-center overflow-hidden pt-14">
      <div className="absolute inset-0 pointer-events-none">
        <div className="absolute top-20 left-10 w-px h-40 bg-border" />
        <div className="absolute top-20 right-10 w-px h-40 bg-border" />
        <div className="absolute bottom-20 left-1/2 -translate-x-1/2 w-64 h-px bg-border" />
      </div>

      <div className="max-w-7xl mx-auto px-6 relative z-10 w-full">
        <div className="grid lg:grid-cols-2 gap-12 items-center">
          <motion.div
            initial={{ x: -40, opacity: 0 }}
            animate={{ x: 0, opacity: 1 }}
            transition={{ duration: 0.5, ease: 'easeOut' }}
            className="space-y-6"
          >
            <div className="space-y-2">
              <motion.h1
                className="text-5xl md:text-6xl font-light leading-tight"
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.1 }}
              >
                Run work
                <br />
                <span className="text-accent font-medium">concurrently.</span>
              </motion.h1>
              <motion.p
                className="text-lg text-textMuted max-w-md"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                transition={{ delay: 0.2 }}
              >
                GoTask is a concurrent background job engine for Go. Jobs flow
                through a priority queue, get dispatched to a worker pool, and
                execute with retries, cancellation, and full observability.
              </motion.p>
            </div>

            <motion.div
              className="flex gap-3 pt-2"
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.3 }}
            >
              <motion.button
                whileHover={{ scale: 1.02 }}
                whileTap={{ scale: 0.98 }}
                className="px-5 py-2.5 bg-accent text-bg font-mono font-bold text-sm rounded hover:bg-accentHover transition-colors shadow-lg shadow-accent/20"
              >
                Live Demo ↓
              </motion.button>
              <motion.button
                whileHover={{ scale: 1.02 }}
                whileTap={{ scale: 0.98 }}
                className="px-5 py-2.5 border border-border font-mono text-sm rounded hover:border-accent transition-colors"
              >
                GitHub →
              </motion.button>
            </motion.div>

            <motion.div
              className="pt-4 space-y-2"
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.4 }}
            >
              <div className="text-xs font-mono text-textDim">
                Priority Queue → Goroutine Workers → SQLite → Exponential Backoff
              </div>
              <div className="flex gap-6 text-xs font-mono text-textMuted">
                <span className="flex items-center gap-1">
                  <div className="w-1.5 h-1.5 bg-green rounded-full" />
                  Priority Scheduling
                </span>
                <span className="flex items-center gap-1">
                  <div className="w-1.5 h-1.5 bg-blue rounded-full" />
                  Context Cancellation
                </span>
                <span className="flex items-center gap-1">
                  <div className="w-1.5 h-1.5 bg-yellow rounded-full" />
                  Retry with Backoff
                </span>
                <span className="flex items-center gap-1">
                  <div className="w-1.5 h-1.5 bg-cyan rounded-full" />
                  SQLite Persistence
                </span>
              </div>
            </motion.div>
          </motion.div>

          <motion.div
            initial={{ x: 40, opacity: 0 }}
            animate={{ x: 0, opacity: 1 }}
            transition={{ duration: 0.5, ease: 'easeOut', delay: 0.1 }}
          >
            <div className="bg-surfaceAlt border border-border rounded-xl p-6 font-mono text-xs">
              <div className="flex items-center gap-2 mb-4 pb-2 border-b border-border">
                <div className="flex gap-1">
                  <div className="w-2 h-2 rounded-full bg-red" />
                  <div className="w-2 h-2 rounded-full bg-yellow" />
                  <div className="w-2 h-2 rounded-full bg-green" />
                </div>
                <span className="text-textMuted">Job Pipeline</span>
              </div>

              <div className="space-y-3">
                {PIPELINE_STAGES.map((stage, i) => (
                  <motion.div
                    key={stage.id}
                    className={`flex items-center gap-3 py-2 ${stage.bg} ${
                      activeStage === i ? stage.border : 'border-transparent'
                    } border rounded-md px-3 transition-all`}
                  >
                    <motion.div
                      className={`w-2 h-2 rounded-full ${stage.color.replace('text-', 'bg-')}`}
                      animate={{
                        scale: activeStage === i ? 1.5 : 1,
                        opacity: activeStage === i ? 1 : 0.5,
                      }}
                    />
                    <span className={`font-medium ${stage.color}`}>{stage.label}</span>
                    <div className="ml-auto text-textDim">
                      {i === 4 ? `${jobs.filter((j) => j.stage === i).length} completed` : ''}
                    </div>
                  </motion.div>
                ))}

                <div className="mt-4 space-y-1.5">
                  <div className="text-textMuted text-xs">Active Jobs</div>
                  {jobs.slice(-5).map((job) => (
                    <motion.div
                      key={job.id}
                      className="flex items-center justify-between py-1.5 border-l-2 border-border pl-2"
                      initial={{ opacity: 0, x: 20 }}
                      animate={{ opacity: 1, x: 0 }}
                    >
                      <span className="text-textMuted font-mono">#{Math.round(job.id)}</span>
                      <span className={`font-mono ${job.statusColor}`}>{job.stageLabel}</span>
                      <span className="text-textDim">{job.duration}</span>
                    </motion.div>
                  ))}
                </div>
              </div>
            </div>
          </motion.div>
        </div>
      </div>
    </section>
  )
}
