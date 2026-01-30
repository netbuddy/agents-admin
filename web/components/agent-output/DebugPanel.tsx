'use client'

import { useEffect, useRef, useState, useMemo, useCallback } from 'react'
import { Search, ChevronDown, ChevronRight, Filter, ArrowDown, Pause, Copy, Check, Maximize2, X, Minimize2 } from 'lucide-react'
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter'
import { solarizedlight } from 'react-syntax-highlighter/dist/esm/styles/prism'

interface RawEvent {
  id: number
  run_id: string
  seq: number
  type: string
  timestamp: string
  payload: any
  raw?: string  // 原始 CLI 输出
}

interface DebugPanelProps {
  events: RawEvent[]
  isStreaming?: boolean
}

// Solarized Light 护眼配色方案
// 基于 Solarized 调色板，降低饱和度，柔和护眼
const eventTypeColors: Record<string, { bg: string; text: string; border: string; badge: string }> = {
  run_started: { bg: 'bg-[#eee8d5]', text: 'text-[#859900]', border: 'border-[#859900]/30', badge: '#859900' },
  run_completed: { bg: 'bg-[#eee8d5]', text: 'text-[#859900]', border: 'border-[#859900]/30', badge: '#859900' },
  system_info: { bg: 'bg-[#eee8d5]', text: 'text-[#268bd2]', border: 'border-[#268bd2]/30', badge: '#268bd2' },
  system: { bg: 'bg-[#eee8d5]', text: 'text-[#268bd2]', border: 'border-[#268bd2]/30', badge: '#268bd2' },
  result: { bg: 'bg-[#eee8d5]', text: 'text-[#2aa198]', border: 'border-[#2aa198]/30', badge: '#2aa198' },
  message: { bg: 'bg-[#eee8d5]', text: 'text-[#6c71c4]', border: 'border-[#6c71c4]/30', badge: '#6c71c4' },
  assistant: { bg: 'bg-[#eee8d5]', text: 'text-[#6c71c4]', border: 'border-[#6c71c4]/30', badge: '#6c71c4' },
  thinking: { bg: 'bg-[#eee8d5]', text: 'text-[#d33682]', border: 'border-[#d33682]/30', badge: '#d33682' },
  tool_use_start: { bg: 'bg-[#eee8d5]', text: 'text-[#cb4b16]', border: 'border-[#cb4b16]/30', badge: '#cb4b16' },
  tool_result: { bg: 'bg-[#eee8d5]', text: 'text-[#cb4b16]', border: 'border-[#cb4b16]/30', badge: '#cb4b16' },
  file_read: { bg: 'bg-[#eee8d5]', text: 'text-[#b58900]', border: 'border-[#b58900]/30', badge: '#b58900' },
  file_write: { bg: 'bg-[#eee8d5]', text: 'text-[#cb4b16]', border: 'border-[#cb4b16]/30', badge: '#cb4b16' },
  command: { bg: 'bg-[#eee8d5]', text: 'text-[#657b83]', border: 'border-[#657b83]/30', badge: '#657b83' },
  command_output: { bg: 'bg-[#eee8d5]', text: 'text-[#657b83]', border: 'border-[#657b83]/30', badge: '#657b83' },
  error: { bg: 'bg-[#eee8d5]', text: 'text-[#dc322f]', border: 'border-[#dc322f]/30', badge: '#dc322f' },
  warning: { bg: 'bg-[#eee8d5]', text: 'text-[#b58900]', border: 'border-[#b58900]/30', badge: '#b58900' },
}

const defaultColor = { bg: 'bg-[#eee8d5]', text: 'text-[#657b83]', border: 'border-[#657b83]/30', badge: '#657b83' }

// 自定义语法高亮主题（基于 Solarized Light 护眼配色）
const customStyle = {
  ...solarizedlight,
  'pre[class*="language-"]': {
    ...solarizedlight['pre[class*="language-"]'],
    margin: 0,
    padding: '12px',
    fontSize: '12px',
    background: '#fdf6e3',  // Solarized Light 背景色
    borderRadius: '6px',
    border: '1px solid #eee8d5',
  },
  'code[class*="language-"]': {
    ...solarizedlight['code[class*="language-"]'],
    fontSize: '12px',
    color: '#657b83',  // Solarized 基础文字色
  },
}

