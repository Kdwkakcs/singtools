package log

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
)

// LogLevel represents the log level
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31;1m" // 亮红色
	ColorYellow = "\033[33;1m" // 亮黄色
	ColorBlue   = "\033[34;1m" // 亮蓝色
	ColorCyan   = "\033[36;1m" // 亮青色
	ColorGray   = "\033[37;2m" // 灰色用于调试信息
)

// Logger is a custom logger with a log level
type Logger struct {
	level  LogLevel
	logger *log.Logger
}

// NewLogger creates a new Logger with string level
func NewLogger(levelStr string) *Logger {
	level := INFO // 默认使用 INFO 级别
	switch levelStr {
	case "DEBUG":
		level = DEBUG
	case "INFO":
		level = INFO
	case "WARN":
		level = WARN
	case "ERROR":
		level = ERROR
	default:
		log.Printf(ColorYellow+"[WARN] Unknown log level: %s, using INFO"+ColorReset, levelStr)
	}

	return &Logger{
		level:  level,
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

// Debug logs a debug message with detailed information
func (l *Logger) Debug(msg string) {
	if l.level <= DEBUG {
		_, file, line, _ := runtime.Caller(1)
		l.logger.Printf(ColorGray+"[DEBUG]"+ColorReset+" [%s:%d] %s", filepath.Base(file), line, msg)
	}
}

// Info logs an info message with some details
func (l *Logger) Info(msg string) {
	if l.level <= INFO {
		_, file, _, _ := runtime.Caller(1)
		l.logger.Printf(ColorBlue+"[INFO]"+ColorReset+" [%s] %s", filepath.Base(file), msg)
	}
}

// Warn logs a warning message
func (l *Logger) Warn(msg string) {
	if l.level <= WARN {
		l.logger.Printf(ColorYellow+"[WARN]"+ColorReset+" %s", msg)
	}
}

// Error logs an error message
func (l *Logger) Error(msg string) {
	if l.level <= ERROR {
		_, file, line, _ := runtime.Caller(1)
		l.logger.Printf(ColorRed+"[ERROR]"+ColorReset+" [%s:%d] %s", filepath.Base(file), line, msg)
	}
}
