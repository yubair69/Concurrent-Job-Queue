import React, { useEffect, useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'

const WORKER_COUNTS = [1, 2, 4, 8] as const
type WorkerCount = typeof WORKER_COUNTS[number]

interface Job {
  id: number
  status: 'queued' | 'running' | 'completed' | 'failed'
  workerId: number | null
  duration: number
  startTime: number | null
}

const getDurationForType = (): number => {
  return 300 + Math.random() * 400
}

export const ConcurrencyDemo: React.FC = () => {
  const [workerCount, setWorkerCount] = useState<WorkerCount>(4)
  const [jobs, setJobs] = useState<Job[]>([])
  const [completed, setCompleted] = useState(0)
  const [isRunning, setIsRunning] = useState(false)

  useEffect(() => {
    let spawnInterval: ReturnType<typeof setInterval> | null = null
    let processInterval: ReturnType<typeof setInterval> | null = null

    if (isRunning) {
      let jobCounter = 0

      spawnInterval = setInterval(() => {
        if (jobs.filter((j) => j.status === 'queued' || j.status === 'running').length < 20) {
          const newJob: Job = {
            id: ++jobCounter,
            status: 'queued',
            workerId: null,
            duration: getDurationForType(),
            startTime: null,
          }
          setJobs((prev) => [...prev, newJob])
        }
      }, 150)

      processInterval = setInterval(() => {
        setJobs((prevJobs) => {
          const activeJobs = prevJobs.filter((j) => j.status === 'queued' || j.status === 'running')
          const queuedJobs = activeJobs.filter((j) => j.status === 'queued')

          if (queuedJobs.length === 0) return prevJobs

          const workersBusy = prevJobs
            .filter((j) => j.status === 'running')
            .map((j) => j.workerId)
            .filter((id): id is number => id !== null)

          const availableWorkerCount = Math.max(0, workerCount - workersBusy.length)

          if (availableWorkerCount === 0) return prevJobs

          const jobsToStart = queuedJobs.slice(0, availableWorkerCount)
          const updatedJobs = [...prevJobs]

          jobsToStart.forEach((job) => {
            const jobIndex = updatedJobs.findIndex((j) => j.id === job.id)
            if (jobIndex !== -1) {
              const workerId = (() => {
                for (let w = 1; w <= workerCount; w++) {
                  if (!workersBusy.includes(w)) {
                    workersBusy.push(w)
                    return w
                  }
                }
                return 1
              })()

              updatedJobs[jobIndex] = {
                ...job,
                status: 'running',
                workerId,
                startTime: Date.now(),
              }
            }
          })

          return updatedJobs
        })

              setJobs((prevJobs) => {
          const now = Date.now()
          let newCompleted = 0

          const updatedJobs = prevJobs.map((job) => {
            if (job.status === 'running' && job.startTime) {
              if (now - job.startTime >= job.duration) {
                newCompleted++
                const isSuccess = Math.random() > 0.05
                return {
                  ...job,
                  status: (isSuccess ? 'completed' : 'failed') as 'completed' | 'failed',
                  workerId: null,
                }
              }
            }
            return job
          })

          if (newCompleted > 0) {
            setCompleted((prev) => prev + newCompleted)
          }

          return updatedJobs.filter((j) => !(j.status === 'completed' || j.status === 'failed' && now - (j.startTime || 0) > 5000))
        })
      }, 50)
    }

    return () => {
      if (spawnInterval) clearInterval(spawnInterval)
      if (processInterval) clearInterval(processInterval)
    }
  }, [isRunning, workerCount, jobs])

  const handleStart = () => {
    if (!isRunning) {
      setJobs([])
      setCompleted(0)
      setIsRunning(true)
    } else {
      setIsRunning(false)
    }
  }

  const handleWorkerCountChange = (count: WorkerCount) => {
    setWorkerCount(count)
    setJobs([])
    setCompleted(0)
  }

  return (
    <section id="concurrency" className="py-20 bg-bgAlt">
      <div className="max-w-6xl mx-auto px-6">
        <motion.div
          initial={{ y: 30, opacity: 0 }}
          whileInView={{ y: 0, opacity: 1 }}
          viewport={{ once: true }}
          className="text-center mb-12"
        >
          <h2 className="text-3xl font-bold mb-4 font-mono">Concurrency Demonstration</h2>
          <p className="text-textMuted max-w-2xl mx-auto">
            Observe how GoTask processes jobs concurrently across a goroutine worker pool.
            Compare throughput at different worker counts — 1 worker processes sequentially,
            while 8 workers process jobs in parallel.
          </p>
        </motion.div>

        <div className="mb-6 flex items-center justify-center gap-4">
          <span className="text-sm text-textMuted font-mono">Workers:</span>
          <div className="flex gap-1.5">
            {WORKER_COUNTS.map((count) => (
              <motion.button
                key={count}
                whileHover={{ scale: 1.05 }}
                whileTap={{ scale: 0.95 }}
                onClick={() => handleWorkerCountChange(count)}
                className={`px-4 py-1.5 font-mono text-sm rounded transition-all ${
                  workerCount === count
                    ? 'bg-accent text-bg shadow-lg shadow-accent/20'
                    : 'border border-border text-textMuted hover:border-accent'
                }}`}
              >
                {count}
              </motion.button>
            ))}
          </div>

          <motion.button
            whileHover={{ scale: 1.02 }}
            whileTap={{ scale: 0.98 }}
            onClick={handleStart}
            className={`ml-4 px-4 py-1.5 font-mono text-xs font-bold rounded transition-all ${
              isRunning
                ? 'bg-red/20 border border-red text-red'
                : 'bg-green/20 border border-green text-green'
            }}`}
          >
            {isRunning ? 'Stop' : 'Start'}
          </motion.button>
        </div>

        <div className="bg-surfaceAlt border border-border rounded-xl p-6 font-mono">
          <div className="flex justify-between mb-4 text-sm">
            <div>
              <span className="text-textMuted">Workers:</span>
              <span className="text-blue-400 font-bold ml-2">{workerCount}</span>
            </div>
            <div>
              <span className="text-textMuted">Completed:</span>
              <span className="text-green-400 font-bold ml-2">{completed}</span>
            </div>
            <div>
              <span className="text-textMuted">Queue:</span>
              <span className="text-yellow-400 font-bold ml-2">
                {jobs.filter((j) => j.status === 'queued').length}
              </span>
            </div>
            <div>
              <span className="text-textMuted">Active:</span>
              <span className="text-purple-400 font-bold ml-2">
                {jobs.filter((j) => j.status === 'running').length}
              </span>
            </div>
          </div>

          <div className="grid grid-cols-2 sm:grid-cols-4 md:grid-cols-8 gap-2">
            {Array.from({ length: workerCount }).map((_, i) => {
              const workerId = i + 1
              const runningJob = jobs.find((j) => j.status === 'running' && j.workerId === workerId)
              const workerCompleted = jobs.filter(
                (j) => j.workerId === workerId && (j.status === 'completed' || j.status === 'failed')
              ).length

              return (
                <div key={workerId} className="border border-border rounded-lg p-2.5 text-center">
                  <div className="text-xs text-textMuted mb-1">W{workerId}</div>
                  <div
                    className={`w-2 h-2 mx-auto rounded-full mb-1 ${
                      runningJob ? 'bg-blue animate-pulse' : 'bg-textDim'
                    }`}
                  />
                  <div className={`text-xs font-mono ${runningJob ? 'text-blue' : 'text-textMuted'}`}>
                    {runningJob ? 'running' : 'idle'}
                  </div>
                  <div className="text-xs text-textDim mt-1">{workerCompleted} done</div>
                </div>
              )
            })}
          </div>
        </div>

        <div className="mt-6 bg-surfaceAlt border border-border rounded-xl p-6">
          <h3 className="text-lg font-bold font-mono mb-3">Job Timeline</h3>
          <div className="relative">
            <div className="absolute left-1/2 -translate-x-1/2 h-full w-px bg-border" />
            <div className="h-2" />

            <div className="space-y-2">
              <AnimatePresence>
                {jobs.slice(-20).map((job) => (
                  <motion.div
                    key={job.id}
                    initial={{ opacity: 0, x: -20 }}
                    animate={{ opacity: 1, x: 0 }}
                    exit={{ opacity: 0, x: 20 }}
                    className="relative flex items-center"
                  >
                    <div
                      className={`absolute left-1/2 -translate-x-1/2 w-px h-4 ${
                        job.status === 'running'
                          ? 'bg-blue animate-pulse'
                          : job.status === 'completed'
                          ? 'bg-green'
                          : job.status === 'failed'
                          ? 'bg-red'
                          : 'bg-yellow'
                      }`}
                    />
                    <div
                      className={`ml-auto mr-4 px-3 py-1.5 rounded text-xs ${
                        job.status === 'running'
                          ? 'bg-blue-900/20 border border-blue-500/30 text-blue-400'
                          : job.status === 'completed'
                          ? 'bg-green-900/20 border border-green-500/30 text-green-400'
                          : job.status === 'failed'
                          ? 'bg-red-900/20 border border-red-500/30 text-red-400'
                          : 'bg-yellow-900/20 border border-yellow-500/30 text-yellow-400'
                      }`}
                    >
                      #{job.id}
                    </div>
                    <div
                      className={`px-2 py-0.5 rounded text-xs ${
                        job.status === 'running'
                          ? 'bg-blue-900/30 text-blue-400'
                          : job.status === 'completed'
                          ? 'bg-green-900/30 text-green-400'
                          : job.status === 'failed'
                          ? 'bg-red-900/30 text-red-400'
                          : 'bg-yellow-900/30 text-yellow-400'
                      }`}
                    >
                      {job.status}
                    </div>
                    {job.workerId && (
                      <div className="ml-2 text-xs text-textMuted">→ W{job.workerId}</div>
                    )}
                  </motion.div>
                ))}
              </AnimatePresence>
            </div>

            {jobs.length === 0 && !isRunning && (
              <div className="text-center py-12 text-textDim font-mono text-sm">
                Click Start to begin dispatching jobs
              </div>
            )}
          </div>
        </div>
      </div>
    </section>
  )
}
