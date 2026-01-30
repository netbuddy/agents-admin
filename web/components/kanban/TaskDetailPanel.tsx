'use client'

import { useEffect, useState, useRef } from 'react'
import { X, Clock, CheckCircle, XCircle, RefreshCw, Play, Square, ExternalLink, User, Server, LayoutList, Bug } from 'lucide-react'
import AgentOutput from '../agent-output/AgentOutput'
import DebugPanel from '../agent-output/DebugPanel'

interface Task {
  id: string
  name: string
  status: string
  spec: any
  created_at: string
  updated_at: string
}

interface Run {
  id: string
  task_id: string
  status: string
  node_id: string
  started_at: string | null
  finished_at: string | null
  error: string | null
  created_at: string
}

interface Event {
  id: number
  run_id: string
  seq: number
  type: string
  timestamp: string
  payload: any
}

interface TaskDetailPanelProps {
  task: Task
  onClose: () => void
  onStartRun: () => Promise<void>
  onRefresh: () => void
}

const statusConfig: Record<string, { icon: React.ReactNode; color: string; bgColor: string; label: string }> = {
  pending: {
    icon: <Clock className="w-4 h-4" />,
    color: 'text-gray-600',
    bgColor: 'bg-gray-100',
    label: '待处理'
  },
  running: {
    icon: <RefreshCw className="w-4 h-4 animate-spin" />,
    color: 'text-blue-600',
    bgColor: 'bg-blue-100',
    label: '运行中'
  },
  completed: {
    icon: <CheckCircle className="w-4 h-4" />,
    color: 'text-green-600',
    bgColor: 'bg-green-100',
    label: '已完成'
  },
  failed: {
    icon: <XCircle className="w-4 h-4" />,
    color: 'text-red-600',
    bgColor: 'bg-red-100',
    label: '失败'
  }
}

type ViewMode = 'formatted' | 'debug'

