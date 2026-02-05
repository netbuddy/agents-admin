// Package executor Prometheus 指标导出
package nodemanager

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics 包含所有 Executor 指标
type Metrics struct {
	// 心跳指标
	HeartbeatTotal   prometheus.Counter
	HeartbeatErrors  prometheus.Counter
	HeartbeatLatency prometheus.Histogram

	// 任务执行指标
	TasksPending prometheus.Gauge
	TasksRunning prometheus.Gauge
	TasksTotal   *prometheus.CounterVec
	TaskDuration *prometheus.HistogramVec

	// 容器指标
	ContainerStartsTotal *prometheus.CounterVec
	ContainersRunning    prometheus.Gauge

	// 事件上报指标
	EventsReported     *prometheus.CounterVec
	EventReportErrors  prometheus.Counter
	EventReportLatency prometheus.Histogram

	// 认证任务指标
	AuthTasksTotal   *prometheus.CounterVec
	AuthTasksRunning prometheus.Gauge
}

// NewMetrics 创建 Executor 指标实例
func NewMetrics(namespace, nodeID string) *Metrics {
	labels := prometheus.Labels{"node_id": nodeID}

	return &Metrics{
		HeartbeatTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Name:        "heartbeat_total",
				Help:        "Total heartbeats sent",
				ConstLabels: labels,
			},
		),
		HeartbeatErrors: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Name:        "heartbeat_errors_total",
				Help:        "Total heartbeat errors",
				ConstLabels: labels,
			},
		),
		HeartbeatLatency: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Namespace:   namespace,
				Name:        "heartbeat_latency_seconds",
				Help:        "Heartbeat latency in seconds",
				Buckets:     []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
				ConstLabels: labels,
			},
		),
		TasksPending: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "tasks_pending",
				Help:        "Number of pending tasks",
				ConstLabels: labels,
			},
		),
		TasksRunning: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "tasks_running",
				Help:        "Number of currently running tasks",
				ConstLabels: labels,
			},
		),
		TasksTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Name:        "tasks_total",
				Help:        "Total tasks executed by status",
				ConstLabels: labels,
			},
			[]string{"driver", "status"},
		),
		TaskDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace:   namespace,
				Name:        "task_duration_seconds",
				Help:        "Task execution duration in seconds",
				Buckets:     []float64{1, 5, 10, 30, 60, 120, 300, 600, 1800, 3600},
				ConstLabels: labels,
			},
			[]string{"driver", "status"},
		),
		ContainerStartsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Name:        "container_starts_total",
				Help:        "Total container starts by image",
				ConstLabels: labels,
			},
			[]string{"image", "status"},
		),
		ContainersRunning: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "containers_running",
				Help:        "Number of currently running containers",
				ConstLabels: labels,
			},
		),
		EventsReported: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Name:        "events_reported_total",
				Help:        "Total events reported by type",
				ConstLabels: labels,
			},
			[]string{"type"},
		),
		EventReportErrors: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Name:        "event_report_errors_total",
				Help:        "Total event report errors",
				ConstLabels: labels,
			},
		),
		EventReportLatency: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Namespace:   namespace,
				Name:        "event_report_latency_seconds",
				Help:        "Event report latency in seconds",
				Buckets:     []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
				ConstLabels: labels,
			},
		),
		AuthTasksTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Name:        "auth_tasks_total",
				Help:        "Total auth tasks by status",
				ConstLabels: labels,
			},
			[]string{"status"},
		),
		AuthTasksRunning: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "auth_tasks_running",
				Help:        "Number of currently running auth tasks",
				ConstLabels: labels,
			},
		),
	}
}

// RecordHeartbeat 记录心跳
func (m *Metrics) RecordHeartbeat(latency time.Duration, success bool) {
	m.HeartbeatTotal.Inc()
	m.HeartbeatLatency.Observe(latency.Seconds())
	if !success {
		m.HeartbeatErrors.Inc()
	}
}

// RecordTaskStart 记录任务开始
func (m *Metrics) RecordTaskStart() {
	m.TasksRunning.Inc()
}

// RecordTaskComplete 记录任务完成
func (m *Metrics) RecordTaskComplete(driver, status string, duration time.Duration) {
	m.TasksRunning.Dec()
	m.TasksTotal.WithLabelValues(driver, status).Inc()
	m.TaskDuration.WithLabelValues(driver, status).Observe(duration.Seconds())
}

// RecordContainerStart 记录容器启动
func (m *Metrics) RecordContainerStart(image string, success bool) {
	status := "success"
	if !success {
		status = "failed"
	}
	m.ContainerStartsTotal.WithLabelValues(image, status).Inc()
	if success {
		m.ContainersRunning.Inc()
	}
}

// RecordContainerStop 记录容器停止
func (m *Metrics) RecordContainerStop() {
	m.ContainersRunning.Dec()
}

// RecordEventReport 记录事件上报
func (m *Metrics) RecordEventReport(eventType string, latency time.Duration, success bool) {
	m.EventsReported.WithLabelValues(eventType).Inc()
	m.EventReportLatency.Observe(latency.Seconds())
	if !success {
		m.EventReportErrors.Inc()
	}
}

// RecordAuthTaskStart 记录认证任务开始
func (m *Metrics) RecordAuthTaskStart() {
	m.AuthTasksRunning.Inc()
}

// RecordAuthTaskComplete 记录认证任务完成
func (m *Metrics) RecordAuthTaskComplete(status string) {
	m.AuthTasksRunning.Dec()
	m.AuthTasksTotal.WithLabelValues(status).Inc()
}

// SetTasksPending 设置待处理任务数
func (m *Metrics) SetTasksPending(count int) {
	m.TasksPending.Set(float64(count))
}

// Handler 返回 Prometheus HTTP Handler
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
