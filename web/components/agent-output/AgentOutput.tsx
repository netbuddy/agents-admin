'use client'

import { useEffect, useRef } from 'react'
import { Bot, FileText, Terminal, PenTool, AlertCircle, Rocket, PartyPopper, Loader2, Wrench, Eye, Info, CheckCircle2 } from 'lucide-react'
import MessageBlock from './MessageBlock'
import ToolBlock from './ToolBlock'
import FileBlock from './FileBlock'
import CommandBlock from './CommandBlock'

interface Event {
  id: number
  run_id: string
  seq: number
  type: string
  timestamp: string
  payload: any
}

interface AgentOutputProps {
  events: Event[]
  isStreaming?: boolean
  error?: string | null
}

// 事件类型图标映射
const eventIcons: Record<string, React.ReactNode> = {
  run_started: <Rocket className="w-4 h-4 text-green-500" />,
  run_completed: <PartyPopper className="w-4 h-4 text-green-500" />,
  system_info: <Info className="w-4 h-4 text-blue-400" />,
  result: <CheckCircle2 className="w-4 h-4 text-green-500" />,
  message: <Bot className="w-4 h-4 text-blue-500" />,
  thinking: <Loader2 className="w-4 h-4 text-gray-400 animate-spin" />,
  tool_use_start: <Wrench className="w-4 h-4 text-purple-500" />,
  tool_result: <Wrench className="w-4 h-4 text-purple-500" />,
  file_read: <Eye className="w-4 h-4 text-yellow-600" />,
  file_write: <PenTool className="w-4 h-4 text-orange-500" />,
  command: <Terminal className="w-4 h-4 text-gray-600" />,
  command_output: <Terminal className="w-4 h-4 text-gray-600" />,
  error: <AlertCircle className="w-4 h-4 text-red-500" />,
}

