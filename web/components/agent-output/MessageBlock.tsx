'use client'

import { useState } from 'react'
import { Bot, ChevronDown, ChevronUp, Copy, Check } from 'lucide-react'

interface MessageBlockProps {
  content: string
  timestamp: string
}

export default function MessageBlock({ content, timestamp }: MessageBlockProps) {
  const [expanded, setExpanded] = useState(true)
  const [copied, setCopied] = useState(false)
  
  const isLong = content.length > 500
  const displayContent = expanded || !isLong ? content : content.slice(0, 500) + '...'

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(content)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch (err) {
      console.error('Failed to copy:', err)
    }
  }

  // 简单的 Markdown 渲染（处理代码块、粗体、列表等）
  const renderContent = (text: string) => {
    // 处理代码块
    const parts = text.split(/(```[\s\S]*?```)/g)
    
    return parts.map((part, idx) => {
      if (part.startsWith('```')) {
        // 代码块
        const match = part.match(/```(\w+)?\n?([\s\S]*?)```/)
        if (match) {
          const [, lang, code] = match
          return (
            <div key={idx} className="my-2 rounded-lg overflow-hidden border border-gray-200">
              {lang && (
                <div className="bg-gray-100 px-3 py-1 text-xs text-gray-500 border-b border-gray-200">
                  {lang}
                </div>
              )}
              <pre className="bg-gray-900 text-gray-100 p-3 overflow-x-auto text-xs">
                <code>{code.trim()}</code>
              </pre>
            </div>
          )
        }
      }
      
      // 普通文本，处理行内格式
      return (
        <span key={idx} className="whitespace-pre-wrap">
          {part.split('\n').map((line, lineIdx) => {
            // 处理列表
            if (line.match(/^[\-\*]\s/)) {
              return (
                <div key={lineIdx} className="flex gap-2">
                  <span className="text-gray-400">•</span>
                  <span>{line.slice(2)}</span>
                </div>
              )
            }
            // 处理粗体
            const formatted = line.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
            if (formatted !== line) {
              return (
                <div key={lineIdx} dangerouslySetInnerHTML={{ __html: formatted }} />
              )
            }
            return lineIdx > 0 ? <div key={lineIdx}>{line || <br />}</div> : line
          })}
        </span>
      )
    })
  }

  return (
    <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
      {/* 头部 */}
      <div className="flex items-center justify-between px-3 sm:px-4 py-2 bg-blue-50 border-b border-gray-200">
        <div className="flex items-center gap-2">
          <Bot className="w-4 h-4 text-blue-500" />
          <span className="text-sm font-medium text-blue-700">Agent</span>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={handleCopy}
            className="p-1 hover:bg-blue-100 rounded text-blue-600"
            title="复制"
          >
            {copied ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
          </button>
          <span className="text-xs text-blue-400">{timestamp}</span>
        </div>
      </div>
      
      {/* 内容 */}
      <div className="px-3 sm:px-4 py-2 sm:py-3 text-sm text-gray-700">
        {renderContent(displayContent)}
      </div>
      
      {/* 展开/收起按钮 */}
      {isLong && (
        <button
          onClick={() => setExpanded(!expanded)}
          className="w-full py-2 flex items-center justify-center gap-1 text-sm text-blue-600 hover:bg-blue-50 border-t border-gray-200"
        >
          {expanded ? (
            <>
              <ChevronUp className="w-4 h-4" />
              收起
            </>
          ) : (
            <>
              <ChevronDown className="w-4 h-4" />
              展开全部
            </>
          )}
        </button>
      )}
    </div>
  )
}
