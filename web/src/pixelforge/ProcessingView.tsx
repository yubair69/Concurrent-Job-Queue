import React from 'react'
import { UploadView, JobView, WorkerState, formatBytes } from './api'

const STATUS_CLASS: Record<string, string> = {
  queued: 'text-yellow',
  running: 'text-accent',
  succeeded: 'text-green',
  failed: 'text-red',
  cancelled: 'text-textDim',
}

const bar = (job: JobView): number => {
  if (job.status === 'succeeded') return 100
  if (job.status === 'failed' || job.status === 'cancelled') return 100
  if (job.status === 'running') return 60
  return 8
}

const WorkerRow: React.FC<{ worker: WorkerState; jobs: JobView[] }> = ({ worker, jobs }) => {
  const job = jobs.find((j) => j.id === worker.job_id)
  const busy = worker.status === 'busy'
  return (
    <div className="grid grid-cols-[38px_1fr_88px] items-center gap-3 font-mono text-xs py-1.5 border-b border-border/50 last:border-0">
      <span className={busy ? 'text-accent' : 'text-textDim'}>
        {busy ? '●' : '○'} {String(worker.id).padStart(2, '0')}
      </span>
      <span className="flex items-center gap-2 min-w-0">
        <span className="h-1 flex-1 bg-bgAlt overflow-hidden">
          <span
            className={`block h-full transition-all duration-300 ${busy ? 'bg-accent' : 'bg-border'}`}
            style={{ width: busy ? '70%' : '100%' }}
          />
        </span>
        <span className={`truncate w-32 ${busy ? 'text-text' : 'text-textDim'}`}>
          {busy ? job?.label ?? worker.job_type : 'idle'}
        </span>
      </span>
      <span className="text-textDim text-right">{worker.processed_count} done</span>
    </div>
  )
}

export const ProcessingView: React.FC<{ upload: UploadView; onReset: () => void }> = ({ upload, onReset }) => {
  const done = upload.jobs.filter((j) => j.status === 'succeeded').length
  const failed = upload.jobs.filter((j) => j.status === 'failed').length
  const running = upload.jobs.filter((j) => j.status === 'running').length
  const waiting = upload.jobs.filter((j) => j.status === 'queued').length
  const finished = upload.status === 'completed' || upload.status === 'completed_with_failures'

  return (
    <section className="max-w-5xl mx-auto px-6 pt-24 pb-16 space-y-6">
      <div className="flex items-start justify-between gap-4 flex-wrap">
        <div>
          <div className="font-mono text-xs text-accent tracking-widest mb-1">
            {finished ? 'PROCESSING COMPLETE' : 'PROCESSING'}
          </div>
          <h2 className="text-2xl font-light">{upload.filename}</h2>
          <div className="font-mono text-xs text-textDim mt-1">
            {upload.media_type} · engine: {upload.engine} · upload {upload.upload_id.slice(0, 8)}
          </div>
        </div>
        <button onClick={onReset} className="px-4 py-2 border border-border font-mono text-xs hover:border-accent transition-colors">
          ← New upload
        </button>
      </div>

      {upload.engine === 'simulated' && (
        <div className="border border-yellow/40 bg-yellow/5 px-4 py-2 font-mono text-xs text-yellow">
          ffmpeg is not installed on this host — video jobs run through the real queue and workers,
          but their outputs are simulated rather than genuinely transcoded.
        </div>
      )}

      <div className="grid grid-cols-2 md:grid-cols-4 border border-border divide-x divide-border font-mono">
        {[
          ['QUEUE', waiting, 'text-yellow'],
          ['RUNNING', running, 'text-accent'],
          ['DONE', done, 'text-green'],
          ['FAILED', failed, failed ? 'text-red' : 'text-textDim'],
        ].map(([label, value, cls]) => (
          <div key={label as string} className="px-4 py-3">
            <div className="text-xs text-textDim">{label as string}</div>
            <div className={`text-2xl ${cls as string}`}>{String(value).padStart(2, '0')}</div>
          </div>
        ))}
      </div>

      <div className="grid lg:grid-cols-2 gap-6">
        <div className="border border-border bg-surface">
          <div className="px-4 py-2 border-b border-border font-mono text-xs text-textMuted">
            WORKER POOL · {upload.workers?.filter((w) => w.status === 'busy').length ?? 0} active
          </div>
          <div className="px-4 py-2">
            {(upload.workers ?? []).map((w) => (
              <WorkerRow key={w.id} worker={w} jobs={upload.jobs} />
            ))}
          </div>
        </div>

        <div className="border border-border bg-surface">
          <div className="px-4 py-2 border-b border-border font-mono text-xs text-textMuted">
            JOBS · {upload.jobs.length} created from 1 upload
          </div>
          <div className="px-4 py-2 space-y-2">
            {upload.jobs.map((job) => (
              <div key={job.id} className="font-mono text-xs">
                <div className="flex justify-between gap-2">
                  <span className="truncate">{job.label}</span>
                  <span className={STATUS_CLASS[job.status]}>{job.status}</span>
                </div>
                <div className="h-0.5 bg-bgAlt mt-1">
                  <div
                    className={`h-full transition-all duration-500 ${
                      job.status === 'succeeded' ? 'bg-green' : job.status === 'failed' ? 'bg-red' : 'bg-accent'
                    }`}
                    style={{ width: `${bar(job)}%` }}
                  />
                </div>
                {job.error && <div className="text-red mt-0.5 truncate">{job.error}</div>}
              </div>
            ))}
          </div>
        </div>
      </div>

      <div className="border border-border bg-surface">
        <div className="px-4 py-2 border-b border-border font-mono text-xs text-textMuted">RESULTS</div>
        <div className="p-4 grid sm:grid-cols-2 lg:grid-cols-3 gap-3">
          {upload.jobs.filter((j) => j.output_url).map((job) => (
            <a
              key={job.id}
              href={job.output_url}
              target="_blank"
              rel="noreferrer"
              className="border border-border hover:border-accent transition-colors group"
            >
              <div className="aspect-video bg-bgAlt flex items-center justify-center overflow-hidden">
                {job.output_url?.endsWith('.jpg') ? (
                  <img src={job.output_url} alt={job.label} className="max-h-full max-w-full object-contain" />
                ) : (
                  <span className="font-mono text-xs text-textDim">{job.output_url?.split('.').pop()}</span>
                )}
              </div>
              <div className="px-3 py-2 font-mono text-xs flex justify-between gap-2">
                <span className="truncate group-hover:text-accent transition-colors">{job.label}</span>
                <span className="text-textDim">{formatBytes(job.size_bytes)}</span>
              </div>
            </a>
          ))}
          {upload.jobs.every((j) => !j.output_url) && (
            <div className="font-mono text-xs text-textDim">Results appear here as workers finish.</div>
          )}
        </div>
      </div>
    </section>
  )
}
