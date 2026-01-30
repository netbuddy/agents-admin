'use client'

import { useEffect, useState } from 'react'
import { Plus, X } from 'lucide-react'
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

export default function Home() {
  const [tasks, setTasks] = useState<Task[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null)

  const fetchTasks = async () => {
    try {
      const res = await fetch('/api/v1/tasks?limit=100')
      const data = await res.json()
      setTasks(data.tasks || [])
    } catch (err) {
      console.error('Failed to fetch tasks:', err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchTasks()
    const interval = setInterval(fetchTasks, 5000)
    return () => clearInterval(interval)
  }, [])

  const getTasksByStatus = (status: string) => {
    return tasks.filter(t => t.status === status)
  }

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
    // 获取最新的 run 并取消
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
      <div className="flex h-[calc(100vh-120px)]">
        {/* 主看板区域 */}
        <div className={`flex-1 overflow-hidden transition-all duration-300 ${selectedTask ? 'mr-0' : ''}`}>
          {/* Actions bar */}
          <div className="mb-4 flex items-center justify-between">
            <p className="text-sm text-gray-500">
              共 {tasks.length} 个任务
            </p>
            <button
              onClick={() => setShowCreateModal(true)}
              className="flex items-center gap-2 px-4 py-2 rounded-lg bg-blue-600 text-white hover:bg-blue-700"
            >
              <Plus className="w-4 h-4" />
              新建任务
            </button>
          </div>

          {/* Kanban board */}
          {loading && tasks.length === 0 ? (
            <div className="flex items-center justify-center h-64">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
            </div>
          ) : (
            <div className="grid grid-cols-4 gap-4 h-[calc(100%-60px)] overflow-auto pb-4">
              {columns.map(col => (
                <KanbanColumn
                  key={col.id}
                  title={col.title}
                  status={col.id}
                  tasks={getTasksByStatus(col.id)}
                  color={col.color}
                  selectedTaskId={selectedTaskId}
                  onSelectTask={(taskId) => setSelectedTaskId(taskId)}
                  onStartRun={handleStartRun}
                  onStopRun={handleStopRun}
                  onDeleteTask={handleDeleteTask}
                />
              ))}
            </div>
          )}
        </div>

        {/* 侧边详情面板 */}
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
