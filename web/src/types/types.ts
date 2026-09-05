export type JobStatus = 'queued' | 'running' | 'succeeded' | 'failed' | 'cancelled'

export interface Job {
  id: string
  type: string
  payload: Record<string, unknown>
  priority: number
  status: JobStatus
  attempts: number
  maxAttempts: number
  workerId?: string
  createdAt: string
  startedAt?: string
  completedAt?: string
  updatedAt: string
  runAt: string
  durationMs?: number
}

export interface Worker {
  id: number
  state: 'idle' | 'running' | 'stopping'
  currentJobId?: string
  processedCount: number
}

export interface QueueStats {
  depth: number
  enqueued: number
  dequeued: number
  capacity: number
}

export interface PoolStats {
  activeWorkers: number
  totalWorkers: number
  totalProcessed: number
}

export interface Metrics {
  queued: number
  running: number
  succeeded: number
  failed: number
  cancelled: number
  activeWorkers: number
  totalWorkers: number
  totalProcessed: number
  totalAttempted: number
  queueDepth: number
  queueEnqueued: number
  queueDequeued: number
  queueCapacity: number
  throughput: number
}

export interface JobTypeConfig {
  name: string
  description: string
}

export const JOB_TYPES: Record<string, JobTypeConfig> = {
  echo: { name: 'Echo', description: 'Echo back the payload' },
  sleep: { name: 'Sleep', description: 'Sleep for a configurable duration' },
  email: { name: 'Email', description: 'Simulate sending an email notification' },
  'image-processing': { name: 'Image Processing', description: 'Process an uploaded image' },
  'data-sync': { name: 'Data Sync', description: 'Synchronize data across systems' },
}
