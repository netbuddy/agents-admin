'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import { 
  Activity, Clock, CheckCircle2, XCircle, AlertCircle, 
  RefreshCw, Search, Filter, ChevronRight, Zap,
  TrendingUp, Timer, BarChart3
} from 'lucide-react';
import { AdminLayout } from '@/components/layout';
import { useTranslation } from 'react-i18next';
import { useFormatDate } from '@/i18n/useFormatDate';

interface WorkflowSummary {
  id: string;
  type: string;
  name: string;
  state: string;
  progress: number;
  event_count: number;
  start_time: string | null;
  update_time: string | null;
  end_time: string | null;
  duration_ms: number | null;
  node_id: string;
  error: string;
  metadata: Record<string, any>;
}

interface MonitorStats {
  total_workflows: number;
  active_workflows: number;
  completed_today: number;
  failed_today: number;
  avg_duration_ms: number;
  workflows_by_type: Record<string, number>;
  workflows_by_state: Record<string, number>;
}

const getWsBase = () => {
  if (typeof window === 'undefined') return 'ws://localhost:8080';
  if (process.env.NEXT_PUBLIC_WS_URL) return process.env.NEXT_PUBLIC_WS_URL;
  const host = window.location.hostname;
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${protocol}//${host}:8080`;
};

const stateConfig: Record<string, { color: string; bgColor: string; icon: any; labelKey: string }> = {
  pending: { color: 'text-gray-500', bgColor: 'bg-gray-100', icon: Clock, labelKey: 'status.waiting' },
  running: { color: 'text-blue-500', bgColor: 'bg-blue-100', icon: Activity, labelKey: 'status.running' },
  waiting: { color: 'text-yellow-500', bgColor: 'bg-yellow-100', icon: AlertCircle, labelKey: 'monitor.waitingUser' },
  completed: { color: 'text-green-500', bgColor: 'bg-green-100', icon: CheckCircle2, labelKey: 'status.completed' },
  failed: { color: 'text-red-500', bgColor: 'bg-red-100', icon: XCircle, labelKey: 'status.failed' },
  unknown: { color: 'text-gray-400', bgColor: 'bg-gray-50', icon: AlertCircle, labelKey: 'status.unknown' },
};

const typeLabelKeys: Record<string, string> = {
  auth: 'monitor.typeAuth',
  run: 'monitor.typeRun',
};

function formatDuration(ms: number | null): string {
  if (!ms) return '-';
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}

// formatTime is now handled by useFormatDate hook in components

function StatCard({ title, value, icon: Icon, trend, color }: { 
  title: string; 
  value: string | number; 
  icon: any; 
  trend?: string;
  color: string;
}) {
  return (
    <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-3 sm:p-5 hover:shadow-md transition-shadow">
      <div className="flex items-center justify-between">
        <div className="min-w-0">
          <p className="text-xs sm:text-sm text-gray-500 font-medium truncate">{title}</p>
          <p className="text-xl sm:text-2xl font-bold mt-0.5 sm:mt-1">{value}</p>
          {trend && <p className="text-xs text-gray-400 mt-0.5 sm:mt-1 hidden sm:block">{trend}</p>}
        </div>
        <div className={`p-2 sm:p-3 rounded-lg sm:rounded-xl ${color} flex-shrink-0`}>
          <Icon className="w-5 h-5 sm:w-6 sm:h-6 text-white" />
        </div>
      </div>
    </div>
  );
}

