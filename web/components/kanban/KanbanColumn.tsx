'use client'

import TaskCard from './TaskCard'

interface Task {
  id: string
  name: string
  status: string
  spec: any
  created_at: string
}

interface KanbanColumnProps {
  title: string
  status: string
  tasks: Task[]
  color: string
  selectedTaskId?: string | null
  onSelectTask: (taskId: string) => void
  onStartRun: (taskId: string) => Promise<void>
  onStopRun: (taskId: string) => Promise<void>
  onDeleteTask: (taskId: string) => Promise<void>
}

export default function KanbanColumn({
  title,
  status,
  tasks,
  color,
  selectedTaskId,
  onSelectTask,
  onStartRun,
  onStopRun,
  onDeleteTask
}: KanbanColumnProps) {
  return (
    <div className={`rounded-lg p-4 ${color} min-h-[400px] flex flex-col`}>
      <div className="flex items-center justify-between mb-4">
        <h2 className="font-semibold text-gray-700">{title}</h2>
        <span className="text-sm font-medium text-gray-500 bg-white/50 px-2 py-0.5 rounded">
          {tasks.length}
        </span>
      </div>
      
      <div className="flex-1 space-y-3 overflow-y-auto">
        {tasks.length === 0 ? (
          <div className="text-center py-8 text-sm text-gray-400">
            暂无任务
          </div>
        ) : (
          tasks.map(task => (
            <TaskCard
              key={task.id}
              task={task}
              isSelected={selectedTaskId === task.id}
              onSelect={() => onSelectTask(task.id)}
              onStartRun={() => onStartRun(task.id)}
              onStopRun={() => onStopRun(task.id)}
              onDelete={() => onDeleteTask(task.id)}
            />
          ))
        )}
      </div>
    </div>
  )
}
