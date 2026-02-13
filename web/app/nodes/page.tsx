'use client'

import { useEffect, useState, useCallback } from 'react'
import {
  Network, CheckCircle, Clock, AlertCircle, Server,
  Cpu, HardDrive, Tag, ChevronRight, X, Trash2, RefreshCw, Plus, Pencil, Check
} from 'lucide-react'
import { AdminLayout } from '@/components/layout'
import AddNodeWizard from '@/components/AddNodeWizard'
import { useTranslation } from 'react-i18next'
import { useFormatDate } from '@/i18n/useFormatDate'

interface NodeLabels {
  [key: string]: string
}

interface NodeCapacity {
  max_runners?: number
  cpu?: string
  memory?: string
  [key: string]: any
}

interface Node {
  id: string
  display_name?: string
  status: string
  hostname?: string
  ips?: string
  labels?: NodeLabels
  capacity?: NodeCapacity
  last_heartbeat?: string
  created_at?: string
  updated_at?: string
}

type NodeStatus = 'online' | 'offline' | 'unknown'

const getNodeStatus = (node: Node): NodeStatus => {
  if (node.status === 'online') return 'online'
  if (node.status === 'offline') return 'offline'
  return 'unknown'
}

const statusConfig: Record<NodeStatus, { color: string; bg: string; dot: string; labelKey: string }> = {
  online:  { color: 'text-green-700', bg: 'bg-green-50 border-green-200', dot: 'bg-green-500', labelKey: 'online' },
  offline: { color: 'text-red-700',   bg: 'bg-red-50 border-red-200',     dot: 'bg-red-500',   labelKey: 'offline' },
  unknown: { color: 'text-gray-500',  bg: 'bg-gray-50 border-gray-200',   dot: 'bg-gray-400',  labelKey: 'unknown' },
}

// formatTime and timeAgo are now handled by useFormatDate hook and useTranslation in components

function NodeCard({ node, onClick }: { node: Node; onClick: () => void }) {
  const { t } = useTranslation('nodes')
  const { formatRelative } = useFormatDate()
  const status = getNodeStatus(node)
  const cfg = statusConfig[status]
  const displayName = node.display_name || node.hostname || node.id
  const firstIP = node.ips ? node.ips.split(',')[0] : ''
  const labelCount = node.labels ? Object.keys(node.labels).length : 0

  return (
    <div
      onClick={onClick}
      className={`bg-white rounded-xl border shadow-sm hover:shadow-md hover:border-blue-300 transition-all cursor-pointer p-4 sm:p-5`}
    >
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-3 min-w-0">
          <div className={`p-2 rounded-lg ${status === 'online' ? 'bg-green-100' : status === 'offline' ? 'bg-red-100' : 'bg-gray-100'}`}>
            <Server className={`w-5 h-5 ${cfg.color}`} />
          </div>
          <div className="min-w-0">
            <h3 className="font-semibold text-gray-900 truncate">{displayName}</h3>
            {firstIP && <p className="text-xs text-gray-500 mt-0.5 font-mono">{firstIP}</p>}
            {!firstIP && <p className="text-xs text-gray-500 mt-0.5">{t('heartbeat')}: {node.last_heartbeat ? formatRelative(node.last_heartbeat) : t('neverReported')}</p>}
          </div>
        </div>
        <div className="flex items-center gap-1.5 flex-shrink-0">
          <span className={`w-2 h-2 rounded-full ${cfg.dot} ${status === 'online' ? 'animate-pulse' : ''}`} />
          <span className={`text-xs font-medium ${cfg.color}`}>{t(cfg.labelKey)}</span>
        </div>
      </div>

      <div className="flex items-center gap-3 text-xs text-gray-500">
        <div className="flex items-center gap-1">
          <Clock className="w-3 h-3" />
          <span>{node.last_heartbeat ? formatRelative(node.last_heartbeat) : t('neverReported')}</span>
        </div>
        {labelCount > 0 && (
          <div className="flex items-center gap-1">
            <Tag className="w-3 h-3" />
            <span>{labelCount} {t('labels')}</span>
          </div>
        )}
        <ChevronRight className="w-4 h-4 ml-auto text-gray-400" />
      </div>
    </div>
  )
}