function WorkflowCard({ workflow, onClick }: { workflow: WorkflowSummary; onClick: () => void }) {
  const { t } = useTranslation();
  const { formatShortTime } = useFormatDate();
  const config = stateConfig[workflow.state] || stateConfig.unknown;
  const StateIcon = config.icon;

  return (
    <div 
      className="bg-white rounded-xl shadow-sm border border-gray-100 p-4 hover:shadow-md hover:border-blue-200 transition-all cursor-pointer"
      onClick={onClick}
    >
      <div className="flex items-start justify-between">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${config.bgColor} ${config.color}`}>
              {t(typeLabelKeys[workflow.type] || workflow.type, { ns: 'monitor', defaultValue: workflow.type })}
            </span>
            <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${config.bgColor} ${config.color}`}>
              <StateIcon className="w-3 h-3 inline mr-1" />
              {t(config.labelKey)}
            </span>
          </div>
          <h3 className="font-semibold text-gray-900 mt-2 truncate">{workflow.name}</h3>
          <p className="text-xs text-gray-500 mt-1 font-mono">{workflow.id}</p>
        </div>
        <ChevronRight className="w-5 h-5 text-gray-400 flex-shrink-0" />
      </div>

      {/* Progress Bar */}
      <div className="mt-4">
        <div className="flex justify-between text-xs text-gray-500 mb-1">
          <span>{t('label.progress')}</span>
          <span>{workflow.progress}%</span>
        </div>
        <div className="h-2 bg-gray-100 rounded-full overflow-hidden">
          <div 
            className={`h-full rounded-full transition-all ${
              workflow.state === 'failed' ? 'bg-red-500' :
              workflow.state === 'completed' ? 'bg-green-500' :
              workflow.state === 'waiting' ? 'bg-yellow-500' :
              'bg-blue-500'
            }`}
            style={{ width: `${workflow.progress}%` }}
          />
        </div>
      </div>

      {/* Meta Info */}
      <div className="flex items-center gap-4 mt-4 text-xs text-gray-500">
        <div className="flex items-center gap-1">
          <Zap className="w-3 h-3" />
          <span>{workflow.event_count} {t('monitor.events', { ns: 'monitor' })}</span>
        </div>
        <div className="flex items-center gap-1">
          <Timer className="w-3 h-3" />
          <span>{formatDuration(workflow.duration_ms)}</span>
        </div>
        <div className="flex items-center gap-1">
          <Clock className="w-3 h-3" />
          <span>{workflow.update_time ? formatShortTime(workflow.update_time) : '-'}</span>
        </div>
      </div>

      {workflow.error && (
        <div className="mt-3 p-2 bg-red-50 rounded-lg text-xs text-red-600 truncate">
          {workflow.error}
        </div>
      )}
    </div>
  );
}

