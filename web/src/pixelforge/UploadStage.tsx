import React, { useCallback, useRef, useState } from 'react'

interface Props {
  onProcess: (file: File) => void
  busy: boolean
  progress: number
  error: string | null
}

const ACCEPTED = '.jpg,.jpeg,.png,.gif,.bmp,.webp,.mp4,.mov,.mkv,.webm,.avi,.m4v'

const detectKind = (file: File): 'image' | 'video' | 'unsupported' => {
  const ext = file.name.toLowerCase().split('.').pop() ?? ''
  if (['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp'].includes(ext)) return 'image'
  if (['mp4', 'mov', 'mkv', 'webm', 'avi', 'm4v'].includes(ext)) return 'video'
  return 'unsupported'
}

const PIPELINES: Record<'image' | 'video', string[]> = {
  image: ['Read metadata', 'Generate thumbnail', 'Resize 1280px', 'Compress', 'Optimized version'],
  video: ['Generate metadata', 'Generate thumbnail', 'Extract audio', 'Transcode 480p', 'Transcode 720p', 'Transcode 1080p'],
}

export const UploadStage: React.FC<Props> = ({ onProcess, busy, progress, error }) => {
  const [file, setFile] = useState<File | null>(null)
  const [preview, setPreview] = useState<string | null>(null)
  const [dragging, setDragging] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  const select = useCallback((next: File | null) => {
    if (!next) return
    setFile(next)
    setPreview((old) => {
      if (old) URL.revokeObjectURL(old)
      return URL.createObjectURL(next)
    })
  }, [])

  const kind = file ? detectKind(file) : null

  return (
    <section className="max-w-5xl mx-auto px-6 pt-24 pb-10">
      <div className="mb-10">
        <div className="font-mono text-xs text-accent tracking-widest mb-3">CONCURRENT MEDIA PROCESSING</div>
        <h1 className="text-4xl md:text-5xl font-light leading-tight mb-4">
          One upload becomes
          <br />
          <span className="text-accent font-medium">a queue of parallel jobs.</span>
        </h1>
        <p className="text-textMuted max-w-xl">
          PixelForge splits every file into independent processing jobs and hands them to the
          GoTask engine — a priority queue feeding a pool of concurrent Go workers.
        </p>
        <div className="flex flex-wrap gap-x-3 gap-y-1 mt-4 font-mono text-xs text-textDim">
          <span>UPLOAD</span><span className="text-accent">→</span>
          <span>GOTASK QUEUE</span><span className="text-accent">→</span>
          <span>WORKER POOL</span><span className="text-accent">→</span>
          <span>RESULTS</span>
        </div>
      </div>

      <div
        onDragOver={(e) => { e.preventDefault(); setDragging(true) }}
        onDragLeave={() => setDragging(false)}
        onDrop={(e) => {
          e.preventDefault()
          setDragging(false)
          select(e.dataTransfer.files?.[0] ?? null)
        }}
        className={`border border-dashed transition-colors ${
          dragging ? 'border-accent bg-accent/5' : 'border-border bg-surface'
        }`}
      >
        {!file ? (
          <button
            onClick={() => inputRef.current?.click()}
            className="w-full py-20 flex flex-col items-center gap-3 text-center"
          >
            <div className="w-10 h-10 border border-accent/60 flex items-center justify-center text-accent font-mono">
              +
            </div>
            <div className="font-mono text-sm">Drop an image or video here</div>
            <div className="font-mono text-xs text-textDim">or click to browse · JPG PNG GIF WEBP · MP4 MOV MKV WEBM</div>
          </button>
        ) : (
          <div className="p-5 grid md:grid-cols-[220px_1fr] gap-5 items-start">
            <div className="border border-border bg-bgAlt aspect-video flex items-center justify-center overflow-hidden">
              {kind === 'image' && preview ? (
                <img src={preview} alt={file.name} className="max-h-full max-w-full object-contain" />
              ) : preview ? (
                <video src={preview} className="max-h-full max-w-full" muted />
              ) : null}
            </div>

            <div className="font-mono text-xs space-y-3">
              <div className="grid grid-cols-[90px_1fr] gap-y-1 text-textMuted">
                <span className="text-textDim">file</span><span className="text-text truncate">{file.name}</span>
                <span className="text-textDim">type</span><span className="text-accent">{kind}</span>
                <span className="text-textDim">size</span><span>{(file.size / 1024 / 1024).toFixed(2)} MB</span>
                <span className="text-textDim">jobs</span>
                <span>{kind !== 'unsupported' ? PIPELINES[kind as 'image' | 'video'].length : 0} will be queued</span>
              </div>

              {kind !== 'unsupported' && (
                <div className="border-t border-border pt-3">
                  <div className="text-textDim mb-2">PIPELINE</div>
                  <div className="flex flex-wrap gap-1.5">
                    {PIPELINES[kind as 'image' | 'video'].map((step) => (
                      <span key={step} className="border border-border px-2 py-0.5 text-textMuted">{step}</span>
                    ))}
                  </div>
                </div>
              )}

              {busy && (
                <div className="border-t border-border pt-3">
                  <div className="flex justify-between text-textDim mb-1">
                    <span>uploading</span><span>{progress}%</span>
                  </div>
                  <div className="h-1 bg-bgAlt"><div className="h-full bg-accent transition-all" style={{ width: `${progress}%` }} /></div>
                </div>
              )}

              {error && <div className="text-red border-t border-border pt-3">{error}</div>}

              <div className="flex gap-2 pt-1">
                <button
                  disabled={busy || kind === 'unsupported'}
                  onClick={() => file && onProcess(file)}
                  className="px-4 py-2 bg-accent text-bg font-bold hover:bg-accentHover transition-colors disabled:opacity-40"
                >
                  {busy ? 'Uploading…' : 'Process'}
                </button>
                <button
                  disabled={busy}
                  onClick={() => { setFile(null); setPreview(null) }}
                  className="px-4 py-2 border border-border hover:border-accent transition-colors disabled:opacity-40"
                >
                  Clear
                </button>
              </div>
            </div>
          </div>
        )}
        <input
          ref={inputRef}
          type="file"
          accept={ACCEPTED}
          className="hidden"
          onChange={(e) => select(e.target.files?.[0] ?? null)}
        />
      </div>
    </section>
  )
}
