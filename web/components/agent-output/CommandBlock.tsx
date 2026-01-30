'use client'

import { useState } from 'react'
import { Terminal, ChevronDown, ChevronRight, Copy, Check, CheckCircle, XCircle } from 'lucide-react'

interface CommandBlockProps {
  command: string
  output?: string
  exitCode?: number
  timestamp: string
}

export default function CommandBlock({ command, output, exitCode, timestamp }: CommandBlockProps) {
  const [expanded, setExpanded] = useState(!!output)
  const [copied, setCopied] = useState(false)
  
  const hasOutput = output && output.trim().length > 0
  const isSuccess = exitCode === undefined || exitCode === 0
  
  const handleCopy = async (e: React.MouseEvent) => {
    e.stopPropagation()
    try {
      await navigator.clipboard.writeText(command)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch (err) {
      console.error('Failed to copy:', err)
    }
  }

  return (
    <div className="rounded-lg border border-gray-300 bg-gray-100 overflow-hidden">
      {/* 头部 */}
      <button
        onClick={() => hasOutput && setExpanded(!expanded)}
        className={`w-full flex items-center justify-between px-4 py-2 ${
          hasOutput ? 'hover:bg-gray-200 cursor-pointer' : 'cursor-default'
        } transition-colors`}
      >
        <div className="flex items-center gap-2 flex-1 min-w-0">
          {hasOutput ? (
            expanded ? (
              <ChevronDown className="w-4 h-4 text-gray-500 flex-shrink-0" />
            ) : (
              <ChevronRight className="w-4 h-4 text-gray-500 flex-shrink-0" />
            )
          ) : (
            <span className="w-4" />
          )}
          <Terminal className="w-4 h-4 text-gray-600 flex-shrink-0" />
          <span className="text-sm font-medium text-gray-700 flex-shrink-0">执行命令</span>
        </div>
        <div className="flex items-center gap-2 flex-shrink-0">
          {exitCode !== undefined && (
            isSuccess 
              ? <CheckCircle className="w-4 h-4 text-green-500" />
              : <XCircle className="w-4 h-4 text-red-500" />
          )}
          <button
            onClick={handleCopy}
            className="p-1 rounded hover:bg-gray-200 text-gray-600"
            title="复制命令"
          >
            {copied ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
          </button>
          <span className="text-xs text-gray-400">{timestamp}</span>
        </div>
      </button>
      
      {/* 命令显示 */}
      <div className="px-4 py-2 bg-gray-900 border-t border-gray-300">
        <div className="flex items-center gap-2">
          <span className="text-green-400 font-mono text-sm">$</span>
          <code className="text-gray-100 font-mono text-sm flex-1 overflow-x-auto">
            {command}
          </code>
        </div>
      </div>
      
      {/* 输出内容 */}
      {expanded && hasOutput && (
        <div className="bg-gray-800 border-t border-gray-700 max-h-64 overflow-y-auto">
          <pre className="p-3 text-xs font-mono text-gray-300 whitespace-pre-wrap overflow-x-auto">
            {output.slice(0, 5000)}{output.length > 5000 ? '\n... (输出已截断)' : ''}
          </pre>
        </div>
      )}
      
      {/* 退出码 */}
      {exitCode !== undefined && (
        <div className={`px-4 py-1 text-xs border-t ${
          isSuccess 
            ? 'bg-green-50 border-green-200 text-green-600' 
            : 'bg-red-50 border-red-200 text-red-600'
        }`}>
          退出码: {exitCode}
        </div>
      )}
    </div>
  )
}
