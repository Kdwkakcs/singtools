package log

import (
	"log"
	"os"
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
	ColorRed    = "\033[31m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
)

// Logger is a custom logger with a log level
type Logger struct {
	level  LogLevel
	logger *log.Logger
}

// NewLogger creates a new Logger
func NewLogger(level LogLevel) *Logger {
	return &Logger{
		level:  level,
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string) {
	if l.level <= DEBUG {
		l.logger.Println(ColorCyan + "[DEBUG] " + msg + ColorReset)
	}
}

// Info logs an info message
func (l *Logger) Info(msg string) {
	if l.level <= INFO {
		l.logger.Println(ColorBlue + "[INFO] " + msg + ColorReset)
	}
}

// Warn logs a warning message
func (l *Logger) Warn(msg string) {
	if l.level <= WARN {
		l.logger.Println(ColorYellow + "[WARN] " + msg + ColorReset)
	}
}

// Error logs an error message
func (l *Logger) Error(msg string) {
	if l.level <= ERROR {
		l.logger.Println(ColorRed + "[ERROR] " + msg + ColorReset)
	}
}