export default function MonitorPage() {
  const { t } = useTranslation('monitor');
  const [workflows, setWorkflows] = useState<WorkflowSummary[]>([]);
  const [stats, setStats] = useState<MonitorStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [selectedWorkflow, setSelectedWorkflow] = useState<string | null>(null);
  const [filterType, setFilterType] = useState<string>('');
  const [filterState, setFilterState] = useState<string>('');
  const [searchTerm, setSearchTerm] = useState('');
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [wsConnected, setWsConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);

  const fetchData = useCallback(async () => {
    try {
      const params = new URLSearchParams();
      if (filterType) params.append('type', filterType);
      if (filterState) params.append('state', filterState);

      const [workflowsRes, statsRes] = await Promise.all([
        fetch(`/api/v1/monitor/workflows?${params}`),
        fetch(`/api/v1/monitor/stats`),
      ]);

      if (workflowsRes.ok) {
        const data = await workflowsRes.json();
        setWorkflows(data.workflows || []);
      }

      if (statsRes.ok) {
        const data = await statsRes.json();
        setStats(data);
      }
    } catch (error) {
      console.error('Failed to fetch monitor data:', error);
    } finally {
      setLoading(false);
    }
  }, [filterType, filterState]);

  // WebSocket 连接
  useEffect(() => {
    if (!autoRefresh) {
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
        setWsConnected(false);
      }
      return;
    }

    const connectWebSocket = () => {
      const ws = new WebSocket(`${getWsBase()}/ws/monitor`);
      wsRef.current = ws;

      ws.onopen = () => {
        console.log('[Monitor WS] Connected');
        setWsConnected(true);
        setLoading(false);
      };

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data);
          if (msg.type === 'workflows') {
            setWorkflows(msg.data || []);
          } else if (msg.type === 'stats') {
            setStats(msg.data);
          }
        } catch (e) {
          console.error('[Monitor WS] Parse error:', e);
        }
      };

      ws.onclose = () => {
        console.log('[Monitor WS] Disconnected');
        setWsConnected(false);
        // 自动重连
        if (autoRefresh) {
          setTimeout(connectWebSocket, 3000);
        }
      };

      ws.onerror = (error) => {
        console.error('[Monitor WS] Error:', error);
      };
    };

    connectWebSocket();

    return () => {
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, [autoRefresh]);

  // 初始加载（作为 WebSocket 的后备）
  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const filteredWorkflows = workflows.filter(w => {
    if (filterType && w.type !== filterType) return false;
    if (filterState && w.state !== filterState) return false;
    if (searchTerm) {
      const term = searchTerm.toLowerCase();
      return w.name.toLowerCase().includes(term) ||
             w.id.toLowerCase().includes(term);
    }
    return true;
  });

  if (selectedWorkflow) {
    const workflow = workflows.find(w => w.id === selectedWorkflow);
    if (workflow) {
      return (
        <AdminLayout title={t('title')} onRefresh={fetchData} loading={loading}>
          <WorkflowDetail 
            workflow={workflow} 
            onBack={() => setSelectedWorkflow(null)} 
          />
        </AdminLayout>
      );
    }
  }

  return (
    <AdminLayout title="工作流监控" onRefresh={fetchData} loading={loading}>
      {/* WebSocket 状态 + 控制栏 */}
      <div className="flex items-center gap-2 mb-4">
        <div className={`flex items-center gap-1.5 px-2 py-1 rounded-full text-xs ${
          wsConnected ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'
        }`}>
          <span className={`w-2 h-2 rounded-full ${wsConnected ? 'bg-green-500 animate-pulse' : 'bg-gray-400'}`} />
          <span>{wsConnected ? t('detail.liveConnection', { ns: 'tasks', defaultValue: 'Live' }) : t('status.offline')}</span>
        </div>
        <button
          onClick={() => setAutoRefresh(!autoRefresh)}
          className={`flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium transition-colors ${
            autoRefresh 
              ? 'bg-green-100 text-green-700' 
              : 'bg-gray-100 text-gray-600'
          }`}
        >
          <RefreshCw className={`w-3.5 h-3.5 ${autoRefresh && wsConnected ? 'animate-spin' : ''}`} />
          {autoRefresh ? t('action.refresh', { ns: 'common' }) : t('monitor.paused', { defaultValue: 'Paused' })}
        </button>
      </div>

      {/* Stats Grid */}
      {stats && (
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4 mb-4 sm:mb-6">
          <StatCard 
            title={t('monitor.totalWorkflows', { defaultValue: 'Total Workflows' })} 
            value={stats.total_workflows} 
            icon={BarChart3}
            color="bg-gradient-to-br from-blue-500 to-blue-600"
          />
          <StatCard 
            title={t('monitor.active', { defaultValue: 'Active' })} 
            value={stats.active_workflows} 
            icon={Activity}
            trend={t('monitor.liveRunning', { defaultValue: 'Running' })}
            color="bg-gradient-to-br from-green-500 to-green-600"
          />
          <StatCard 
            title={t('monitor.completedToday', { defaultValue: 'Completed Today' })} 
            value={stats.completed_today} 
            icon={CheckCircle2}
            color="bg-gradient-to-br from-emerald-500 to-emerald-600"
          />
          <StatCard 
            title={t('monitor.failedToday', { defaultValue: 'Failed Today' })} 
            value={stats.failed_today} 
            icon={XCircle}
            color="bg-gradient-to-br from-red-500 to-red-600"
          />
        </div>
      )}

      {/* Filters */}
      <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-3 sm:p-4 mb-4 sm:mb-6">
        <div className="flex flex-col sm:flex-row sm:items-center gap-3 sm:gap-4">
          <div className="flex-1 min-w-0">
            <div className="relative">
              <Search className="w-5 h-5 text-gray-400 absolute left-3 top-1/2 -translate-y-1/2" />
              <input
                type="text"
                placeholder={t('monitor.searchPlaceholder', { defaultValue: 'Search workflows...' })}
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="w-full pl-10 pr-4 py-2 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent text-sm"
              />
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Filter className="w-4 h-4 text-gray-500 hidden sm:block" />
            <select
              value={filterType}
              onChange={(e) => setFilterType(e.target.value)}
              className="flex-1 sm:flex-none px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
            >
              <option value="">{t('monitor.allTypes', { defaultValue: 'All Types' })}</option>
              <option value="auth">{t('monitor.typeAuth', { defaultValue: 'OAuth Auth' })}</option>
              <option value="run">{t('monitor.typeRun', { defaultValue: 'Task Execution' })}</option>
            </select>
            <select
              value={filterState}
              onChange={(e) => setFilterState(e.target.value)}
              className="flex-1 sm:flex-none px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-sm"
            >
              <option value="">{t('monitor.allStates', { defaultValue: 'All States' })}</option>
              <option value="pending">{t('status.waiting')}</option>
              <option value="running">{t('status.running')}</option>
              <option value="waiting">{t('monitor.waitingUser', { defaultValue: 'Waiting User' })}</option>
              <option value="completed">{t('status.completed')}</option>
              <option value="failed">{t('status.failed')}</option>
            </select>
          </div>
        </div>
      </div>

      {/* Workflow List */}
      {loading ? (
        <div className="flex items-center justify-center py-20">
          <RefreshCw className="w-8 h-8 text-blue-500 animate-spin" />
        </div>
      ) : filteredWorkflows.length === 0 ? (
        <div className="text-center py-20">
          <Activity className="w-12 h-12 text-gray-300 mx-auto mb-4" />
          <p className="text-gray-500">{t('noWorkflows')}</p>
          <p className="text-sm text-gray-400 mt-1">{t('noWorkflowsHint')}</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
          {filteredWorkflows.map(workflow => (
            <WorkflowCard 
              key={workflow.id} 
              workflow={workflow} 
              onClick={() => setSelectedWorkflow(workflow.id)}
            />
          ))}
        </div>
      )}
    </AdminLayout>
  );
}

