export interface JobView {
  id: string
  type: string
  label: string
  status: 'queued' | 'running' | 'succeeded' | 'failed' | 'cancelled'
  attempts: number
  error?: string
  output_url?: string
  size_bytes?: number
  duration_ms?: number
}

export interface WorkerState {
  id: number
  status: 'idle' | 'busy'
  job_id?: string
  job_type?: string
  since?: string
  processed_count: number
}

export interface UploadView {
  upload_id: string
  filename: string
  media_type: 'image' | 'video'
  status: 'queued' | 'processing' | 'completed' | 'completed_with_failures'
  jobs: JobView[]
  queue_depth: number
  workers?: WorkerState[]
  engine: string
}

export interface Metrics {
  queued: number
  running: number
  succeeded: number
  failed: number
  cancelled: number
  active_workers: number
  total_workers: number
  total_processed: number
  queue_depth: number
  queue_capacity: number
}

export async function fetchUpload(id: string): Promise<UploadView> {
  const res = await fetch(`/api/uploads/${id}`)
  if (!res.ok) throw new Error(`upload ${id}: ${res.status}`)
  return res.json()
}

export async function fetchMetrics(): Promise<Metrics> {
  const res = await fetch('/api/metrics')
  if (!res.ok) throw new Error(`metrics: ${res.status}`)
  return res.json()
}

export async function fetchUploads(): Promise<UploadView[]> {
  const res = await fetch('/api/uploads?limit=12')
  if (!res.ok) throw new Error(`uploads: ${res.status}`)
  const data = await res.json()
  return data.uploads ?? []
}

// Uses XHR rather than fetch so upload progress is real, not simulated.
export function uploadFile(
  file: File,
  onProgress: (percent: number) => void
): Promise<UploadView> {
  return new Promise((resolve, reject) => {
    const form = new FormData()
    form.append('file', file)

    const xhr = new XMLHttpRequest()
    xhr.open('POST', '/api/uploads')
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) onProgress(Math.round((e.loaded / e.total) * 100))
    }
    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve(JSON.parse(xhr.responseText))
      } else {
        try {
          reject(new Error(JSON.parse(xhr.responseText).error ?? `upload failed (${xhr.status})`))
        } catch {
          reject(new Error(`upload failed (${xhr.status})`))
        }
      }
    }
    xhr.onerror = () => reject(new Error('network error during upload'))
    xhr.send(form)
  })
}

export const formatBytes = (bytes?: number): string => {
  if (!bytes) return '—'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
}
