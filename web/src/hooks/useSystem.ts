import { useCallback, useEffect, useRef, useState } from 'react'
import { Job, JobStatus } from '../types'

const generateJobId = (): string => {
  return Math.random().toString(36).substring(2, 10).toUpperCase()
}

const simulateJobDuration = (type: string): number => {
  const durations: Record<string, [number, number]> = {
    echo: [100, 200],
    sleep: [800, 1500],
    email: [200, 500],
    'image-processing': [1500, 3000],
    'data-sync': [2000, 4000],
  }
  const range = durations[type] || [300, 600]
  return range[0] + Math.random() * (range[1] - range[0])
}

export type WorkerState = 'idle' | 'running' | 'stopping'

export interface WorkerInfo {
  id: number
  state: WorkerState
  currentJobId: string | null
  processedCount: number
}

export interface SystemState {
  jobs: Job[]
  workers: WorkerInfo[]
  queueDepth: number
  activeWorkers: number
  totalProcessed: number
  throughput: number
}

export interface SystemActions {
  enqueueJob: (type: string, priority: number, payload?: Record<string, unknown>) => string
  cancelJob: (jobId: string) => void
  setWorkerCount: (count: number) => void
  reset: () => void
}

export const useSystem = (initialWorkerCount: number) => {
  const [jobs, setJobs] = useState<Job[]>([])
  const [workers, setWorkers] = useState<WorkerInfo[]>(() =>
    Array.from({ length: initialWorkerCount }, (_, i) => ({
      id: i + 1,
      state: 'idle' as WorkerState,
      currentJobId: null,
      processedCount: 0,
    }))
  )
  const [workerCount, setWorkerCountState] = useState(initialWorkerCount)
  const activeTimersRef = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map())

  const updateJobStatus = useCallback((jobId: string, status: JobStatus) => {
    setJobs((prev) =>
      prev.map((j) =>
        j.id === jobId
          ? { ...j, status, updatedAt: new Date().toISOString() }
          : j
      )
    )
  }, [])

  const enqueueJob = useCallback(
    (type: string, priority: number, payload: Record<string, unknown> = {}) => {
      const id = generateJobId()
      const now = new Date()
      const job: Job = {
        id,
        type,
        payload,
        priority,
        status: 'queued',
        attempts: 0,
        maxAttempts: 3,
        createdAt: now.toISOString(),
        updatedAt: now.toISOString(),
        runAt: now.toISOString(),
      }

      setJobs((prev) => [...prev.slice(-39), job])

      setTimeout(() => {
        dispatchJob(id)
      }, 100)

      return id
    },
    []
  )

  const dispatchJob = useCallback((jobId: string) => {
    setJobs((prev) => {
      const queuedJobs = prev.filter((j) => j.status === 'queued')
      if (queuedJobs.length === 0) return prev

      const jobToProcess = queuedJobs.find((j) => j.id === jobId)
      if (!jobToProcess) return prev

      updateJobStatus(jobToProcess.id, 'running')

      const worker = workers.find((w) => w.state === 'idle')
      if (worker) {
        setWorkers((prevWorkers) =>
          prevWorkers.map((w) =>
            w.id === worker.id
              ? { ...w, state: 'running' as WorkerState, currentJobId: jobToProcess.id }
              : w
          )
        )

        const duration = simulateJobDuration(jobToProcess.type)

        const timer = setTimeout(() => {
          const success = Math.random() > 0.1
          setJobs((prevJobs) =>
            prevJobs.map((j) =>
              j.id === jobToProcess.id
                ? {
                    ...j,
                    status: (success ? 'succeeded' : 'failed') as JobStatus,
                    completedAt: new Date().toISOString(),
                    updatedAt: new Date().toISOString(),
                    durationMs: Math.floor(duration),
                    workerId: String(worker.id),
                  }
                : j
            )
          )
          setWorkers((prevWorkers) =>
            prevWorkers.map((w) =>
              w.id === worker.id
                ? {
                    ...w,
                    state: 'idle' as WorkerState,
                    currentJobId: null,
                    processedCount: w.processedCount + 1,
                  }
                : w
            )
          )
          activeTimersRef.current.delete(jobToProcess.id)
        }, duration)

        activeTimersRef.current.set(jobToProcess.id, timer)
      }

      return prev.map((j) =>
        j.id === jobToProcess.id
          ? { ...j, status: 'running' as JobStatus, updatedAt: new Date().toISOString() }
          : j
      )
    })
  }, [workers, updateJobStatus])

  useEffect(() => {
    const interval = setInterval(() => {
      setJobs((prev) => {
        const newJobs = [...prev]
        const idleWorker = workers.find((w) => w.state === 'idle')
        if (idleWorker && newJobs.some((j) => j.status === 'queued')) {
          const queuedJob = newJobs.find((j) => j.status === 'queued')
          if (queuedJob) {
            setTimeout(() => dispatchJob(queuedJob.id), 50)
          }
        }

        return newJobs
      })
    }, 150)

    return () => clearInterval(interval)
  }, [workers, dispatchJob])

  const cancelJob = useCallback((jobId: string) => {
    const timer = activeTimersRef.current.get(jobId)
    if (timer) {
      clearTimeout(timer)
      activeTimersRef.current.delete(jobId)
    }

    setJobs((prev) =>
      prev.map((j) =>
        j.id === jobId
          ? { ...j, status: 'cancelled' as JobStatus, completedAt: new Date().toISOString(), updatedAt: new Date().toISOString() }
          : j
      )
    )

    const job = jobs.find((j) => j.id === jobId)
    if (job && job.workerId) {
      setWorkers((prev) =>
        prev.map((w) =>
          w.id === parseInt(job.workerId!)
            ? { ...w, state: 'idle' as WorkerState, currentJobId: null }
            : w
        )
      )
    }
  }, [jobs])

  const setWorkerCount = useCallback((count: number) => {
    setWorkerCountState(count)
    setWorkers(
      Array.from({ length: count }, (_, i) => ({
        id: i + 1,
        state: 'idle' as WorkerState,
        currentJobId: null,
        processedCount: 0,
      }))
    )
  }, [])

  const reset = useCallback(() => {
    setJobs([])
    setWorkers(
      Array.from({ length: workerCount }, (_, i) => ({
        id: i + 1,
        state: 'idle' as WorkerState,
        currentJobId: null,
        processedCount: 0,
      }))
    )
    activeTimersRef.current.forEach((timer) => clearTimeout(timer))
    activeTimersRef.current.clear()
  }, [workerCount])

  return {
    jobs,
    workers,
    queueDepth: jobs.filter((j) => j.status === 'queued').length,
    activeWorkers: workers.filter((w) => w.state === 'running').length,
    totalProcessed: workers.reduce((sum, w) => sum + w.processedCount, 0),
    throughput: jobs.filter((j) => j.status === 'succeeded').length,
    enqueueJob,
    cancelJob,
    setWorkerCount,
    reset,
  }
}
