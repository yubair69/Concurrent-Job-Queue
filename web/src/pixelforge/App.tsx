import React, { useCallback, useEffect, useState } from 'react'
import { UploadStage } from './UploadStage'
import { ProcessingView } from './ProcessingView'
import { UploadView, fetchUpload, uploadFile } from './api'

export const PixelForgeApp: React.FC = () => {
  const [upload, setUpload] = useState<UploadView | null>(null)
  const [busy, setBusy] = useState(false)
  const [progress, setProgress] = useState(0)
  const [error, setError] = useState<string | null>(null)

  const process = useCallback(async (file: File) => {
    setBusy(true)
    setError(null)
    setProgress(0)
    try {
      setUpload(await uploadFile(file, setProgress))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'upload failed')
    } finally {
      setBusy(false)
    }
  }, [])

  // Deep link: /?upload=<id> reopens a previous processing run.
  useEffect(() => {
    const id = new URLSearchParams(window.location.search).get('upload')
    if (id) fetchUpload(id).then(setUpload).catch(() => {})
  }, [])

  // Poll real backend state while any job is still in flight.
  useEffect(() => {
    if (!upload) return
    const settled = upload.status === 'completed' || upload.status === 'completed_with_failures'
    const interval = window.setInterval(
      () => {
        fetchUpload(upload.upload_id).then(setUpload).catch(() => {})
      },
      settled ? 3000 : 500
    )
    return () => window.clearInterval(interval)
  }, [upload])

  return (
    <div className="min-h-screen bg-bg text-text">
      <header className="fixed top-0 inset-x-0 z-50 border-b border-border bg-bg/85 backdrop-blur-sm">
        <div className="max-w-5xl mx-auto px-6 py-3 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-5 h-5 border border-accent grid grid-cols-2 gap-px p-px">
              <span className="bg-accent" /><span className="bg-accent/40" />
              <span className="bg-accent/40" /><span className="bg-accent" />
            </div>
            <span className="font-mono text-sm font-bold">
              Pixel<span className="text-accent">Forge</span>
            </span>
          </div>
          <span className="font-mono text-xs text-textDim">powered by GoTask</span>
        </div>
      </header>

      <main>
        {upload ? (
          <ProcessingView upload={upload} onReset={() => setUpload(null)} />
        ) : (
          <UploadStage onProcess={process} busy={busy} progress={progress} error={error} />
        )}
      </main>

      <footer className="border-t border-border py-6">
        <div className="max-w-5xl mx-auto px-6 font-mono text-xs text-textDim flex justify-between gap-4 flex-wrap">
          <span>PixelForge · concurrent media processing on the GoTask engine</span>
          <span>Go · priority queue · worker pool · SQLite</span>
        </div>
      </footer>
    </div>
  )
}
