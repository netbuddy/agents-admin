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
  compact?: boolean
  totalCount?: number
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
  onDeleteTask,
  compact = false,
  totalCount,
}: KanbanColumnProps) {
  // compact 模式：移动端直接渲染卡片，无列容器
  if (compact) {
    return (
      <>
        {tasks.map(task => (
          <TaskCard
            key={task.id}
            task={task}
            isSelected={selectedTaskId === task.id}
            onSelect={() => onSelectTask(task.id)}
            onStartRun={() => onStartRun(task.id)}
            onStopRun={() => onStopRun(task.id)}
            onDelete={() => onDeleteTask(task.id)}
          />
        ))}
      </>
    )
  }

  return (
    <div className={`rounded-lg p-3 lg:p-4 ${color} min-h-[200px] md:min-h-[400px] flex flex-col`}>
      {title && (
        <div className="flex items-center justify-between mb-3 lg:mb-4">
          <h2 className="font-semibold text-gray-700 text-sm lg:text-base">{title}</h2>
          <span className="text-xs lg:text-sm font-medium text-gray-500 bg-white/50 px-2 py-0.5 rounded">
            {totalCount != null && totalCount !== tasks.length ? `${tasks.length}/${totalCount}` : tasks.length}
          </span>
        </div>
      )}
      
      <div className="flex-1 space-y-2 lg:space-y-3 overflow-y-auto touch-scroll">
        {tasks.length === 0 ? (
          <div className="text-center py-6 md:py-8 text-sm text-gray-400">
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
