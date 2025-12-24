package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"
)

// LogLevel represents the severity level of a log message
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

// String returns string representation of log level
func (l LogLevel) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Fields represents structured log fields
type Fields map[string]interface{}

// StructuredLogger provides structured JSON logging with context
type StructuredLogger struct {
	level      LogLevel
	output     io.Writer
	mu         sync.Mutex
	service    string
	version    string
	hostname   string
}

// LogEntry represents a single structured log entry
type LogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Level       string                 `json:"level"`
	Service     string                 `json:"service"`
	Version     string                 `json:"version"`
	Hostname    string                 `json:"hostname"`
	Message     string                 `json:"message"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
	TenantID    string                 `json:"tenant_id,omitempty"`
	File        string                 `json:"file,omitempty"`
	Line        int                    `json:"line,omitempty"`
	Function    string                 `json:"function,omitempty"`
	Error       string                 `json:"error,omitempty"`
	StackTrace  string                 `json:"stack_trace,omitempty"`
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(service, version string, level LogLevel) *StructuredLogger {
	hostname, _ := os.Hostname()

	return &StructuredLogger{
		level:    level,
		output:   os.Stdout,
		service:  service,
		version:  version,
		hostname: hostname,
	}
}

// SetOutput sets the output destination for logs
func (l *StructuredLogger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = w
}

// SetLevel sets the minimum log level
func (l *StructuredLogger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// Debug logs a debug message with structured fields
func (l *StructuredLogger) Debug(ctx context.Context, message string, fields Fields) {
	l.log(ctx, DebugLevel, message, fields, nil)
}

// Info logs an info message with structured fields
func (l *StructuredLogger) Info(ctx context.Context, message string, fields Fields) {
	l.log(ctx, InfoLevel, message, fields, nil)
}

// Warn logs a warning message with structured fields
func (l *StructuredLogger) Warn(ctx context.Context, message string, fields Fields) {
	l.log(ctx, WarnLevel, message, fields, nil)
}

// Error logs an error message with structured fields and error details
func (l *StructuredLogger) Error(ctx context.Context, message string, fields Fields, err error) {
	l.log(ctx, ErrorLevel, message, fields, err)
}

// Fatal logs a fatal message and exits the program
func (l *StructuredLogger) Fatal(ctx context.Context, message string, fields Fields, err error) {
	l.log(ctx, FatalLevel, message, fields, err)
	os.Exit(1)
}

// log is the internal logging implementation
func (l *StructuredLogger) log(ctx context.Context, level LogLevel, message string, fields Fields, err error) {
	// Check log level
	if level < l.level {
		return
	}

	// Create log entry
	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     level.String(),
		Service:   l.service,
		Version:   l.version,
		Hostname:  l.hostname,
		Message:   message,
		Fields:    fields,
	}

	// Extract context values
	if ctx != nil {
		if requestID, ok := ctx.Value("request_id").(string); ok {
			entry.RequestID = requestID
		}
		if tenantID, ok := ctx.Value("tenant_id").(string); ok {
			entry.TenantID = tenantID
		}
	}

	// Add caller information for error and fatal levels
	if level >= ErrorLevel {
		if pc, file, line, ok := runtime.Caller(2); ok {
			entry.File = file
			entry.Line = line
			if fn := runtime.FuncForPC(pc); fn != nil {
				entry.Function = fn.Name()
			}
		}

		// Add error details
		if err != nil {
			entry.Error = err.Error()

			// Add stack trace for fatal errors
			if level == FatalLevel {
				entry.StackTrace = captureStackTrace()
			}
		}
	}

	// Marshal to JSON
	data, marshalErr := json.Marshal(entry)
	if marshalErr != nil {
		// Fallback to simple log if JSON marshaling fails
		fmt.Fprintf(os.Stderr, "Failed to marshal log entry: %v\n", marshalErr)
		fmt.Fprintf(os.Stderr, "%s [%s] %s: %v\n",
			entry.Timestamp.Format(time.RFC3339),
			entry.Level,
			message,
			fields)
		return
	}

	// Write log entry
	l.mu.Lock()
	defer l.mu.Unlock()

	l.output.Write(data)
	l.output.Write([]byte("\n"))
}

// captureStackTrace captures the current stack trace
func captureStackTrace() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

// WithFields creates a new logger with additional fields
func (l *StructuredLogger) WithFields(fields Fields) *ContextLogger {
	return &ContextLogger{
		logger: l,
		fields: fields,
	}
}

// ContextLogger wraps StructuredLogger with additional context fields
type ContextLogger struct {
	logger *StructuredLogger
	fields Fields
}

// Debug logs a debug message with context fields
func (c *ContextLogger) Debug(ctx context.Context, message string, fields Fields) {
	mergedFields := c.mergeFields(fields)
	c.logger.Debug(ctx, message, mergedFields)
}

// Info logs an info message with context fields
func (c *ContextLogger) Info(ctx context.Context, message string, fields Fields) {
	mergedFields := c.mergeFields(fields)
	c.logger.Info(ctx, message, mergedFields)
}

// Warn logs a warning message with context fields
func (c *ContextLogger) Warn(ctx context.Context, message string, fields Fields) {
	mergedFields := c.mergeFields(fields)
	c.logger.Warn(ctx, message, mergedFields)
}

// Error logs an error message with context fields
func (c *ContextLogger) Error(ctx context.Context, message string, fields Fields, err error) {
	mergedFields := c.mergeFields(fields)
	c.logger.Error(ctx, message, mergedFields, err)
}

// Fatal logs a fatal message with context fields
func (c *ContextLogger) Fatal(ctx context.Context, message string, fields Fields, err error) {
	mergedFields := c.mergeFields(fields)
	c.logger.Fatal(ctx, message, mergedFields, err)
}

// mergeFields merges context fields with provided fields
func (c *ContextLogger) mergeFields(fields Fields) Fields {
	merged := make(Fields)

	// Copy context fields
	for k, v := range c.fields {
		merged[k] = v
	}

	// Override with provided fields
	for k, v := range fields {
		merged[k] = v
	}

	return merged
}
