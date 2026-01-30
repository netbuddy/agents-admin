'use client'

import { useState } from 'react'
import { Wrench, ChevronDown, ChevronRight, Check, X } from 'lucide-react'

interface ToolBlockProps {
  type: 'tool_use_start' | 'tool_result'
  toolName: string
  input?: any
  output?: any
  success?: boolean
  timestamp: string
}

export default function ToolBlock({ type, toolName, input, output, success = true, timestamp }: ToolBlockProps) {
  const [expanded, setExpanded] = useState(type === 'tool_result')
  
  const isStart = type === 'tool_use_start'
  
  return (
    <div className={`rounded-lg border overflow-hidden ${
      isStart 
        ? 'border-purple-200 bg-purple-50' 
        : success 
          ? 'border-green-200 bg-green-50' 
          : 'border-red-200 bg-red-50'
    }`}>
      {/* 头部 */}
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center justify-between px-4 py-2 hover:bg-white/30 transition-colors"
      >
        <div className="flex items-center gap-2">
          {expanded ? (
            <ChevronDown className="w-4 h-4 text-gray-500" />
          ) : (
            <ChevronRight className="w-4 h-4 text-gray-500" />
          )}
          <Wrench className={`w-4 h-4 ${isStart ? 'text-purple-500' : success ? 'text-green-500' : 'text-red-500'}`} />
          <span className={`text-sm font-medium ${isStart ? 'text-purple-700' : success ? 'text-green-700' : 'text-red-700'}`}>
            {isStart ? '调用工具' : '工具结果'}
          </span>
          <code className="text-xs px-2 py-0.5 rounded bg-white/50 text-gray-700">{toolName}</code>
          {!isStart && (
            success 
              ? <Check className="w-4 h-4 text-green-500" />
              : <X className="w-4 h-4 text-red-500" />
          )}
        </div>
        <span className={`text-xs ${isStart ? 'text-purple-400' : success ? 'text-green-400' : 'text-red-400'}`}>
          {timestamp}
        </span>
      </button>
      
      {/* 展开内容 */}
      {expanded && (
        <div className="px-4 py-3 border-t border-white/50 bg-white/30">
          {input && (
            <div className="mb-3">
              <label className="block text-xs text-gray-500 mb-1">输入参数</label>
              <pre className="text-xs bg-white rounded p-2 overflow-x-auto border border-gray-200">
                {typeof input === 'string' ? input : JSON.stringify(input, null, 2)}
              </pre>
            </div>
          )}
          
          {output !== undefined && (
            <div>
              <label className="block text-xs text-gray-500 mb-1">输出结果</label>
              <pre className={`text-xs rounded p-2 overflow-x-auto border ${
                success ? 'bg-white border-gray-200' : 'bg-red-50 border-red-200'
              }`}>
                {typeof output === 'string' 
                  ? output.slice(0, 1000) + (output.length > 1000 ? '...' : '')
                  : JSON.stringify(output, null, 2)?.slice(0, 1000)
                }
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