function NodeDetail({ node, onClose, onDelete, onUpdate }: { node: Node; onClose: () => void; onDelete: (id: string) => void; onUpdate?: (updated: Node) => void }) {
  const { t } = useTranslation('nodes')
  const { formatShortDateTime } = useFormatDate()
  const status = getNodeStatus(node)
  const cfg = statusConfig[status]
  const [deleting, setDeleting] = useState(false)
  const [editingName, setEditingName] = useState(false)
  const [nameInput, setNameInput] = useState(node.display_name || '')
  const [nameError, setNameError] = useState('')
  const displayName = node.display_name || node.hostname || node.id
  const ipList = node.ips ? node.ips.split(',').filter(Boolean) : []

  const handleSaveName = async () => {
    const trimmed = nameInput.trim()
    if (!trimmed) { setNameError(t('nameRequired')); return }
    setNameError('')
    try {
      const res = await fetch(`/api/v1/nodes/${node.id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ display_name: trimmed }),
      })
      if (res.ok) {
        const updated = await res.json()
        onUpdate?.({ ...node, display_name: updated.display_name })
        setEditingName(false)
      } else if (res.status === 409) {
        const data = await res.json()
        setNameError(data.error || t('nameDuplicate'))
      }
    } catch (err) {
      console.error('Update display name failed:', err)
    }
  }

  const handleDelete = async () => {
    if (!confirm(t('confirmDelete', { id: displayName }))) return
    setDeleting(true)
    try {
      const res = await fetch(`/api/v1/nodes/${node.id}`, { method: 'DELETE' })
      if (res.ok || res.status === 204) {
        onDelete(node.id)
        onClose()
      }
    } catch (err) {
      console.error('Delete node failed:', err)
    } finally {
      setDeleting(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-end sm:items-center justify-center">
      <div className="fixed inset-0 bg-black/40" onClick={onClose} />
      <div className="relative bg-white rounded-t-2xl sm:rounded-xl shadow-xl w-full sm:max-w-lg max-h-[85vh] overflow-y-auto z-10">
        {/* Header */}
        <div className="sticky top-0 bg-white border-b px-5 py-4 flex items-center justify-between rounded-t-2xl sm:rounded-t-xl">
          <div className="flex items-center gap-3 min-w-0">
            <div className={`p-2 rounded-lg ${status === 'online' ? 'bg-green-100' : 'bg-red-100'}`}>
              <Server className={`w-5 h-5 ${cfg.color}`} />
            </div>
            <div className="min-w-0">
              {editingName ? (
                <div>
                  <div className="flex items-center gap-1.5">
                    <input
                      type="text"
                      value={nameInput}
                      onChange={e => { setNameInput(e.target.value); setNameError('') }}
                      onKeyDown={e => e.key === 'Enter' && handleSaveName()}
                      className={`text-sm font-bold border rounded px-2 py-0.5 w-40 focus:outline-none focus:ring-2 focus:ring-blue-500 ${nameError ? 'border-red-400' : ''}`}
                      autoFocus
                      placeholder={node.hostname || node.id}
                    />
                    <button onClick={handleSaveName} className="p-1 hover:bg-green-100 rounded">
                      <Check className="w-4 h-4 text-green-600" />
                    </button>
                    <button onClick={() => { setEditingName(false); setNameError('') }} className="p-1 hover:bg-gray-100 rounded">
                      <X className="w-4 h-4 text-gray-400" />
                    </button>
                  </div>
                  {nameError && <p className="text-xs text-red-500 mt-0.5">{nameError}</p>}
                </div>
              ) : (
                <div className="flex items-center gap-1.5">
                  <h2 className="font-bold text-gray-900 truncate">{displayName}</h2>
                  <button onClick={() => { setNameInput(node.display_name || ''); setEditingName(true) }} className="p-1 hover:bg-gray-100 rounded" title={t('editDisplayName')}>
                    <Pencil className="w-3.5 h-3.5 text-gray-400" />
                  </button>
                </div>
              )}
              <div className="flex items-center gap-1.5 mt-0.5">
                <span className={`w-2 h-2 rounded-full ${cfg.dot}`} />
                <span className={`text-xs font-medium ${cfg.color}`}>{t(cfg.labelKey)}</span>
              </div>
            </div>
          </div>
          <button onClick={onClose} className="p-2 hover:bg-gray-100 rounded-lg">
            <X className="w-5 h-5 text-gray-500" />
          </button>
        </div>

        {/* Body */}
        <div className="p-5 space-y-5">
          {/* Host Info */}
          <div>
            <h3 className="text-sm font-semibold text-gray-700 mb-3">{t('hostInfo')}</h3>
            <div className="grid grid-cols-2 gap-3">
              <InfoItem label={t('hostname')} value={node.hostname || '-'} />
              <InfoItem label={t('status')} value={t(cfg.labelKey)} />
              {ipList.length > 0 && (
                <div className="col-span-2">
                  <p className="text-xs text-gray-500 mb-1">{t('ipAddresses')}</p>
                  <div className="flex flex-wrap gap-1.5">
                    {ipList.map(ip => (
                      <span key={ip} className="inline-flex px-2 py-0.5 rounded text-xs font-mono bg-gray-100 text-gray-700">{ip}</span>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Basic Info */}
          <div>
            <h3 className="text-sm font-semibold text-gray-700 mb-3">{t('basicInfo')}</h3>
            <div className="grid grid-cols-2 gap-3">
              <InfoItem label={t('nodeId')} value={node.id} />
              <InfoItem label={t('lastHeartbeat')} value={node.last_heartbeat ? formatShortDateTime(node.last_heartbeat) : '-'} />
              <InfoItem label={t('createdAt')} value={node.created_at ? formatShortDateTime(node.created_at) : '-'} />
            </div>
          </div>

          {/* Capacity */}
          {node.capacity && Object.keys(node.capacity).length > 0 && (
            <div>
              <h3 className="text-sm font-semibold text-gray-700 mb-3">{t('capacityConfig')}</h3>
              <div className="bg-gray-50 rounded-lg p-3">
                <pre className="text-xs text-gray-600 whitespace-pre-wrap">
                  {JSON.stringify(node.capacity, null, 2)}
                </pre>
              </div>
            </div>
          )}

          {/* Labels */}
          {node.labels && Object.keys(node.labels).length > 0 && (
            <div>
              <h3 className="text-sm font-semibold text-gray-700 mb-3">{t('labels')}</h3>
              <div className="flex flex-wrap gap-2">
                {Object.entries(node.labels).map(([k, v]) => (
                  <span key={k} className="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium bg-blue-50 text-blue-700 border border-blue-200">
                    {k}: {v}
                  </span>
                ))}
              </div>
            </div>
          )}

          {/* Danger Zone */}
          <div className="border-t pt-4">
            <button
              onClick={handleDelete}
              disabled={deleting}
              className="flex items-center gap-2 px-4 py-2 text-sm text-red-600 hover:bg-red-50 rounded-lg transition-colors disabled:opacity-50"
            >
              <Trash2 className="w-4 h-4" />
              {deleting ? t('deleting') : t('deleteNode')}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

function InfoItem({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-gray-50 rounded-lg p-3">
      <p className="text-xs text-gray-500">{label}</p>
      <p className="text-sm font-medium text-gray-900 mt-0.5 truncate" title={value}>{value}</p>
    </div>
  )
}

export default function NodesPage() {
  const { t } = useTranslation('nodes')
  const [nodes, setNodes] = useState<Node[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedNode, setSelectedNode] = useState<string | null>(null)
  const [showAddWizard, setShowAddWizard] = useState(false)

  const fetchNodes = useCallback(async (showLoading = false) => {
    if (showLoading) setLoading(true)
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
  }, [])

  useEffect(() => {
    fetchNodes(true)
    const interval = setInterval(() => fetchNodes(), 10000)
    return () => clearInterval(interval)
  }, [fetchNodes])

  const handleDelete = (id: string) => {
    setNodes(prev => prev.filter(n => n.id !== id))
  }

  // 按状态排序：在线 > 离线 > 未知
  const sortedNodes = [...nodes].sort((a, b) => {
    const order: Record<NodeStatus, number> = { online: 0, offline: 1, unknown: 2 }
    return order[getNodeStatus(a)] - order[getNodeStatus(b)]
  })

  const onlineCount = nodes.filter(n => getNodeStatus(n) === 'online').length
  const offlineCount = nodes.filter(n => getNodeStatus(n) !== 'online').length
  const selected = selectedNode ? nodes.find(n => n.id === selectedNode) : null

  return (
    <AdminLayout title={t('title')} onRefresh={fetchNodes} loading={loading}>
      {/* 操作栏 */}
      <div className="flex justify-end mb-4">
        <button
          onClick={() => setShowAddWizard(true)}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white text-sm rounded-lg hover:bg-blue-700 transition-colors"
        >
          <Plus className="w-4 h-4" />
          {t('addNode')}
        </button>
      </div>

      {/* 统计概览 */}
      <div className="grid grid-cols-3 gap-3 mb-5">
        <div className="bg-white rounded-xl border p-4 text-center">
          <p className="text-2xl font-bold text-gray-900">{nodes.length}</p>
          <p className="text-xs text-gray-500 mt-1">{t('total')}</p>
        </div>
        <div className="bg-white rounded-xl border p-4 text-center">
          <p className="text-2xl font-bold text-green-600">{onlineCount}</p>
          <p className="text-xs text-gray-500 mt-1">{t('online')}</p>
        </div>
        <div className="bg-white rounded-xl border p-4 text-center">
          <p className="text-2xl font-bold text-red-600">{offlineCount}</p>
          <p className="text-xs text-gray-500 mt-1">{t('offline')}</p>
        </div>
      </div>

      {loading ? (
        <div className="flex items-center justify-center h-64">
          <RefreshCw className="w-8 h-8 text-blue-500 animate-spin" />
        </div>
      ) : nodes.length === 0 ? (
        <div className="bg-white rounded-xl border p-8 text-center">
          <Network className="w-12 h-12 mx-auto text-gray-400 mb-4" />
          <h3 className="text-lg font-medium mb-2">{t('noNodes')}</h3>
          <p className="text-gray-500 text-sm">{t('noNodesHint')}</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
          {sortedNodes.map(node => (
            <NodeCard
              key={node.id}
              node={node}
              onClick={() => setSelectedNode(node.id)}
            />
          ))}
        </div>
      )}

      {/* 节点详情弹窗 */}
      {selected && (
        <NodeDetail
          node={selected}
          onClose={() => setSelectedNode(null)}
          onDelete={handleDelete}
          onUpdate={(updated) => setNodes(prev => prev.map(n => n.id === updated.id ? updated : n))}
        />
      )}

      {/* 添加节点向导 */}
      {showAddWizard && (
        <AddNodeWizard
          onClose={() => setShowAddWizard(false)}
          onSuccess={() => fetchNodes(true)}
        />
      )}
    </AdminLayout>
  )
}
