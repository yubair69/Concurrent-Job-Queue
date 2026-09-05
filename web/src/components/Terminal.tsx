import React, { useState, useEffect, useRef } from 'react'
import { motion, AnimatePresence } from 'framer-motion'

interface TerminalLine {
  id: number
  type: 'prompt' | 'command' | 'output' | 'error' | 'success' | 'comment' | 'empty'
  content: string
  delay: number
}

const buildDemoLines = (): TerminalLine[] => [
  { id: 1, type: 'prompt', content: '$ gotask --help', delay: 0 },
  { id: 2, type: 'comment', content: 'gotask CLI — concurrent background job queue client', delay: 200 },
  { id: 3, type: 'empty', content: '', delay: 200 },
  { id: 4, type: 'comment', content: 'Usage: gotask <command> [arguments]', delay: 200 },
  { id: 5, type: 'comment', content: 'Commands:', delay: 200 },
  { id: 6, type: 'comment', content: '  submit   Submit a new job', delay: 200 },
  { id: 7, type: 'comment', content: '  status   Get job status', delay: 200 },
  { id: 8, type: 'comment', content: '  cancel   Cancel a job', delay: 200 },
  { id: 9, type: 'comment', content: '  workers  List worker metrics', delay: 200 },
  { id: 10, type: 'empty', content: '', delay: 200 },
  { id: 11, type: 'prompt', content: '$ gotask submit --type echo --priority 5', delay: 500 },
  { id: 12, type: 'success', content: 'Job successfully submitted:', delay: 300 },
  { id: 13, type: 'success', content: '  ID:       a1b2c3d4', delay: 100 },
  { id: 14, type: 'success', content: '  Type:     echo', delay: 100 },
  { id: 15, type: 'success', content: '  Status:   queued', delay: 100 },
  { id: 16, type: 'success', content: '  Priority: 5', delay: 100 },
  { id: 17, type: 'empty', content: '', delay: 300 },
  { id: 18, type: 'prompt', content: '$ gotask status a1b2c3d4', delay: 300 },
  { id: 19, type: 'output', content: 'Job Details:', delay: 200 },
  { id: 20, type: 'output', content: '  ID:           a1b2c3d4', delay: 100 },
  { id: 21, type: 'output', content: '  Type:         echo', delay: 100 },
  { id: 22, type: 'output', content: '  Status:       succeeded', delay: 200 },
  { id: 23, type: 'output', content: '  Priority:     5', delay: 100 },
  { id: 24, type: 'output', content: '  Attempts:     1 (Max: 3)', delay: 100 },
  { id: 25, type: 'output', content: '  Created At:   2025-01-15T10:30:00Z', delay: 100 },
  { id: 26, type: 'output', content: '  Started At:   2025-01-15T10:30:01Z', delay: 100 },
  { id: 27, type: 'output', content: '  Completed At: 2025-01-15T10:30:01Z', delay: 100 },
  { id: 28, type: 'empty', content: '', delay: 300 },
  { id: 29, type: 'prompt', content: '$ gotask submit --type sleep --payload \'{"duration_ms": 2000}\' --retries 3', delay: 300 },
  { id: 30, type: 'success', content: 'Job successfully submitted:', delay: 300 },
  { id: 31, type: 'success', content: '  ID:       e5f6g7h8', delay: 100 },
  { id: 32, type: 'success', content: '  Type:     sleep', delay: 100 },
  { id: 33, type: 'success', content: '  Status:   queued', delay: 100 },
  { id: 34, type: 'empty', content: '', delay: 300 },
  { id: 35, type: 'prompt', content: '$ gotask workers', delay: 400 },
  { id: 36, type: 'output', content: 'GoTask System Metrics & Workers:', delay: 300 },
  { id: 37, type: 'output', content: '  Active Workers:   4', delay: 100 },
  { id: 38, type: 'output', content: '  Queued Jobs:      3', delay: 100 },
  { id: 39, type: 'output', content: '  Running Jobs:     1', delay: 100 },
  { id: 40, type: 'output', content: '  Succeeded Jobs:   1', delay: 100 },
  { id: 41, type: 'output', content:  ' Failed Jobs:      0', delay: 100 },
  { id: 42, type: 'output', content: '  Total Processed:  42', delay: 100 },
  { id: 43, type: 'empty', content: '', delay: 500 },
  { id: 44, type: 'prompt', content: '$ gotask cancel e5f6g7h8', delay: 300 },
  { id: 45, type: 'success', content: 'Job e5f6g7h8 cancelled successfully (Status: cancelled)', delay: 200 },
  { id: 46, type: 'empty', content: '', delay: 300 },
  { id: 47, type: 'prompt', content: '$ ', delay: 300 },
]