export default function DebugPanel({ events, isStreaming }: DebugPanelProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedTypes, setSelectedTypes] = useState<Set<string>>(new Set())
  const [expandedItems, setExpandedItems] = useState<Set<number>>(new Set())
  const [autoScroll, setAutoScroll] = useState(true)
  const [showFilter, setShowFilter] = useState(false)
  const [copiedId, setCopiedId] = useState<number | null>(null)
  const [maximizedEvent, setMaximizedEvent] = useState<RawEvent | null>(null)

  // 获取所有事件类型（从原始数据中提取）
  const eventTypes = useMemo(() => {
    const types = new Set<string>()
    events.forEach(e => {
      // 优先从 raw 中提取 type
      if (e.raw) {
        try {
          const parsed = JSON.parse(e.raw)
          types.add(parsed.type || e.type)
        } catch {
          types.add(e.type)
        }
      } else {
        types.add(e.type)
      }
    })
    return Array.from(types).sort()
  }, [events])

  // 获取原始类型（从 raw 字段解析）
  const getRawType = (event: RawEvent): string => {
    if (event.raw) {
      try {
        const parsed = JSON.parse(event.raw)
        return parsed.type || event.type
      } catch {
        return event.type
      }
    }
    return event.type
  }

  // 只保留有 raw 字段的事件（真正的 agent 原始输出）
  // 平台生成的事件（run_started, run_completed）没有 raw 字段
  const rawEvents = useMemo(() => {
    return events.filter(event => event.raw && event.raw.trim() !== '')
  }, [events])

  // 过滤事件
  const filteredEvents = useMemo(() => {
    return rawEvents.filter(event => {
      const rawType = getRawType(event)
      // 类型过滤
      if (selectedTypes.size > 0 && !selectedTypes.has(rawType)) {
        return false
      }
      // 搜索过滤
      if (searchQuery) {
        const query = searchQuery.toLowerCase()
        const rawContent = event.raw || JSON.stringify(event.payload)
        return rawContent.toLowerCase().includes(query) || rawType.toLowerCase().includes(query)
      }
      return true
    })
  }, [rawEvents, selectedTypes, searchQuery])

  // 自动滚动到底部
  useEffect(() => {
    if (autoScroll && isStreaming && containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight
    }
  }, [filteredEvents, autoScroll, isStreaming])

  // 监听滚动，判断是否在底部
  const handleScroll = useCallback(() => {
    if (!containerRef.current) return
    const { scrollTop, scrollHeight, clientHeight } = containerRef.current
    const isAtBottom = scrollHeight - scrollTop - clientHeight < 50
    if (isAtBottom !== autoScroll) {
      setAutoScroll(isAtBottom)
    }
  }, [autoScroll])

  const toggleExpand = (id: number) => {
    setExpandedItems(prev => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }

  const toggleType = (type: string) => {
    setSelectedTypes(prev => {
      const next = new Set(prev)
      if (next.has(type)) {
        next.delete(type)
      } else {
        next.add(type)
      }
      return next
    })
  }

  const copyToClipboard = async (content: string) => {
    try {
      await navigator.clipboard.writeText(content)
    } catch (err) {
      console.error('Failed to copy:', err)
    }
  }

  const formatTimestamp = (ts: string) => {
    return new Date(ts).toLocaleTimeString('zh-CN', { 
      hour: '2-digit', 
      minute: '2-digit', 
      second: '2-digit',
      fractionalSecondDigits: 3
    })
  }

  const getEventColor = (type: string) => {
    return eventTypeColors[type] || defaultColor
  }

  // 获取显示内容（优先使用 raw）
  const getDisplayContent = (event: RawEvent): string => {
    if (event.raw) {
      try {
        const parsed = JSON.parse(event.raw)
        return JSON.stringify(parsed, null, 2)
      } catch {
        return event.raw
      }
    }
    return JSON.stringify(event.payload, null, 2)
  }

  // 获取预览文本
  const getPreview = (event: RawEvent): string => {
    const rawType = getRawType(event)
    
    if (event.raw) {
      try {
        const parsed = JSON.parse(event.raw)
        switch (rawType) {
          case 'system':
            return `v${parsed.qwen_code_version || '?'} | ${parsed.model || 'unknown'} | ${parsed.permission_mode || ''}`
          case 'assistant':
            const content = parsed.message?.content?.[0]?.text
            return content?.substring(0, 80) || ''
          case 'result':
            return `${parsed.subtype || ''} | ${parsed.duration_ms ? `${(parsed.duration_ms / 1000).toFixed(1)}s` : ''} | tokens: ${parsed.usage?.total_tokens || '?'}`
          default:
            break
        }
      } catch {}
    }
    
    const payload = event.payload
    if (!payload) return ''
    
    if (payload.content) return payload.content.substring(0, 80)
    if (payload.result) return payload.result.substring(0, 80)
    if (payload.message) return payload.message.substring(0, 80)
    
    return ''
  }

  return (
    <>
      {/* Solarized Light 护眼配色 */}
      <div className="h-full flex flex-col bg-[#fdf6e3] text-[#657b83] font-mono text-sm">
        {/* 工具栏 */}
        <div className="flex-shrink-0 flex items-center gap-2 px-3 py-2 bg-[#eee8d5] border-b border-[#93a1a1]/30">
          {/* 搜索框 */}
          <div className="flex-1 relative">
            <Search className="absolute left-2 top-1/2 -translate-y-1/2 w-4 h-4 text-[#93a1a1]" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="搜索..."
              className="w-full pl-8 pr-3 py-1.5 bg-[#fdf6e3] border border-[#93a1a1]/40 rounded text-sm text-[#657b83] placeholder-[#93a1a1] focus:outline-none focus:ring-1 focus:ring-[#268bd2]/50"
            />
          </div>

          {/* 过滤按钮 */}
          <button
            onClick={() => setShowFilter(!showFilter)}
            className={`flex items-center gap-1 px-2 py-1.5 rounded text-xs ${
              selectedTypes.size > 0 
                ? 'bg-[#268bd2] text-white' 
                : 'bg-[#fdf6e3] text-[#657b83] border border-[#93a1a1]/40 hover:bg-[#eee8d5]'
            }`}
          >
            <Filter className="w-3.5 h-3.5" />
            {selectedTypes.size > 0 && <span>{selectedTypes.size}</span>}
          </button>

          {/* 自动滚动按钮 */}
          <button
            onClick={() => setAutoScroll(!autoScroll)}
            className={`flex items-center gap-1 px-2 py-1.5 rounded text-xs ${
              autoScroll 
                ? 'bg-[#859900] text-white' 
                : 'bg-[#fdf6e3] text-[#657b83] border border-[#93a1a1]/40 hover:bg-[#eee8d5]'
            }`}
            title={autoScroll ? '自动滚动开启' : '自动滚动关闭'}
          >
            {autoScroll ? <ArrowDown className="w-3.5 h-3.5" /> : <Pause className="w-3.5 h-3.5" />}
          </button>

          {/* 事件计数 */}
          <span className="text-xs text-[#93a1a1]">
            {filteredEvents.length}/{rawEvents.length}
          </span>

          {/* 实时指示器 */}
          {isStreaming && (
            <span className="flex items-center gap-1 text-xs text-[#859900]">
              <span className="w-2 h-2 bg-[#859900] rounded-full animate-pulse" />
              LIVE
            </span>
          )}
        </div>

        {/* 类型过滤器 */}
        {showFilter && (
          <div className="flex-shrink-0 px-3 py-2 bg-[#eee8d5] border-b border-[#93a1a1]/30">
            <div className="flex flex-wrap gap-1">
              {eventTypes.map(type => {
                const color = getEventColor(type)
                const isSelected = selectedTypes.has(type)
                return (
                  <button
                    key={type}
                    onClick={() => toggleType(type)}
                    className={`px-2 py-0.5 rounded text-xs border transition-all ${
                      isSelected
                        ? `${color.bg} ${color.text} ${color.border}`
                        : 'bg-[#fdf6e3] text-[#93a1a1] border-[#93a1a1]/40 hover:bg-[#eee8d5]'
                    }`}
                  >
                    {type}
                  </button>
                )
              })}
              {selectedTypes.size > 0 && (
                <button
                  onClick={() => setSelectedTypes(new Set())}
                  className="px-2 py-0.5 rounded text-xs text-[#dc322f] hover:text-[#cb4b16]"
                >
                  清除
                </button>
              )}
            </div>
          </div>
        )}

        {/* 事件列表 */}
        <div 
          ref={containerRef}
          onScroll={handleScroll}
          className="flex-1 overflow-auto"
        >
          {filteredEvents.length === 0 ? (
            <div className="flex items-center justify-center h-full text-[#93a1a1]">
              {rawEvents.length === 0 ? '暂无 Agent 原始输出' : '没有匹配的事件'}
            </div>
          ) : (
            <div className="divide-y divide-gray-800">
              {filteredEvents.map((event) => {
                const isExpanded = expandedItems.has(event.id)
                const rawType = getRawType(event)
                const color = getEventColor(rawType)
                const displayContent = getDisplayContent(event)
                
                return (
                  <div key={event.id} className="group">
                    {/* 事件头部 */}
                    <div 
                      className="flex items-center gap-2 px-3 py-2 cursor-pointer hover:bg-[#eee8d5]"
                      onClick={() => toggleExpand(event.id)}
                    >
                      {/* 展开/折叠图标 */}
                      <span className="text-[#93a1a1]">
                        {isExpanded 
                          ? <ChevronDown className="w-4 h-4" /> 
                          : <ChevronRight className="w-4 h-4" />
                        }
                      </span>

                      {/* 序号 */}
                      <span className="text-gray-600 w-6 text-right text-xs">
                        #{event.seq}
                      </span>

                      {/* 类型标签 */}
                      <span 
                        className="px-2 py-0.5 rounded text-xs font-medium"
                        style={{ 
                          backgroundColor: `${color.badge}20`,
                          color: color.badge,
                          border: `1px solid ${color.badge}40`
                        }}
                      >
                        {rawType}
                      </span>

                      {/* 预览内容 */}
                      <span className="flex-1 truncate text-[#839496] text-xs">
                        {getPreview(event)}
                      </span>

                      {/* 时间戳 */}
                      <span className="text-gray-600 text-xs">
                        {formatTimestamp(event.timestamp)}
                      </span>

                      {/* 放大按钮 */}
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          setMaximizedEvent(event)
                        }}
                        className="p-1 rounded hover:bg-[#eee8d5] text-[#93a1a1] hover:text-[#657b83] opacity-0 group-hover:opacity-100 transition-opacity"
                        title="放大查看"
                      >
                        <Maximize2 className="w-3.5 h-3.5" />
                      </button>

                      {/* 复制按钮 */}
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          copyToClipboard(displayContent)
                          setCopiedId(event.id)
                          setTimeout(() => setCopiedId(null), 2000)
                        }}
                        className="p-1 rounded hover:bg-[#eee8d5] text-[#93a1a1] hover:text-[#657b83] opacity-0 group-hover:opacity-100 transition-opacity"
                        title="复制 JSON"
                      >
                        {copiedId === event.id 
                          ? <Check className="w-3.5 h-3.5 text-green-400" /> 
                          : <Copy className="w-3.5 h-3.5" />
                        }
                      </button>
                    </div>

                    {/* 展开的 JSON 内容 */}
                    {isExpanded && (
                      <div className="px-3 pb-3">
                        <SyntaxHighlighter
                          language="json"
                          style={customStyle}
                          wrapLongLines
                          customStyle={{
                            margin: 0,
                            borderRadius: '6px',
                            border: `1px solid ${color.badge}30`,
                          }}
                        >
                          {displayContent}
                        </SyntaxHighlighter>
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          )}
        </div>
      </div>

      {/* 放大浮窗 */}
      {maximizedEvent && (
        <div className="fixed inset-0 z-50 bg-black/30 flex items-center justify-center p-4">
          <div className="bg-[#fdf6e3] rounded-lg w-full max-w-5xl max-h-[90vh] flex flex-col shadow-2xl border border-[#93a1a1]/40">
            {/* 浮窗头部 */}
            <div className="flex items-center justify-between px-4 py-3 border-b border-[#93a1a1]/30 bg-[#eee8d5] rounded-t-lg">
              <div className="flex items-center gap-3">
                <span 
                  className="px-2 py-1 rounded text-sm font-medium"
                  style={{ 
                    backgroundColor: `${getEventColor(getRawType(maximizedEvent)).badge}20`,
                    color: getEventColor(getRawType(maximizedEvent)).badge,
                    border: `1px solid ${getEventColor(getRawType(maximizedEvent)).badge}40`
                  }}
                >
                  {getRawType(maximizedEvent)}
                </span>
                <span className="text-[#93a1a1] text-sm">
                  #{maximizedEvent.seq} · {formatTimestamp(maximizedEvent.timestamp)}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => {
                    copyToClipboard(getDisplayContent(maximizedEvent))
                    setCopiedId(maximizedEvent.id)
                    setTimeout(() => setCopiedId(null), 2000)
                  }}
                  className="flex items-center gap-1 px-3 py-1.5 rounded bg-[#eee8d5] text-[#657b83] border border-[#93a1a1]/40 hover:bg-[#fdf6e3] text-sm"
                >
                  {copiedId === maximizedEvent.id ? (
                    <>
                      <Check className="w-4 h-4 text-[#859900]" />
                      已复制
                    </>
                  ) : (
                    <>
                      <Copy className="w-4 h-4" />
                      复制
                    </>
                  )}
                </button>
                <button
                  onClick={() => setMaximizedEvent(null)}
                  className="p-1.5 rounded hover:bg-[#fdf6e3] text-[#93a1a1] hover:text-[#657b83]"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>
            </div>
            
            {/* 浮窗内容 */}
            <div className="flex-1 overflow-auto p-4">
              <SyntaxHighlighter
                language="json"
                style={customStyle}
                wrapLongLines
                showLineNumbers
                customStyle={{
                  margin: 0,
                  borderRadius: '6px',
                  fontSize: '13px',
                  lineHeight: '1.5',
                }}
              >
                {getDisplayContent(maximizedEvent)}
              </SyntaxHighlighter>
            </div>
          </div>
        </div>
      )}
    </>
  )
}
