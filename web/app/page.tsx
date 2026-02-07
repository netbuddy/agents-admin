'use client'

import { useEffect, useState, useCallback } from 'react'
import { Plus, Search, X } from 'lucide-react'
import { AdminLayout } from '@/components/layout'
import { KanbanColumn, TaskDetailPanel } from '@/components/kanban'
import CreateTaskModal from '@/components/CreateTaskModal'

interface Task {
  id: string
  name: string
  status: string
  spec: any
  created_at: string
  updated_at: string
  parent_id?: string
}

const columns = [
  { id: 'pending', title: '待处理', color: 'bg-gray-100' },
  { id: 'running', title: '运行中', color: 'bg-blue-100' },
  { id: 'completed', title: '已完成', color: 'bg-green-100' },
  { id: 'failed', title: '失败', color: 'bg-red-100' },
]

const TIME_RANGES = [
  { label: '全部', value: '' },
  { label: '今天', value: 'today' },
  { label: '近7天', value: '7d' },
  { label: '近30天', value: '30d' },
]

function getSinceDate(range: string): string {
  if (!range) return ''
  const now = new Date()
  switch (range) {
    case 'today':
      now.setHours(0, 0, 0, 0)
      return now.toISOString()
    case '7d':
      now.setDate(now.getDate() - 7)
      return now.toISOString()
    case '30d':
      now.setDate(now.getDate() - 30)
      return now.toISOString()
    default:
      return ''
  }
}

const PAGE_SIZE = 20

