'use client'

import { useEffect, useState, useRef } from 'react'
import { useRouter } from 'next/navigation'
import { ArrowLeft, Play, Square, RefreshCw, Terminal, FileText, Clock, CheckCircle, XCircle, AlertCircle } from 'lucide-react'
import Link from 'next/link'
import { useTranslation } from 'react-i18next'
import { useFormatDate } from '@/i18n/useFormatDate'

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

const statusIcons: Record<string, React.ReactNode> = {
  queued: <Clock className="w-4 h-4 text-gray-500" />,
  running: <RefreshCw className="w-4 h-4 text-blue-500 animate-spin" />,
  done: <CheckCircle className="w-4 h-4 text-green-500" />,
  failed: <XCircle className="w-4 h-4 text-red-500" />,
  cancelled: <AlertCircle className="w-4 h-4 text-orange-500" />,
}

const eventTypeColors: Record<string, string> = {
  run_started: 'bg-green-100 text-green-800',
  run_completed: 'bg-green-100 text-green-800',
  message: 'bg-blue-100 text-blue-800',
  tool_use_start: 'bg-purple-100 text-purple-800',
  tool_result: 'bg-purple-100 text-purple-800',
  file_read: 'bg-yellow-100 text-yellow-800',
  file_write: 'bg-yellow-100 text-yellow-800',
  command: 'bg-gray-100 text-gray-800',
  command_output: 'bg-gray-100 text-gray-800',
  error: 'bg-red-100 text-red-800',
}