export default function TaskDetailPanel({ task, onClose, onStartRun, onRefresh }: TaskDetailPanelProps) {
  const [runs, setRuns] = useState<Run[]>([])
  const [selectedRun, setSelectedRun] = useState<Run | null>(null)
  const [events, setEvents] = useState<Event[]>([])
  const [loading, setLoading] = useState(true)
  const [wsConnected, setWsConnected] = useState(false)
  const [viewMode, setViewMode] = useState<ViewMode>('formatted')
  const [, setTick] = useState(0) // 用于强制刷新耗时显示
  const wsRef = useRef<WebSocket | null>(null)

  // 运行中任务每秒刷新耗时显示
  useEffect(() => {
    if (selectedRun?.status === 'running') {
      const timer = setInterval(() => {
        setTick(t => t + 1)
      }, 1000)
      return () => clearInterval(timer)
    }
  }, [selectedRun?.status])

  const agentType = task.spec?.agent?.type || 'unknown'
  const accountId = task.spec?.agent?.account_id
  const prompt = task.spec?.prompt || ''
  const status = statusConfig[task.status] || statusConfig.pending

  // 获取 Run 列表
  const fetchRuns = async (forceSelectFirst = false) => {
    try {
      const res = await fetch(`/api/v1/tasks/${task.id}/runs`)
      if (res.ok) {
        const data = await res.json()
        const runsList = data.runs || []
        setRuns(runsList)
        
        if (runsList.length > 0) {
          if (forceSelectFirst) {
            // 强制选择第一个（任务切换时）
            setSelectedRun(runsList[0])
          } else {
            // 更新当前选中的 run 状态
            setSelectedRun(prev => {
              if (prev) {
                const updatedRun = runsList.find((r: Run) => r.id === prev.id)
                return updatedRun || runsList[0]
              }
              return runsList[0]
            })
          }
        } else {
          setSelectedRun(null)
        }
      }
    } catch (err) {
      console.error('Failed to fetch runs:', err)
    } finally {
      setLoading(false)
    }
  }

  // 获取事件列表
  const fetchEvents = async (runId: string) => {
    try {
      const res = await fetch(`/api/v1/runs/${runId}/events?limit=1000`)
      if (res.ok) {
        const data = await res.json()
        setEvents(data.events || [])
      }
    } catch (err) {
      console.error('Failed to fetch events:', err)
    }
  }

  // 建立 WebSocket 连接
  const connectWebSocket = (runId: string) => {
    if (wsRef.current) {
      wsRef.current.close()
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const ws = new WebSocket(`${protocol}//${window.location.host}/api/v1/runs/${runId}/stream`)
    
    ws.onopen = () => {
      setWsConnected(true)
    }

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        if (data.type === 'event') {
          setEvents(prev => [...prev, data.event])
        }
      } catch (err) {
        console.error('Failed to parse WS message:', err)
      }
    }

    ws.onclose = () => {
      setWsConnected(false)
    }

    wsRef.current = ws
  }

  // 任务切换时重置所有状态
  useEffect(() => {
    // 清空旧数据
    setRuns([])
    setSelectedRun(null)
    setEvents([])
    setLoading(true)
    setWsConnected(false)
    wsRef.current?.close()
    
    // 获取新任务的数据（强制选择第一个 run）
    fetchRuns(true)
    
    // 定期刷新 Run 列表（不强制切换）
    const interval = setInterval(() => fetchRuns(false), 5000)
    return () => {
      clearInterval(interval)
      wsRef.current?.close()
    }
  }, [task.id])

  useEffect(() => {
    if (selectedRun) {
      fetchEvents(selectedRun.id)
      if (selectedRun.status === 'running') {
        connectWebSocket(selectedRun.id)
      } else {
        wsRef.current?.close()
        setWsConnected(false)
      }
    }
  }, [selectedRun?.id, selectedRun?.status])

  const handleStartRun = async () => {
    await onStartRun()
    // 创建新 run 后，强制选择第一个（最新的）
    fetchRuns(true)
  }

  const handleCancelRun = async () => {
    if (!selectedRun) return
    try {
      await fetch(`/api/v1/runs/${selectedRun.id}/cancel`, { method: 'POST' })
      fetchRuns()
      onRefresh()
    } catch (err) {
      console.error('Failed to cancel run:', err)
    }
  }

  const formatDuration = (run: Run | null) => {
    if (!run) return '-'
    
    // 如果任务还没开始（queued 状态），显示等待中
    if (run.status === 'queued') {
      return '等待中'
    }
    
    // 使用 started_at，如果没有则使用 created_at
    const startTimeStr = run.started_at || run.created_at
    if (!startTimeStr) return '-'
    
    const startTime = new Date(startTimeStr).getTime()
    const createdTime = new Date(run.created_at).getTime()
    
    // 如果任务正在运行，使用当前时间计算
    if (run.status === 'running') {
      const now = Date.now()
      
      // 验证时间合理性：started_at 不应该早于 created_at 太多
      // 如果 started_at 比 created_at 早超过 1 分钟，使用 created_at
      let effectiveStart = startTime
      if (startTime < createdTime - 60000) {
        effectiveStart = createdTime
      }
      
      // 如果是未来时间，显示刚开始
      if (effectiveStart > now) {
        return '0秒'
      }
      
      const seconds = Math.floor((now - effectiveStart) / 1000)
      if (seconds < 0) return '0秒'
      if (seconds < 60) return `${seconds}秒`
      const minutes = Math.floor(seconds / 60)
      const remainingSeconds = seconds % 60
      return `${minutes}分${remainingSeconds}秒`
    }
    
    // 任务已完成，使用 finished_at
    if (run.finished_at) {
      const endTime = new Date(run.finished_at).getTime()
      const seconds = Math.floor((endTime - startTime) / 1000)
      if (seconds < 0) return '-'
      if (seconds < 60) return `${seconds}秒`
      const minutes = Math.floor(seconds / 60)
      const remainingSeconds = seconds % 60
      return `${minutes}分${remainingSeconds}秒`
    }
    
    return '-'
  }

  return (
    <div className="w-[480px] bg-white border-l border-gray-200 flex flex-col h-full overflow-hidden">
      {/* 头部 */}
      <div className="flex-shrink-0 border-b border-gray-200">
        <div className="px-4 py-3 flex items-center justify-between">
          <h2 className="font-semibold text-gray-900 truncate flex-1">任务详情</h2>
          <button
            onClick={onClose}
            className="p-1 hover:bg-gray-100 rounded text-gray-500"
          >
            <X className="w-5 h-5" />
          </button>
        </div>
      </div>

      {/* 任务信息 */}
      <div className="flex-shrink-0 p-4 border-b border-gray-200 bg-gray-50">
        <h3 className="font-medium text-gray-900 mb-2">{task.name}</h3>
        
        <div className="grid grid-cols-2 gap-2 text-sm mb-3">
          <div className="flex items-center gap-2">
            <span className={`flex items-center gap-1 px-2 py-1 rounded ${status.bgColor} ${status.color}`}>
              {status.icon}
              <span className="font-medium">{status.label}</span>
            </span>
          </div>
          <div className="flex items-center gap-2 text-gray-600">
            <span className="px-2 py-1 rounded bg-gray-100">{agentType}</span>
          </div>
        </div>

        {accountId && (
          <div className="flex items-center gap-4 text-xs text-gray-500 mb-3">
            <span className="flex items-center gap-1">
              <User className="w-3 h-3" />
              {accountId}
            </span>
          </div>
        )}

        {/* 提示词 */}
        <div className="bg-white rounded-lg border border-gray-200 p-3">
          <label className="block text-xs text-gray-500 mb-1">任务提示词</label>
          <p className="text-sm text-gray-700 whitespace-pre-wrap line-clamp-4">
            {prompt || '无提示词'}
          </p>
        </div>
      </div>

      {/* Run 选择器 */}
      {runs.length > 0 && (
        <div className="flex-shrink-0 px-4 py-2 border-b border-gray-200 bg-gray-50">
          <div className="flex items-center gap-2">
            <label className="text-xs text-gray-500">运行记录:</label>
            <select
              value={selectedRun?.id || ''}
              onChange={(e) => {
                const run = runs.find(r => r.id === e.target.value)
                if (run) setSelectedRun(run)
              }}
              className="flex-1 text-sm border rounded px-2 py-1 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              {runs.map((run, idx) => (
                <option key={run.id} value={run.id}>
                  #{runs.length - idx} - {run.status} - {new Date(run.created_at).toLocaleString('zh-CN')}
                </option>
              ))}
            </select>
            {wsConnected && (
              <span className="flex items-center gap-1 text-xs text-green-600">
                <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
                实时
              </span>
            )}
          </div>
          
          {selectedRun && (
            <div className="mt-2 flex items-center justify-between">
              <div className="flex items-center gap-4 text-xs text-gray-500">
                <span>耗时: {formatDuration(selectedRun)}</span>
                {selectedRun.node_id && <span>节点: {selectedRun.node_id}</span>}
              </div>
              
              {/* 视图切换 Tab */}
              <div className="flex items-center bg-gray-200 rounded-lg p-0.5">
                <button
                  onClick={() => setViewMode('formatted')}
                  className={`flex items-center gap-1 px-2 py-1 rounded text-xs font-medium transition-colors ${
                    viewMode === 'formatted'
                      ? 'bg-white text-gray-700 shadow-sm'
                      : 'text-gray-500 hover:text-gray-700'
                  }`}
                  title="格式化视图"
                >
                  <LayoutList className="w-3.5 h-3.5" />
                  格式化
                </button>
                <button
                  onClick={() => setViewMode('debug')}
                  className={`flex items-center gap-1 px-2 py-1 rounded text-xs font-medium transition-colors ${
                    viewMode === 'debug'
                      ? 'bg-white text-gray-700 shadow-sm'
                      : 'text-gray-500 hover:text-gray-700'
                  }`}
                  title="调试视图 - 显示原始事件"
                >
                  <Bug className="w-3.5 h-3.5" />
                  调试
                </button>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Agent 输出 */}
      <div className="flex-1 overflow-hidden">
        {loading ? (
          <div className="flex items-center justify-center h-full">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
          </div>
        ) : selectedRun ? (
          viewMode === 'formatted' ? (
            <AgentOutput 
              events={events} 
              isStreaming={wsConnected && selectedRun.status === 'running'}
              error={selectedRun.error}
            />
          ) : (
            <DebugPanel 
              events={events}
              isStreaming={wsConnected && selectedRun.status === 'running'}
            />
          )
        ) : (
          <div className="flex flex-col items-center justify-center h-full text-gray-500 p-8">
            <Clock className="w-12 h-12 mb-4 text-gray-300" />
            <p className="text-sm mb-4">暂无运行记录</p>
            {task.status === 'pending' && (
              <button
                onClick={handleStartRun}
                className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
              >
                <Play className="w-4 h-4" />
                开始执行
              </button>
            )}
          </div>
        )}
      </div>

      {/* 底部操作栏 */}
      <div className="flex-shrink-0 px-4 py-3 border-t border-gray-200 bg-gray-50">
        <div className="flex items-center justify-between gap-2">
          <a
            href={`/tasks/${task.id}`}
            className="flex items-center gap-1 text-sm text-blue-600 hover:underline"
          >
            <ExternalLink className="w-4 h-4" />
            在新页面打开
          </a>
          <div className="flex gap-2">
            {selectedRun?.status === 'running' && (
              <button
                onClick={handleCancelRun}
                className="flex items-center gap-2 px-3 py-1.5 border border-red-300 text-red-600 rounded hover:bg-red-50"
              >
                <Square className="w-4 h-4" />
                取消
              </button>
            )}
            {(task.status === 'pending' || task.status === 'failed' || task.status === 'completed') && (
              <button
                onClick={handleStartRun}
                className="flex items-center gap-2 px-3 py-1.5 bg-blue-600 text-white rounded hover:bg-blue-700"
              >
                <Play className="w-4 h-4" />
                {task.status === 'pending' ? '执行' : '重新执行'}
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