export default function AgentOutput({ events, isStreaming, error }: AgentOutputProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const endRef = useRef<HTMLDivElement>(null)

  // 自动滚动到底部
  useEffect(() => {
    if (isStreaming) {
      endRef.current?.scrollIntoView({ behavior: 'smooth' })
    }
  }, [events, isStreaming])

  // 合并相邻的相同类型事件（例如连续的 message）
  const groupedEvents = events.reduce<Event[][]>((groups, event) => {
    const lastGroup = groups[groups.length - 1]
    
    // 可合并的事件类型
    if (lastGroup && 
        lastGroup[0].type === event.type && 
        ['command_output'].includes(event.type)) {
      lastGroup.push(event)
    } else {
      groups.push([event])
    }
    
    return groups
  }, [])

  const renderEventGroup = (eventGroup: Event[], index: number) => {
    const event = eventGroup[0]
    const timestamp = new Date(event.timestamp).toLocaleTimeString('zh-CN')
    
    switch (event.type) {
      case 'run_started':
        return (
          <div key={index} className="flex items-center gap-3 py-3 px-4 bg-green-50 rounded-lg border border-green-200">
            {eventIcons.run_started}
            <div className="flex-1">
              <span className="text-sm font-medium text-green-700">任务开始执行</span>
              <span className="text-xs text-green-600 ml-2">{timestamp}</span>
            </div>
          </div>
        )
        
      case 'run_completed':
        return (
          <div key={index} className="flex items-center gap-3 py-3 px-4 bg-green-50 rounded-lg border border-green-200">
            {eventIcons.run_completed}
            <div className="flex-1">
              <span className="text-sm font-medium text-green-700">任务执行完成</span>
              <span className="text-xs text-green-600 ml-2">{timestamp}</span>
            </div>
          </div>
        )

      case 'system_info':
        // 显示 Agent 系统信息（版本、模型等）
        return (
          <div key={index} className="flex items-start gap-3 py-2 px-4 bg-blue-50 rounded-lg border border-blue-200">
            {eventIcons.system_info}
            <div className="flex-1">
              <span className="text-xs font-medium text-blue-700">Agent 初始化</span>
              <div className="text-xs text-blue-600 mt-1 space-x-2">
                {event.payload?.qwen_code_version && <span>版本: {event.payload.qwen_code_version}</span>}
                {event.payload?.model && <span>模型: {event.payload.model}</span>}
                {event.payload?.permission_mode && <span>模式: {event.payload.permission_mode}</span>}
              </div>
            </div>
            <span className="text-xs text-blue-400">{timestamp}</span>
          </div>
        )

      case 'result':
        // 显示执行结果摘要
        return (
          <div key={index} className="flex items-start gap-3 py-3 px-4 bg-green-50 rounded-lg border border-green-200">
            {eventIcons.result}
            <div className="flex-1">
              <span className="text-sm font-medium text-green-700">执行结果</span>
              {event.payload?.usage && (
                <div className="text-xs text-green-600 mt-1">
                  Token 使用: {event.payload.usage.total_tokens?.toLocaleString() || '-'}
                </div>
              )}
            </div>
            <span className="text-xs text-green-400">{timestamp}</span>
          </div>
        )
        
      case 'message':
        return (
          <MessageBlock 
            key={index} 
            content={event.payload?.content || event.payload?.text || ''} 
            timestamp={timestamp}
          />
        )
        
      case 'thinking':
        return (
          <div key={index} className="flex items-start gap-3 py-2 px-4 bg-gray-50 rounded-lg border border-gray-200 italic">
            {eventIcons.thinking}
            <div className="flex-1 text-sm text-gray-500">
              {event.payload?.content || '思考中...'}
            </div>
          </div>
        )
        
      case 'tool_use_start':
      case 'tool_result':
        return (
          <ToolBlock 
            key={index}
            type={event.type}
            toolName={event.payload?.tool_name || event.payload?.tool || event.payload?.name || 'unknown'}
            input={event.payload?.input || event.payload?.arguments}
            output={event.type === 'tool_result' ? event.payload?.output || event.payload?.result : undefined}
            success={event.payload?.success !== false}
            timestamp={timestamp}
          />
        )
        
      case 'file_read':
        return (
          <FileBlock
            key={index}
            type="read"
            path={event.payload?.path || event.payload?.file}
            timestamp={timestamp}
          />
        )
        
      case 'file_write':
        return (
          <FileBlock
            key={index}
            type="write"
            path={event.payload?.path || event.payload?.file}
            content={event.payload?.content}
            diff={event.payload?.diff}
            timestamp={timestamp}
          />
        )
        
      case 'command':
      case 'command_output':
        const output = eventGroup.map(e => e.payload?.output || e.payload?.stdout || '').join('')
        return (
          <CommandBlock
            key={index}
            command={event.payload?.command || event.payload?.cmd}
            output={output}
            exitCode={eventGroup[eventGroup.length - 1].payload?.exit_code}
            timestamp={timestamp}
          />
        )
        
      case 'error':
        return (
          <div key={index} className="flex items-start gap-3 py-3 px-4 bg-red-50 rounded-lg border border-red-200">
            {eventIcons.error}
            <div className="flex-1">
              <span className="text-sm font-medium text-red-700">错误</span>
              <p className="text-sm text-red-600 mt-1">{event.payload?.message || event.payload?.error || '未知错误'}</p>
            </div>
            <span className="text-xs text-red-400">{timestamp}</span>
          </div>
        )
        
      default:
        // 默认渲染：显示事件类型和 payload
        return (
          <div key={index} className="flex items-start gap-3 py-2 px-4 bg-gray-50 rounded-lg border border-gray-200">
            <span className="text-xs px-2 py-0.5 rounded bg-gray-200 text-gray-600">{event.type}</span>
            <div className="flex-1 text-sm text-gray-600 font-mono overflow-x-auto">
              <pre className="whitespace-pre-wrap text-xs">
                {JSON.stringify(event.payload, null, 2)?.slice(0, 200)}
              </pre>
            </div>
            <span className="text-xs text-gray-400">{timestamp}</span>
          </div>
        )
    }
  }

  return (
    <div ref={containerRef} className="h-full overflow-y-auto bg-gray-50">
      <div className="p-4 space-y-3">
        {events.length === 0 && !isStreaming && (
          <div className="text-center py-12 text-gray-400">
            <Terminal className="w-12 h-12 mx-auto mb-4 opacity-50" />
            <p className="text-sm">等待输出...</p>
          </div>
        )}
        
        {groupedEvents.map((group, idx) => renderEventGroup(group, idx))}
        
        {isStreaming && (
          <div className="flex items-center gap-2 py-2 px-4 text-blue-600">
            <Loader2 className="w-4 h-4 animate-spin" />
            <span className="text-sm">Agent 正在工作中...</span>
          </div>
        )}
        
        {error && (
          <div className="flex items-start gap-3 py-3 px-4 bg-red-50 rounded-lg border border-red-200">
            <AlertCircle className="w-5 h-5 text-red-500 flex-shrink-0" />
            <div className="flex-1">
              <span className="text-sm font-medium text-red-700">执行失败</span>
              <p className="text-sm text-red-600 mt-1">{error}</p>
            </div>
          </div>
        )}
        
        <div ref={endRef} />
      </div>
    </div>
  )
}