// Workflow Detail Component
function WorkflowDetail({ workflow, onBack }: { workflow: WorkflowSummary; onBack: () => void }) {
  const { t } = useTranslation('monitor');
  const { formatShortTime } = useFormatDate();
  const [events, setEvents] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function fetchEvents() {
      try {
        const res = await fetch(
          `/api/v1/monitor/workflows/${workflow.type}/${workflow.id}/events`
        );
        if (res.ok) {
          const data = await res.json();
          setEvents(data.events || []);
        }
      } catch (error) {
        console.error('Failed to fetch events:', error);
      } finally {
        setLoading(false);
      }
    }
    fetchEvents();
  }, [workflow.id, workflow.type]);

  const config = stateConfig[workflow.state] || stateConfig.unknown;
  const StateIcon = config.icon;

  const levelConfig: Record<string, { color: string; bgColor: string }> = {
    info: { color: 'text-blue-600', bgColor: 'bg-blue-100' },
    success: { color: 'text-green-600', bgColor: 'bg-green-100' },
    warning: { color: 'text-yellow-600', bgColor: 'bg-yellow-100' },
    error: { color: 'text-red-600', bgColor: 'bg-red-100' },
  };

  return (
    <div>
      {/* 返回按钮 + 标题 */}
      <div className="flex items-center gap-3 mb-6">
        <button
          onClick={onBack}
          className="flex items-center gap-1.5 px-3 py-1.5 text-sm text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg transition-colors"
        >
          <ChevronRight className="w-4 h-4 rotate-180" />
          {t('action.goBack', { ns: 'common' })}
        </button>
        <div className="flex-1 min-w-0">
          <h2 className="text-lg font-bold text-gray-900 truncate">{workflow.name}</h2>
          <p className="text-xs text-gray-500 font-mono">{workflow.id}</p>
        </div>
        <div className={`flex items-center gap-2 px-3 py-1.5 rounded-full ${config.bgColor} ${config.color}`}>
          <StateIcon className="w-4 h-4" />
          <span className="text-sm font-medium">{t(config.labelKey)}</span>
        </div>
      </div>

      <div>
        {/* Info Cards */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 sm:gap-4 mb-4 sm:mb-6">
          <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-4">
            <p className="text-sm text-gray-500">{t('startedAt')}</p>
            <p className="text-lg font-semibold mt-1">{workflow.start_time ? formatShortTime(workflow.start_time) : '-'}</p>
          </div>
          <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-4">
            <p className="text-sm text-gray-500">{t('duration')}</p>
            <p className="text-lg font-semibold mt-1">{formatDuration(workflow.duration_ms)}</p>
          </div>
          <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-4">
            <p className="text-sm text-gray-500">{t('node')}</p>
            <p className="text-lg font-semibold mt-1 font-mono">{workflow.node_id || '-'}</p>
          </div>
        </div>

        {/* Progress */}
        <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-4 mb-6">
          <div className="flex justify-between text-sm mb-2">
            <span className="font-medium">{t('label.progress', { ns: 'common' })}</span>
            <span className="text-gray-500">{workflow.progress}%</span>
          </div>
          <div className="h-3 bg-gray-100 rounded-full overflow-hidden">
            <div 
              className={`h-full rounded-full transition-all ${
                workflow.state === 'failed' ? 'bg-red-500' :
                workflow.state === 'completed' ? 'bg-green-500' :
                workflow.state === 'waiting' ? 'bg-yellow-500' :
                'bg-blue-500'
              }`}
              style={{ width: `${workflow.progress}%` }}
            />
          </div>
        </div>

        {/* Metadata */}
        {workflow.metadata && Object.keys(workflow.metadata).length > 0 && (
          <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-4 mb-6">
            <h3 className="font-semibold text-gray-900 mb-3">{t('label.metadata', { ns: 'common' })}</h3>
            <div className="grid grid-cols-2 gap-3">
              {Object.entries(workflow.metadata).map(([key, value]) => (
                <div key={key} className="bg-gray-50 rounded-lg p-3">
                  <p className="text-xs text-gray-500">{key}</p>
                  <p className="text-sm font-mono mt-1 truncate" title={String(value)}>
                    {String(value)}
                  </p>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Error */}
        {workflow.error && (
          <div className="bg-red-50 border border-red-200 rounded-xl p-4 mb-6">
            <h3 className="font-semibold text-red-800 mb-2">{t('status.error', { ns: 'common' })}</h3>
            <p className="text-sm text-red-600 font-mono">{workflow.error}</p>
          </div>
        )}

        {/* Event Timeline */}
        <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-4">
          <h3 className="font-semibold text-gray-900 mb-4 flex items-center gap-2">
            <Zap className="w-5 h-5 text-blue-500" />
            {t('monitor.eventTimeline', { defaultValue: 'Event Timeline' })}
            <span className="text-sm font-normal text-gray-500">({events.length})</span>
          </h3>

          {loading ? (
            <div className="flex items-center justify-center py-10">
              <RefreshCw className="w-6 h-6 text-blue-500 animate-spin" />
            </div>
          ) : events.length === 0 ? (
            <div className="text-center py-10 text-gray-500">
              {t('debugPanel.noEvents')}
            </div>
          ) : (
            <div className="relative">
              <div className="absolute left-4 top-0 bottom-0 w-0.5 bg-gray-200" />
              <div className="space-y-4">
                {events.map((event, index) => {
                  const lc = levelConfig[event.level] || levelConfig.info;
                  return (
                    <div key={event.id || index} className="relative pl-10">
                      <div className={`absolute left-2 w-5 h-5 rounded-full ${lc.bgColor} flex items-center justify-center`}>
                        <div className={`w-2 h-2 rounded-full ${lc.color.replace('text-', 'bg-')}`} />
                      </div>
                      <div className="bg-gray-50 rounded-lg p-3">
                        <div className="flex items-center justify-between mb-2">
                          <span className={`text-sm font-medium ${lc.color}`}>{event.type}</span>
                          <span className="text-xs text-gray-400">
                            {event.timestamp ? formatShortTime(event.timestamp) : '-'}
                          </span>
                        </div>
                        {event.data && Object.keys(event.data).length > 0 && (
                          <pre className="text-xs text-gray-600 bg-white rounded p-2 overflow-x-auto">
                            {JSON.stringify(event.data, null, 2)}
                          </pre>
                        )}
                        {event.producer_id && (
                          <p className="text-xs text-gray-400 mt-2">
                            {t('node')}: {event.producer_id}
                          </p>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