export const Terminal: React.FC = () => {
  const [lines, setLines] = useState<TerminalLine[]>([])
  const [currentLine, setCurrentLine] = useState(0)
  const [cursorVisible, setCursorVisible] = useState(true)
  const terminalRef = useRef<HTMLDivElement>(null)
  const cursorRef = useRef<number>(0)

  useEffect(() => {
    const allLines = buildDemoLines()
    cursorRef.current = 0

    const showNext = () => {
      if (cursorRef.current >= allLines.length) {
        return
      }
      const line = allLines[cursorRef.current]
      setLines((prev) => [...prev, line])
      setCurrentLine(cursorRef.current + 1)
      cursorRef.current++

      setTimeout(showNext, line.delay)
    }

    setTimeout(showNext, 300)

    const cursorInterval = setInterval(() => {
      setCursorVisible((prev) => !prev)
    }, 500)

    return () => {
      clearInterval(cursorInterval)
    }
  }, [])

  useEffect(() => {
    if (terminalRef.current) {
      terminalRef.current.scrollTop = terminalRef.current.scrollHeight
    }
  }, [lines])

  const getLineClass = (type: TerminalLine['type']) => {
    switch (type) {
      case 'prompt':
        return 'terminal-line prompt text-cyan'
      case 'command':
        return 'terminal-line command text-accent'
      case 'output':
        return 'terminal-line output text-text'
      case 'error':
        return 'terminal-line error text-red'
      case 'success':
        return 'terminal-line success text-green'
      case 'comment':
        return 'terminal-line comment text-textDim'
      case 'empty':
        return 'terminal-line empty h-3'
      default:
        return 'terminal-line output text-text'
    }
  }

  const renderContent = (line: TerminalLine) => {
    if (line.type === 'empty') return '\u00A0'

    if (line.type === 'prompt' || line.type === 'command') {
      const parts = line.content.split(/(\s+)(--\w+)(?:\s+([^\s]+))?/)
      return (
        <span>
          <span className="text-textDim">$</span>{' '}
          <span className="text-accent">
            {parts[0]}{parts[1]}
          </span>{' '}
          <span className="text-blue">{parts[3]}</span>{' '}
          <span className="text-cyan">{parts[5]}</span>
        </span>
      )
    }

    return <span>{line.content}</span>
  }

  return (
    <section id="terminal" className="py-20 bg-bgAlt">
      <div className="max-w-5xl mx-auto px-6">
        <motion.div
          initial={{ y: 30, opacity: 0 }}
          whileInView={{ y: 0, opacity: 1 }}
          viewport={{ once: true }}
          className="text-center mb-8"
        >
          <h2 className="text-3xl font-bold mb-4 font-mono">GoTask CLI</h2>
          <p className="text-textMuted max-w-2xl mx-auto">
            A native Go CLI for job submission, status monitoring, and system management.
            Commands are typed with flag parsing and produce structured output.
          </p>
        </motion.div>

        <div className="relative">
          <div
            ref={terminalRef}
            className="bg-bg border border-border rounded-xl p-4 h-96 overflow-y-auto font-mono text-sm"
          >
            <AnimatePresence>
              {lines.map((line) => (
                <motion.div
                  key={line.id}
                  initial={{ opacity: 0, y: 4 }}
                  animate={{ opacity: 1, y: 0 }}
                  className={getLineClass(line.type)}
                >
                  {renderContent(line)}
                </motion.div>
              ))}
            </AnimatePresence>
            {currentLine < buildDemoLines().length && (
              <motion.div
                className="terminal-line prompt text-cyan"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
              >
                <span className="text-textDim">$</span> <span className="text-accent">gotask</span>
              </motion.div>
            )}
            {currentLine >= buildDemoLines().length && (
              <motion.span
                animate={{ opacity: cursorVisible ? 1 : 0 }}
                className="text-accent"
              >
                _
              </motion.span>
            )}
          </div>

          <motion.div
            className="absolute bottom-4 right-4 text-xs text-textDim font-mono"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ delay: 1 }}
          >
            Press any key to focus
          </motion.div>
        </div>

        <div className="mt-6 grid grid-cols-1 md:grid-cols-4 gap-3">
          <div className="bg-surfaceAlt border border-border rounded-lg p-3">
            <div className="text-xs text-textMuted font-mono mb-1">Commands</div>
            <div className="text-sm text-text font-mono">submit • status • cancel • workers</div>
          </div>
          <div className="bg-surfaceAlt border border-border rounded-lg p-3">
            <div className="text-xs text-textMuted font-mono mb-1">Job Types</div>
            <div className="text-sm text-text font-mono">echo • sleep • email • image-processing</div>
          </div>
          <div className="bg-surfaceAlt border border-border rounded-lg p-3">
            <div className="text-xs text-textMuted font-mono mb-1">Default Endpt</div>
            <div className="text-sm text-cyan font-mono">localhost:8080</div>
          </div>
          <div className="bg-surfaceAlt border border-border rounded-lg p-3">
            <div className="text-xs text-textMuted font-mono mb-1">Env Override</div>
            <div className="text-sm text-yellow font-mono">GOTASK_SERVER</div>
          </div>
        </div>
      </div>
    </section>
  )
}