export default function TaskDetailClient({ taskId }: { taskId: string }) {
  const { t } = useTranslation('tasks')
  const { formatDateTime } = useFormatDate()
  const router = useRouter()
  
  const [task, setTask] = useState<Task | null>(null)
  const [runs, setRuns] = useState<Run[]>([])
  const [selectedRun, setSelectedRun] = useState<Run | null>(null)
  const [events, setEvents] = useState<Event[]>([])
  const [loading, setLoading] = useState(true)
  const [wsConnected, setWsConnected] = useState(false)
  const [showRunsSidebar, setShowRunsSidebar] = useState(false)
  const [showConfigPanel, setShowConfigPanel] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)
  const eventsEndRef = useRef<HTMLDivElement>(null)

  const fetchTask = async () => {
    try {
      const res = await fetch(`/api/v1/tasks/${taskId}`)
      if (res.ok) {
        setTask(await res.json())
      }
    } catch (err) {
      console.error('Failed to fetch task:', err)
    }
  }

  const fetchRuns = async () => {
    try {
      const res = await fetch(`/api/v1/tasks/${taskId}/runs`)
      if (res.ok) {
        const data = await res.json()
        setRuns(data.runs || [])
        if (data.runs?.length > 0 && !selectedRun) {
          setSelectedRun(data.runs[0])
        }
      }
    } catch (err) {
      console.error('Failed to fetch runs:', err)
    } finally {
      setLoading(false)
    }
  }

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

  const connectWebSocket = (runId: string) => {
    if (wsRef.current) {
      wsRef.current.close()
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const ws = new WebSocket(`${protocol}//${window.location.host}/ws/runs/${runId}/events`)
    
    ws.onopen = () => {
      setWsConnected(true)
    }

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        if (data.type === 'event' && data.data) {
          setEvents(prev => [...prev, data.data])
        } else if (data.type === 'status') {
          console.log('Run status update:', data.data)
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

  useEffect(() => {
    fetchTask()
    fetchRuns()
    const interval = setInterval(fetchRuns, 5000)
    return () => {
      clearInterval(interval)
      wsRef.current?.close()
    }
  }, [taskId])

  useEffect(() => {
    if (selectedRun) {
      fetchEvents(selectedRun.id)
      if (selectedRun.status === 'running') {
        connectWebSocket(selectedRun.id)
      }
    }
  }, [selectedRun?.id])

  useEffect(() => {
    eventsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [events])

  const startRun = async () => {
    try {
      const res = await fetch(`/api/v1/tasks/${taskId}/runs`, { method: 'POST' })
      if (res.ok) {
        const newRun = await res.json()
        setRuns(prev => [newRun, ...prev])
        setSelectedRun(newRun)
      }
    } catch (err) {
      console.error('Failed to start run:', err)
    }
  }

  const cancelRun = async (runId: string) => {
    try {
      await fetch(`/api/v1/runs/${runId}/cancel`, { method: 'POST' })
      fetchRuns()
    } catch (err) {
      console.error('Failed to cancel run:', err)
    }
  }

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
      </div>
    )
  }

  if (!task) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <h2 className="text-xl font-semibold mb-2">{t('detail.notFound')}</h2>
          <Link href="/" className="text-blue-600 hover:underline">{t('detail.goHome')}</Link>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="bg-white border-b px-3 sm:px-6 py-3 sm:py-4">
        <div className="flex items-center gap-2 sm:gap-4">
          <button onClick={() => router.back()} className="p-2 hover:bg-gray-100 rounded flex-shrink-0">
            <ArrowLeft className="w-5 h-5" />
          </button>
          <div className="flex-1 min-w-0">
            <h1 className="text-base sm:text-xl font-semibold truncate">{task.name}</h1>
            <p className="text-xs sm:text-sm text-gray-500 truncate">
              {task.spec?.agent?.type || 'unknown'} · {t('detail.createdAt')} {formatDateTime(task.created_at)}
            </p>
          </div>
          <button
            onClick={startRun}
            className="flex items-center gap-1 sm:gap-2 px-3 sm:px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm flex-shrink-0"
          >
            <Play className="w-4 h-4" />
            <span className="hidden sm:inline">{t('detail.newRun')}</span>
            <span className="sm:hidden">Run</span>
          </button>
        </div>
        {/* 移动端操作栏 */}
        <div className="flex gap-2 mt-2 md:hidden">
          <button
            onClick={() => setShowRunsSidebar(!showRunsSidebar)}
            className="flex-1 flex items-center justify-center gap-1 px-3 py-2 bg-gray-100 rounded-lg text-sm"
          >
            <FileText className="w-4 h-4" />
            {t('detail.runHistory', { count: runs.length })}
          </button>
          <button
            onClick={() => setShowConfigPanel(!showConfigPanel)}
            className="flex-1 flex items-center justify-center gap-1 px-3 py-2 bg-gray-100 rounded-lg text-sm"
          >
            <Terminal className="w-4 h-4" />
            {t('detail.taskConfig')}
          </button>
        </div>
      </header>

      {/* 移动端运行记录抽屉 */}
      {showRunsSidebar && (
        <div className="md:hidden bg-white border-b max-h-[40vh] overflow-y-auto touch-scroll">
          <div className="divide-y">
            {runs.map(run => (
              <button
                key={run.id}
                onClick={() => { setSelectedRun(run); setShowRunsSidebar(false) }}
                className={`w-full p-3 text-left hover:bg-gray-50 ${
                  selectedRun?.id === run.id ? 'bg-blue-50 border-l-2 border-blue-600' : ''
                }`}
              >
                <div className="flex items-center gap-2 mb-1">
                  {statusIcons[run.status] || statusIcons.queued}
                  <span className="font-medium text-sm truncate">{run.id.slice(-8)}</span>
                </div>
                <div className="text-xs text-gray-500">
                  {formatDateTime(run.created_at)}
                </div>
                {run.status === 'running' && (
                  <button
                    onClick={(e) => { e.stopPropagation(); cancelRun(run.id) }}
                    className="mt-2 flex items-center gap-1 text-xs text-red-600 hover:text-red-700"
                  >
                    <Square className="w-3 h-3" />
                    {t('detail.cancelRun')}
                  </button>
                )}
              </button>
            ))}
            {runs.length === 0 && (
              <div className="p-4 text-sm text-gray-500 text-center">{t('detail.noRuns')}</div>
            )}
          </div>
        </div>
      )}

      {/* 移动端配置面板 */}
      {showConfigPanel && (
        <div className="md:hidden bg-white border-b max-h-[40vh] overflow-y-auto touch-scroll p-4 space-y-4">
          <div>
            <label className="text-xs text-gray-500 block mb-1">Prompt</label>
            <div className="bg-gray-50 p-3 rounded text-sm whitespace-pre-wrap">
              {task.spec?.prompt || '-'}
            </div>
          </div>
          <div>
            <label className="text-xs text-gray-500 block mb-1">Agent</label>
            <div className="bg-gray-50 p-3 rounded text-sm">
              <pre className="overflow-x-auto text-xs">
                {JSON.stringify(task.spec?.agent || {}, null, 2)}
              </pre>
            </div>
          </div>
        </div>
      )}

      <div className="flex h-[calc(100vh-73px)] md:h-[calc(100vh-73px)]">
        {/* 桌面端运行记录侧栏 */}
        <aside className="hidden md:block w-56 lg:w-64 bg-white border-r overflow-y-auto touch-scroll flex-shrink-0">
          <div className="p-4 border-b">
            <h2 className="font-semibold text-sm text-gray-500">{t('detail.runHistory', { count: runs.length })}</h2>
          </div>
          <div className="divide-y">
            {runs.map(run => (
              <button
                key={run.id}
                onClick={() => setSelectedRun(run)}
                className={`w-full p-3 text-left hover:bg-gray-50 ${
                  selectedRun?.id === run.id ? 'bg-blue-50 border-l-2 border-blue-600' : ''
                }`}
              >
                <div className="flex items-center gap-2 mb-1">
                  {statusIcons[run.status] || statusIcons.queued}
                  <span className="font-medium text-sm truncate">{run.id.slice(-8)}</span>
                </div>
                <div className="text-xs text-gray-500">
                  {formatDateTime(run.created_at)}
                </div>
                {run.status === 'running' && (
                  <button
                    onClick={(e) => { e.stopPropagation(); cancelRun(run.id) }}
                    className="mt-2 flex items-center gap-1 text-xs text-red-600 hover:text-red-700"
                  >
                    <Square className="w-3 h-3" />
                    {t('detail.cancelRun')}
                  </button>
                )}
              </button>
            ))}
            {runs.length === 0 && (
              <div className="p-4 text-sm text-gray-500 text-center">
                {t('detail.noRuns')}
              </div>
            )}
          </div>
        </aside>

        <main className="flex-1 flex flex-col overflow-hidden min-w-0">
          {selectedRun ? (
            <>
              <div className="bg-white border-b px-3 sm:px-4 py-2 sm:py-3 flex items-center justify-between flex-shrink-0">
                <div className="flex items-center gap-2 sm:gap-3 min-w-0">
                  {statusIcons[selectedRun.status]}
                  <span className="font-medium text-sm sm:text-base truncate">Run {selectedRun.id.slice(-8)}</span>
                  <span className={`text-xs px-2 py-0.5 rounded flex-shrink-0 ${
                    selectedRun.status === 'running' ? 'bg-blue-100 text-blue-800' :
                    selectedRun.status === 'done' ? 'bg-green-100 text-green-800' :
                    selectedRun.status === 'failed' ? 'bg-red-100 text-red-800' :
                    'bg-gray-100 text-gray-800'
                  }`}>
                    {selectedRun.status}
                  </span>
                  {wsConnected && (
                    <span className="hidden sm:flex items-center gap-1 text-xs text-green-600">
                      <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
                      {t('detail.liveConnection')}
                    </span>
                  )}
                </div>
                <div className="text-xs sm:text-sm text-gray-500 hidden sm:block">
                  {selectedRun.node_id && `${t('detail.executionNode')}: ${selectedRun.node_id}`}
                </div>
              </div>

              <div className="flex-1 overflow-y-auto p-2 sm:p-4 bg-gray-900 font-mono text-xs sm:text-sm touch-scroll">
                {events.length === 0 ? (
                  <div className="text-gray-500 text-center py-8">
                    {t('detail.waitingEvents')}
                  </div>
                ) : (
                  <div className="space-y-1 sm:space-y-2">
                    {events.map((event, idx) => (
                      <div key={idx} className="flex gap-1 sm:gap-3">
                        <span className="text-gray-500 w-10 sm:w-16 flex-shrink-0">
                          #{event.seq}
                        </span>
                        <span className={`px-1.5 sm:px-2 py-0.5 rounded text-xs flex-shrink-0 ${
                          eventTypeColors[event.type] || 'bg-gray-700 text-gray-300'
                        }`}>
                          {event.type}
                        </span>
                        <span className="text-gray-300 flex-1 break-all">
                          {event.type === 'message' && event.payload?.content}
                          {event.type === 'tool_use_start' && `${event.payload?.tool_name || event.payload?.tool}`}
                          {event.type === 'file_write' && event.payload?.path}
                          {event.type === 'command' && event.payload?.command}
                          {event.type === 'error' && (
                            <span className="text-red-400">{event.payload?.message}</span>
                          )}
                          {!['message', 'tool_use_start', 'file_write', 'command', 'error'].includes(event.type) && 
                            JSON.stringify(event.payload).slice(0, 100)}
                        </span>
                      </div>
                    ))}
                    <div ref={eventsEndRef} />
                  </div>
                )}
              </div>

              {selectedRun.error && (
                <div className="bg-red-50 border-t border-red-200 px-3 sm:px-4 py-3">
                  <div className="flex items-center gap-2 text-red-800">
                    <XCircle className="w-4 h-4" />
                    <span className="font-medium text-sm">{t('detail.error')}</span>
                  </div>
                  <p className="text-red-700 text-sm mt-1">{selectedRun.error}</p>
                </div>
              )}
            </>
          ) : (
            <div className="flex-1 flex items-center justify-center text-gray-500 text-sm">
              {t('detail.selectRun')}
            </div>
          )}
        </main>

        {/* 桌面端配置侧栏 */}
        <aside className="hidden lg:block w-72 xl:w-80 bg-white border-l overflow-y-auto touch-scroll flex-shrink-0">
          <div className="p-4 border-b">
            <h2 className="font-semibold text-sm text-gray-500">{t('detail.taskConfig')}</h2>
          </div>
          <div className="p-4 space-y-4">
            <div>
              <label className="text-xs text-gray-500 block mb-1">Prompt</label>
              <div className="bg-gray-50 p-3 rounded text-sm whitespace-pre-wrap">
                {task.spec?.prompt || '-'}
              </div>
            </div>
            <div>
              <label className="text-xs text-gray-500 block mb-1">Agent</label>
              <div className="bg-gray-50 p-3 rounded text-sm">
                <pre className="overflow-x-auto text-xs">
                  {JSON.stringify(task.spec?.agent || {}, null, 2)}
                </pre>
              </div>
            </div>
            {task.spec?.workspace && (
              <div>
                <label className="text-xs text-gray-500 block mb-1">Workspace</label>
                <div className="bg-gray-50 p-3 rounded text-sm">
                  <pre className="overflow-x-auto text-xs">
                    {JSON.stringify(task.spec.workspace, null, 2)}
                  </pre>
                </div>
              </div>
            )}
          </div>
        </aside>
      </div>
    </div>
  )
}
