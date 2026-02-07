'use client'

import { useState } from 'react'
import { Play, Square, Eye, Trash2, Clock, CheckCircle, XCircle, RefreshCw, User, Server } from 'lucide-react'

interface Task {
  id: string
  name: string
  status: string
  type?: string
  prompt?: { content?: string }
  agent_id?: string
  spec?: any
  created_at: string
}

interface TaskCardProps {
  task: Task
  isSelected?: boolean
  onSelect: () => void
  onStartRun: () => void
  onStopRun: () => void
  onDelete: () => void
}

const statusConfig: Record<string, { icon: React.ReactNode; color: string; bgColor: string; label: string }> = {
  pending: {
    icon: <Clock className="w-4 h-4" />,
    color: 'text-gray-500',
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

export default function TaskCard({ 
  task, 
  isSelected, 
  onSelect, 
  onStartRun, 
  onStopRun, 
  onDelete 
}: TaskCardProps) {
  const [loading, setLoading] = useState(false)
  
  const agentType = task.type || task.spec?.agent?.type || 'unknown'
  const accountId = task.agent_id || task.spec?.agent?.account_id
  const prompt = task.prompt?.content || task.spec?.prompt || ''
  
  const status = statusConfig[task.status] || statusConfig.pending

  const handleStartRun = async (e: React.MouseEvent) => {
    e.stopPropagation()
    setLoading(true)
    try {
      await onStartRun()
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = (e: React.MouseEvent) => {
    e.stopPropagation()
    if (confirm('确定删除此任务？')) {
      onDelete()
    }
  }

  const handleStopRun = (e: React.MouseEvent) => {
    e.stopPropagation()
    onStopRun()
  }

  return (
    <div 
      onClick={onSelect}
      className={`bg-white rounded-lg shadow-sm border transition-all cursor-pointer hover:shadow-md ${
        isSelected ? 'ring-2 ring-blue-500 border-blue-500' : 'border-gray-200'
      }`}
    >
      {/* 头部：标题和 Agent 类型 */}
      <div className="p-3 border-b border-gray-100">
        <div className="flex items-start justify-between gap-2">
          <h3 className="font-medium text-sm truncate flex-1">{task.name}</h3>
          <span className="text-xs px-2 py-0.5 rounded bg-gray-100 text-gray-600 flex-shrink-0">
            {agentType}
          </span>
        </div>
        
        {/* 账号信息 */}
        {accountId && (
          <div className="mt-2 flex items-center gap-3 text-xs text-gray-500">
            <span className="flex items-center gap-1">
              <User className="w-3 h-3" />
              <span className="truncate max-w-[120px]">{accountId}</span>
            </span>
          </div>
        )}
      </div>

      {/* 中间：提示词预览 */}
      <div className="p-3">
        <p className="text-xs text-gray-500 line-clamp-2 min-h-[2.5rem]">
          {prompt || '无提示词'}
        </p>
      </div>

      {/* 状态栏 */}
      <div className={`px-3 py-2 ${status.bgColor} border-t border-gray-100`}>
        <div className="flex items-center gap-2">
          <span className={status.color}>{status.icon}</span>
          <span className={`text-xs font-medium ${status.color}`}>{status.label}</span>
          {task.status === 'running' && (
            <span className="text-xs text-gray-500">· 执行中...</span>
          )}
        </div>
      </div>

      {/* 底部：时间和操作按钮 */}
      <div className="px-3 py-2 flex items-center justify-between border-t border-gray-100">
        <span className="text-xs text-gray-400">
          {new Date(task.created_at).toLocaleString('zh-CN', { 
            month: 'numeric', 
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
          })}
        </span>
        <div className="flex gap-0.5 sm:gap-1">
          <button
            onClick={(e) => { e.stopPropagation(); onSelect(); }}
            className="p-2 sm:p-1.5 rounded hover:bg-gray-100 text-gray-500"
            title="查看详情"
          >
            <Eye className="w-4 h-4 sm:w-3.5 sm:h-3.5" />
          </button>
          {task.status === 'pending' && (
            <button
              onClick={handleStartRun}
              disabled={loading}
              className="p-2 sm:p-1.5 rounded hover:bg-blue-100 text-blue-600 disabled:opacity-50"
              title="开始执行"
            >
              <Play className="w-4 h-4 sm:w-3.5 sm:h-3.5" />
            </button>
          )}
          {task.status === 'running' && (
            <button
              onClick={handleStopRun}
              className="p-2 sm:p-1.5 rounded hover:bg-red-100 text-red-600"
              title="停止"
            >
              <Square className="w-4 h-4 sm:w-3.5 sm:h-3.5" />
            </button>
          )}
          <button
            onClick={handleDelete}
            className="p-2 sm:p-1.5 rounded hover:bg-red-100 text-red-600"
            title="删除"
          >
            <Trash2 className="w-4 h-4 sm:w-3.5 sm:h-3.5" />
          </button>
        </div>
      </div>
    </div>
  )
}