export default function Home() {
  const [tasks, setTasks] = useState<Task[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null)
  const [mobileFilter, setMobileFilter] = useState<string>('all')
  const [search, setSearch] = useState('')
  const [searchInput, setSearchInput] = useState('')
  const [timeRange, setTimeRange] = useState('')
  const [columnLimits, setColumnLimits] = useState<Record<string, number>>({})

  const fetchTasks = useCallback(async () => {
    try {
      const params = new URLSearchParams({ limit: '100' })
      if (search) params.set('search', search)
      const since = getSinceDate(timeRange)
      if (since) params.set('since', since)
      const res = await fetch(`/api/v1/tasks?${params}`)
      const data = await res.json()
      setTasks(data.tasks || [])
      setTotal(data.total ?? data.tasks?.length ?? 0)
    } catch (err) {
      console.error('Failed to fetch tasks:', err)
    } finally {
      setLoading(false)
    }
  }, [search, timeRange])

  useEffect(() => {
    fetchTasks()
    const interval = setInterval(fetchTasks, 5000)
    return () => clearInterval(interval)
  }, [fetchTasks])

  const getTasksByStatus = (status: string) => {
    return tasks.filter(t => t.status === status)
  }

  const getVisibleTasks = (status: string) => {
    const all = getTasksByStatus(status)
    const limit = columnLimits[status] || PAGE_SIZE
    return all.slice(0, limit)
  }

  const showMore = (status: string) => {
    setColumnLimits(prev => ({
      ...prev,
      [status]: (prev[status] || PAGE_SIZE) + PAGE_SIZE,
    }))
  }

  const handleSearch = () => {
    setSearch(searchInput)
    setColumnLimits({})
  }

  const clearSearch = () => {
    setSearchInput('')
    setSearch('')
    setColumnLimits({})
  }

  const filteredTasks = mobileFilter === 'all'
    ? tasks
    : tasks.filter(t => t.status === mobileFilter)

  const selectedTask = selectedTaskId ? tasks.find(t => t.id === selectedTaskId) : null

  const handleStartRun = async (taskId: string) => {
    try {
      await fetch(`/api/v1/tasks/${taskId}/runs`, { method: 'POST' })
      fetchTasks()
    } catch (err) {
      console.error('Failed to start run:', err)
    }
  }

  const handleStopRun = async (taskId: string) => {
    try {
      const res = await fetch(`/api/v1/tasks/${taskId}/runs`)
      if (res.ok) {
        const data = await res.json()
        const runs = data.runs || []
        const activeRun = runs.find((r: any) => r.status === 'running')
        if (activeRun) {
          await fetch(`/api/v1/runs/${activeRun.id}/cancel`, { method: 'POST' })
          fetchTasks()
        }
      }
    } catch (err) {
      console.error('Failed to stop run:', err)
    }
  }

  const handleDeleteTask = async (taskId: string) => {
    try {
      await fetch(`/api/v1/tasks/${taskId}`, { method: 'DELETE' })
      if (selectedTaskId === taskId) {
        setSelectedTaskId(null)
      }
      fetchTasks()
    } catch (err) {
      console.error('Failed to delete task:', err)
    }
  }

  return (
    <AdminLayout title="任务看板" onRefresh={fetchTasks} loading={loading}>
      <div className="flex flex-col md:flex-row h-[calc(100vh-120px)] md:h-[calc(100vh-120px)]">
        {/* 主看板区域 - 移动端详情面板打开时隐藏 */}
        <div className={`flex-1 overflow-hidden flex flex-col min-w-0 ${
          selectedTask ? 'hidden md:flex' : 'flex'
        }`}>
          {/* Actions bar */}
          <div className="mb-3 sm:mb-4 flex flex-col gap-2 flex-shrink-0">
            <div className="flex items-center justify-between">
              <p className="text-sm text-gray-500">
                共 {total} 个任务{search && <span>，搜索: &quot;{search}&quot;</span>}
              </p>
              <button
                onClick={() => setShowCreateModal(true)}
                className="flex items-center gap-2 px-3 py-2 sm:px-4 rounded-lg bg-blue-600 text-white hover:bg-blue-700 text-sm"
              >
                <Plus className="w-4 h-4" />
                <span className="hidden sm:inline">新建任务</span>
                <span className="sm:hidden">新建</span>
              </button>
            </div>
            <div className="flex items-center gap-2">
              <div className="relative flex-1 max-w-xs">
                <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                <input
                  type="text"
                  value={searchInput}
                  onChange={e => setSearchInput(e.target.value)}
                  onKeyDown={e => e.key === 'Enter' && handleSearch()}
                  placeholder="搜索任务..."
                  className="w-full pl-8 pr-8 py-1.5 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                {searchInput && (
                  <button onClick={clearSearch} className="absolute right-2 top-1/2 -translate-y-1/2">
                    <X className="w-3.5 h-3.5 text-gray-400 hover:text-gray-600" />
                  </button>
                )}
              </div>
              <div className="flex gap-1">
                {TIME_RANGES.map(r => (
                  <button
                    key={r.value}
                    onClick={() => { setTimeRange(r.value); setColumnLimits({}) }}
                    className={`px-2.5 py-1.5 text-xs rounded-lg transition-colors ${
                      timeRange === r.value
                        ? 'bg-gray-900 text-white'
                        : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
                    }`}
                  >
                    {r.label}
                  </button>
                ))}
              </div>
            </div>
          </div>

          {/* 移动端：Tab 筛选器 + 卡片列表 */}
          <div className="md:hidden flex-1 flex flex-col overflow-hidden">
            {/* 状态 Tab */}
            <div className="flex gap-1 mb-3 overflow-x-auto scrollbar-hide flex-shrink-0 -mx-1 px-1">
              <button
                onClick={() => setMobileFilter('all')}
                className={`px-3 py-1.5 rounded-full text-xs font-medium flex-shrink-0 transition-colors ${
                  mobileFilter === 'all'
                    ? 'bg-gray-900 text-white'
                    : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
                }`}
              >
                全部 ({tasks.length})
              </button>
              {columns.map(col => {
                const count = getTasksByStatus(col.id).length
                return (
                  <button
                    key={col.id}
                    onClick={() => setMobileFilter(col.id)}
                    className={`px-3 py-1.5 rounded-full text-xs font-medium flex-shrink-0 transition-colors ${
                      mobileFilter === col.id
                        ? 'bg-gray-900 text-white'
                        : `${col.color} text-gray-700 hover:opacity-80`
                    }`}
                  >
                    {col.title} ({count})
                  </button>
                )
              })}
            </div>

            {/* 移动端卡片列表 */}
            <div className="flex-1 overflow-y-auto touch-scroll space-y-3 pb-4">
              {loading && tasks.length === 0 ? (
                <div className="flex items-center justify-center h-64">
                  <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
                </div>
              ) : filteredTasks.length === 0 ? (
                <div className="text-center py-8 text-sm text-gray-400">暂无任务</div>
              ) : (
                filteredTasks.map(task => (
                  <div key={task.id}>
                    <KanbanColumn
                      title=""
                      status={task.status}
                      tasks={[task]}
                      color="bg-transparent"
                      selectedTaskId={selectedTaskId}
                      onSelectTask={(taskId) => setSelectedTaskId(taskId)}
                      onStartRun={handleStartRun}
                      onStopRun={handleStopRun}
                      onDeleteTask={handleDeleteTask}
                      compact
                    />
                  </div>
                ))
              )}
            </div>
          </div>

          {/* 平板和桌面端：Kanban 列布局 */}
          <div className="hidden md:block flex-1 overflow-hidden">
            {loading && tasks.length === 0 ? (
              <div className="flex items-center justify-center h-64">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
              </div>
            ) : (
              <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 lg:gap-4 h-full overflow-auto pb-4 touch-scroll">
                {columns.map(col => {
                  const allTasks = getTasksByStatus(col.id)
                  const visible = getVisibleTasks(col.id)
                  const hasMore = visible.length < allTasks.length
                  return (
                    <div key={col.id} className="flex flex-col min-h-0">
                      <KanbanColumn
                        title={col.title}
                        status={col.id}
                        tasks={visible}
                        color={col.color}
                        selectedTaskId={selectedTaskId}
                        onSelectTask={(taskId) => setSelectedTaskId(taskId)}
                        onStartRun={handleStartRun}
                        onStopRun={handleStopRun}
                        onDeleteTask={handleDeleteTask}
                        totalCount={allTasks.length}
                      />
                      {hasMore && (
                        <button
                          onClick={() => showMore(col.id)}
                          className="mt-1 w-full py-1.5 text-xs text-blue-600 hover:bg-blue-50 rounded-lg transition-colors"
                        >
                          加载更多（还有 {allTasks.length - visible.length} 条）
                        </button>
                      )}
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        </div>

        {/* 侧边详情面板 - 移动端全屏，桌面端侧边 */}
        {selectedTask && (
          <TaskDetailPanel
            task={selectedTask}
            onClose={() => setSelectedTaskId(null)}
            onStartRun={() => handleStartRun(selectedTask.id)}
            onRefresh={fetchTasks}
          />
        )}
      </div>

      {showCreateModal && (
        <CreateTaskModal
          onClose={() => setShowCreateModal(false)}
          onCreated={() => {
            setShowCreateModal(false)
            fetchTasks()
          }}
        />
      )}
    </AdminLayout>
  )
}
