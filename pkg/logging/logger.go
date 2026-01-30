// Package logging 结构化日志
package logging

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"runtime"
	"time"
)

// ContextKey 上下文键类型
type ContextKey string

const (
	TraceIDKey ContextKey = "trace_id"
	SpanIDKey  ContextKey = "span_id"
	NodeIDKey  ContextKey = "node_id"
	RunIDKey   ContextKey = "run_id"
	TaskIDKey  ContextKey = "task_id"
)

// Logger 结构化日志器
type Logger struct {
	*slog.Logger
	component string
}

// Config 日志配置
type Config struct {
	Level     string `json:"level"`
	Format    string `json:"format"` // json or text
	Output    string `json:"output"` // stdout, stderr, or file path
	Component string `json:"component"`
}

// New 创建新的日志器
func New(cfg Config) *Logger {
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	var output io.Writer
	switch cfg.Output {
	case "stdout", "":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	default:
		f, err := os.OpenFile(cfg.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			output = os.Stdout
		} else {
			output = f
		}
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug,
	}

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(output, opts)
	} else {
		handler = slog.NewTextHandler(output, opts)
	}

	return &Logger{
		Logger:    slog.New(handler),
		component: cfg.Component,
	}
}

// Default 创建默认日志器
func Default(component string) *Logger {
	return New(Config{
		Level:     os.Getenv("LOG_LEVEL"),
		Format:    os.Getenv("LOG_FORMAT"),
		Output:    "stdout",
		Component: component,
	})
}

// WithContext 从上下文提取追踪信息
func (l *Logger) WithContext(ctx context.Context) *Logger {
	attrs := []any{slog.String("component", l.component)}

	if traceID, ok := ctx.Value(TraceIDKey).(string); ok && traceID != "" {
		attrs = append(attrs, slog.String("trace_id", traceID))
	}
	if spanID, ok := ctx.Value(SpanIDKey).(string); ok && spanID != "" {
		attrs = append(attrs, slog.String("span_id", spanID))
	}
	if nodeID, ok := ctx.Value(NodeIDKey).(string); ok && nodeID != "" {
		attrs = append(attrs, slog.String("node_id", nodeID))
	}
	if runID, ok := ctx.Value(RunIDKey).(string); ok && runID != "" {
		attrs = append(attrs, slog.String("run_id", runID))
	}
	if taskID, ok := ctx.Value(TaskIDKey).(string); ok && taskID != "" {
		attrs = append(attrs, slog.String("task_id", taskID))
	}

	return &Logger{
		Logger:    l.Logger.With(attrs...),
		component: l.component,
	}
}

// WithRunID 添加 Run ID
func (l *Logger) WithRunID(runID string) *Logger {
	return &Logger{
		Logger:    l.Logger.With(slog.String("run_id", runID)),
		component: l.component,
	}
}

// WithTaskID 添加 Task ID
func (l *Logger) WithTaskID(taskID string) *Logger {
	return &Logger{
		Logger:    l.Logger.With(slog.String("task_id", taskID)),
		component: l.component,
	}
}

// WithNodeID 添加 Node ID
func (l *Logger) WithNodeID(nodeID string) *Logger {
	return &Logger{
		Logger:    l.Logger.With(slog.String("node_id", nodeID)),
		component: l.component,
	}
}

// WithError 添加错误信息
func (l *Logger) WithError(err error) *Logger {
	if err == nil {
		return l
	}
	return &Logger{
		Logger:    l.Logger.With(slog.String("error", err.Error())),
		component: l.component,
	}
}

// WithDuration 添加持续时间
func (l *Logger) WithDuration(d time.Duration) *Logger {
	return &Logger{
		Logger:    l.Logger.With(slog.Float64("duration_ms", float64(d.Milliseconds()))),
		component: l.component,
	}
}

// LogEntry 日志条目（用于 Loki 兼容格式）
type LogEntry struct {
	Timestamp time.Time              `json:"ts"`
	Level     string                 `json:"level"`
	Message   string                 `json:"msg"`
	Component string                 `json:"component"`
	TraceID   string                 `json:"trace_id,omitempty"`
	SpanID    string                 `json:"span_id,omitempty"`
	NodeID    string                 `json:"node_id,omitempty"`
	RunID     string                 `json:"run_id,omitempty"`
	TaskID    string                 `json:"task_id,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Duration  float64                `json:"duration_ms,omitempty"`
	Caller    string                 `json:"caller,omitempty"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

// ToJSON 转换为 JSON 字符串
func (e *LogEntry) ToJSON() string {
	b, _ := json.Marshal(e)
	return string(b)
}

// HTTPRequestLog HTTP 请求日志
func (l *Logger) HTTPRequestLog(method, path string, status int, duration time.Duration, clientIP string) {
	l.Logger.Info("HTTP request",
		slog.String("method", method),
		slog.String("path", path),
		slog.Int("status", status),
		slog.Float64("duration_ms", float64(duration.Milliseconds())),
		slog.String("client_ip", clientIP),
	)
}

// DBQueryLog 数据库查询日志
func (l *Logger) DBQueryLog(operation, table string, duration time.Duration, err error) {
	attrs := []any{
		slog.String("operation", operation),
		slog.String("table", table),
		slog.Float64("duration_ms", float64(duration.Milliseconds())),
	}
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
		l.Logger.Error("DB query failed", attrs...)
	} else {
		l.Logger.Debug("DB query", attrs...)
	}
}

// TaskLog 任务日志
func (l *Logger) TaskLog(action, runID, taskID string, extra ...any) {
	attrs := []any{
		slog.String("action", action),
		slog.String("run_id", runID),
		slog.String("task_id", taskID),
	}
	attrs = append(attrs, extra...)
	l.Logger.Info("Task event", attrs...)
}

// HeartbeatLog 心跳日志
func (l *Logger) HeartbeatLog(nodeID, status string, latency time.Duration, err error) {
	attrs := []any{
		slog.String("node_id", nodeID),
		slog.String("status", status),
		slog.Float64("latency_ms", float64(latency.Milliseconds())),
	}
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
		l.Logger.Warn("Heartbeat failed", attrs...)
	} else {
		l.Logger.Debug("Heartbeat sent", attrs...)
	}
}

// GetCaller 获取调用者信息
func GetCaller(skip int) string {
	_, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return "unknown"
	}
	short := file
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			short = file[i+1:]
			break
		}
	}
	return short + ":" + string(rune(line))
}
