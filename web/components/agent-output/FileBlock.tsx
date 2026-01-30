'use client'

import { useState } from 'react'
import { Eye, PenTool, ChevronDown, ChevronRight, Copy, Check } from 'lucide-react'

interface FileBlockProps {
  type: 'read' | 'write'
  path: string
  content?: string
  diff?: string
  timestamp: string
}

export default function FileBlock({ type, path, content, diff, timestamp }: FileBlockProps) {
  const [expanded, setExpanded] = useState(false)
  const [copied, setCopied] = useState(false)
  
  const isRead = type === 'read'
  const hasContent = content || diff
  
  const handleCopy = async (e: React.MouseEvent) => {
    e.stopPropagation()
    try {
      await navigator.clipboard.writeText(path)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch (err) {
      console.error('Failed to copy:', err)
    }
  }

  // 渲染 Diff 内容
  const renderDiff = (diffText: string) => {
    const lines = diffText.split('\n')
    return lines.map((line, idx) => {
      let bgColor = ''
      let textColor = 'text-gray-700'
      
      if (line.startsWith('+') && !line.startsWith('+++')) {
        bgColor = 'bg-green-100'
        textColor = 'text-green-800'
      } else if (line.startsWith('-') && !line.startsWith('---')) {
        bgColor = 'bg-red-100'
        textColor = 'text-red-800'
      } else if (line.startsWith('@@')) {
        bgColor = 'bg-blue-50'
        textColor = 'text-blue-600'
      }
      
      return (
        <div key={idx} className={`px-2 ${bgColor} ${textColor}`}>
          {line || ' '}
        </div>
      )
    })
  }

  return (
    <div className={`rounded-lg border overflow-hidden ${
      isRead ? 'border-yellow-200 bg-yellow-50' : 'border-orange-200 bg-orange-50'
    }`}>
      {/* 头部 */}
      <button
        onClick={() => hasContent && setExpanded(!expanded)}
        className={`w-full flex items-center justify-between px-4 py-2 ${
          hasContent ? 'hover:bg-white/30 cursor-pointer' : 'cursor-default'
        } transition-colors`}
      >
        <div className="flex items-center gap-2 flex-1 min-w-0">
          {hasContent ? (
            expanded ? (
              <ChevronDown className="w-4 h-4 text-gray-500 flex-shrink-0" />
            ) : (
              <ChevronRight className="w-4 h-4 text-gray-500 flex-shrink-0" />
            )
          ) : (
            <span className="w-4" />
          )}
          {isRead ? (
            <Eye className="w-4 h-4 text-yellow-600 flex-shrink-0" />
          ) : (
            <PenTool className="w-4 h-4 text-orange-500 flex-shrink-0" />
          )}
          <span className={`text-sm font-medium ${isRead ? 'text-yellow-700' : 'text-orange-700'} flex-shrink-0`}>
            {isRead ? '读取文件' : '修改文件'}
          </span>
          <code className="text-xs px-2 py-0.5 rounded bg-white/50 text-gray-700 truncate">
            {path}
          </code>
        </div>
        <div className="flex items-center gap-2 flex-shrink-0">
          <button
            onClick={handleCopy}
            className={`p-1 rounded ${isRead ? 'hover:bg-yellow-100 text-yellow-600' : 'hover:bg-orange-100 text-orange-600'}`}
            title="复制路径"
          >
            {copied ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
          </button>
          <span className={`text-xs ${isRead ? 'text-yellow-400' : 'text-orange-400'}`}>
            {timestamp}
          </span>
        </div>
      </button>
      
      {/* 展开内容 */}
      {expanded && hasContent && (
        <div className="border-t border-white/50 bg-white/30">
          {diff ? (
            <pre className="text-xs font-mono overflow-x-auto max-h-64 overflow-y-auto">
              {renderDiff(diff)}
            </pre>
          ) : content ? (
            <pre className="text-xs font-mono p-3 overflow-x-auto max-h-64 overflow-y-auto text-gray-700">
              {content.slice(0, 2000)}{content.length > 2000 ? '\n... (内容已截断)' : ''}
            </pre>
          ) : null}
        </div>
      )}
    </div>
  )
}
