// Package api Prometheus 指标导出
package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics 包含所有 API Server 指标
type Metrics struct {
	// HTTP 请求指标
	HTTPRequestsTotal    *prometheus.CounterVec
	HTTPRequestDuration  *prometheus.HistogramVec
	HTTPRequestsInFlight prometheus.Gauge

	// 任务指标
	TasksTotal  *prometheus.GaugeVec
	RunsTotal   *prometheus.GaugeVec
	RunDuration *prometheus.HistogramVec

	// 调度器指标
	SchedulerCyclesTotal   prometheus.Counter
	SchedulerRunsAssigned  prometheus.Counter
	SchedulerCycleDuration prometheus.Histogram

	// WebSocket 指标
	WSConnectionsActive prometheus.Gauge
	WSMessagesTotal     *prometheus.CounterVec

	// 节点指标
	NodesOnline prometheus.Gauge
	NodesTotal  prometheus.Gauge

	// 数据库指标
	DBQueryTotal    *prometheus.CounterVec
	DBQueryDuration *prometheus.HistogramVec
}

// NewMetrics 创建指标实例
func NewMetrics(namespace string) *Metrics {
	return &Metrics{
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_total",
				Help:      "Total HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"method", "path"},
		),
		HTTPRequestsInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "http_requests_in_flight",
				Help:      "Current number of HTTP requests being processed",
			},
		),
		TasksTotal: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "tasks_total",
				Help:      "Total tasks by status",
			},
			[]string{"status"},
		),
		RunsTotal: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "runs_total",
				Help:      "Total runs by status and agent type",
			},
			[]string{"status", "agent_type"},
		),
		RunDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "run_duration_seconds",
				Help:      "Run execution duration in seconds",
				Buckets:   []float64{1, 5, 10, 30, 60, 120, 300, 600, 1800, 3600},
			},
			[]string{"agent_type", "status"},
		),
		SchedulerCyclesTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "scheduler_cycles_total",
				Help:      "Total scheduler cycles",
			},
		),
		SchedulerRunsAssigned: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "scheduler_runs_assigned_total",
				Help:      "Total runs assigned by scheduler",
			},
		),
		SchedulerCycleDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "scheduler_cycle_duration_seconds",
				Help:      "Scheduler cycle duration in seconds",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1},
			},
		),
		WSConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "websocket_connections_active",
				Help:      "Active WebSocket connections",
			},
		),
		WSMessagesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "websocket_messages_total",
				Help:      "Total WebSocket messages",
			},
			[]string{"direction", "type"},
		),
		NodesOnline: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "nodes_online",
				Help:      "Number of online nodes",
			},
		),
		NodesTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "nodes_total",
				Help:      "Total number of registered nodes",
			},
		),
		DBQueryTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "db_queries_total",
				Help:      "Total database queries",
			},
			[]string{"operation", "table"},
		),
		DBQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "db_query_duration_seconds",
				Help:      "Database query duration in seconds",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
			},
			[]string{"operation", "table"},
		),
	}
}

// MetricsMiddleware 创建 HTTP 指标中间件
func (m *Metrics) MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		m.HTTPRequestsInFlight.Inc()
		defer m.HTTPRequestsInFlight.Dec()

		// 包装 ResponseWriter 以捕获状态码
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()
		path := normalizePath(r.URL.Path)
		status := strconv.Itoa(wrapped.statusCode)

		m.HTTPRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
		m.HTTPRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}

// responseWriter 包装 http.ResponseWriter 以捕获状态码
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// normalizePath 规范化路径，将 ID 替换为占位符
func normalizePath(path string) string {
	// 简单的路径规范化，避免高基数
	// 例如 /api/v1/tasks/task-123 -> /api/v1/tasks/{id}
	switch {
	case len(path) > 20 && path[:15] == "/api/v1/tasks/":
		return "/api/v1/tasks/{id}"
	case len(path) > 19 && path[:14] == "/api/v1/runs/":
		return "/api/v1/runs/{id}"
	case len(path) > 20 && path[:15] == "/api/v1/nodes/":
		return "/api/v1/nodes/{id}"
	case len(path) > 23 && path[:18] == "/api/v1/accounts/":
		return "/api/v1/accounts/{id}"
	default:
		return path
	}
}

// Handler 返回 Prometheus HTTP Handler
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// RecordDBQuery 记录数据库查询指标
func (m *Metrics) RecordDBQuery(operation, table string, duration time.Duration) {
	m.DBQueryTotal.WithLabelValues(operation, table).Inc()
	m.DBQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

// RecordRunCompleted 记录 Run 完成指标
func (m *Metrics) RecordRunCompleted(agentType, status string, duration time.Duration) {
	m.RunDuration.WithLabelValues(agentType, status).Observe(duration.Seconds())
}

// RecordSchedulerCycle 记录调度器周期
func (m *Metrics) RecordSchedulerCycle(duration time.Duration, runsAssigned int) {
	m.SchedulerCyclesTotal.Inc()
	m.SchedulerCycleDuration.Observe(duration.Seconds())
	m.SchedulerRunsAssigned.Add(float64(runsAssigned))
}

// RecordWSMessage 记录 WebSocket 消息
func (m *Metrics) RecordWSMessage(direction, msgType string) {
	m.WSMessagesTotal.WithLabelValues(direction, msgType).Inc()
}

// SetNodesCount 设置节点数量
func (m *Metrics) SetNodesCount(online, total int) {
	m.NodesOnline.Set(float64(online))
	m.NodesTotal.Set(float64(total))
}

// SetTasksCount 设置任务数量
func (m *Metrics) SetTasksCount(status string, count int) {
	m.TasksTotal.WithLabelValues(status).Set(float64(count))
}

// SetRunsCount 设置 Run 数量
func (m *Metrics) SetRunsCount(status, agentType string, count int) {
	m.RunsTotal.WithLabelValues(status, agentType).Set(float64(count))
}

// WSConnectionOpened WebSocket 连接打开
func (m *Metrics) WSConnectionOpened() {
	m.WSConnectionsActive.Inc()
}

// WSConnectionClosed WebSocket 连接关闭
func (m *Metrics) WSConnectionClosed() {
	m.WSConnectionsActive.Dec()
}
