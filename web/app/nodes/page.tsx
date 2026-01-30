'use client'

import { useEffect, useState } from 'react'
import { Network, CheckCircle, Clock, AlertCircle } from 'lucide-react'
import { AdminLayout } from '@/components/layout'

interface Node {
  id: string
  status: string  // 从 etcd 判断：online/offline
  last_heartbeat?: string
}

// 获取节点状态（直接使用后端返回的状态，由 etcd 心跳判断）
const getNodeStatus = (node: Node): 'online' | 'offline' | 'unknown' => {
  if (node.status === 'online') return 'online'
  if (node.status === 'offline') return 'offline'
  return 'unknown'
}

export default function NodesPage() {
  const [nodes, setNodes] = useState<Node[]>([])
  const [loading, setLoading] = useState(true)

  const fetchNodes = async () => {
    setLoading(true)
    try {
      const res = await fetch('/api/v1/nodes')
      if (res.ok) {
        const data = await res.json()
        setNodes(data.nodes || [])
      }
    } catch (err) {
      console.error('Failed to fetch nodes:', err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchNodes()
    // 每10秒刷新一次节点状态
    const interval = setInterval(fetchNodes, 10000)
    return () => clearInterval(interval)
  }, [])

  const statusIcon = (status: 'online' | 'offline' | 'unknown') => {
    switch (status) {
      case 'online':
        return <CheckCircle className="w-5 h-5 text-green-500" />
      case 'offline':
        return <AlertCircle className="w-5 h-5 text-red-500" />
      default:
        return <Clock className="w-5 h-5 text-yellow-500" />
    }
  }

  const statusText = (status: 'online' | 'offline' | 'unknown') => {
    switch (status) {
      case 'online': return '在线'
      case 'offline': return '离线'
      default: return '未知'
    }
  }

  // 按状态排序：在线 > 离线 > 未知
  const sortedNodes = [...nodes].sort((a, b) => {
    const statusOrder = { online: 0, offline: 1, unknown: 2 }
    return statusOrder[getNodeStatus(a)] - statusOrder[getNodeStatus(b)]
  })

  return (
    <AdminLayout title="节点管理" onRefresh={fetchNodes} loading={loading}>
      <div className="mb-4">
        <p className="text-sm text-gray-500">监控计算节点状态（Phase 2 功能）</p>
      </div>

      {loading ? (
        <div className="flex items-center justify-center h-64">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
        </div>
      ) : nodes.length === 0 ? (
        <div className="bg-white rounded-lg border p-8 text-center">
          <Network className="w-12 h-12 mx-auto text-gray-400 mb-4" />
          <h3 className="text-lg font-medium mb-2">暂无节点</h3>
          <p className="text-gray-500">节点管理功能将在 Phase 2 实现</p>
        </div>
      ) : (
        <div className="bg-white rounded-lg border divide-y">
          {sortedNodes.map(node => {
            const status = getNodeStatus(node)
            return (
              <div key={node.id} className="px-4 py-3 flex items-center justify-between">
                <div className="flex items-center gap-3">
                  {statusIcon(status)}
                  <div>
                    <p className="font-medium">{node.id}</p>
                    <p className="text-sm text-gray-500">
                      {node.last_heartbeat 
                        ? `最后心跳: ${new Date(node.last_heartbeat).toLocaleString()}`
                        : '从未上报'}
                    </p>
                  </div>
                </div>
                <span className={`text-xs px-2 py-1 rounded-full ${
                  status === 'online' ? 'bg-green-100 text-green-700' :
                  status === 'offline' ? 'bg-red-100 text-red-700' :
                  'bg-gray-100 text-gray-700'
                }`}>
                  {statusText(status)}
                </span>
              </div>
            )
          })}
        </div>
      )}
    </AdminLayout>
  )
}
