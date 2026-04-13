package logger

import (
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

var levelFromString = map[string]LogLevel{
	"debug": DEBUG, "info": INFO, "warn": WARN, "error": ERROR, "fatal": FATAL,
}

// 全局 logger 实例
var globalLogger *Logger

type Logger struct {
	level LogLevel
	zl    zerolog.Logger
}

func NewLogger(level string) *Logger {
	logLevel := INFO
	if l, ok := levelFromString[strings.ToLower(level)]; ok {
		logLevel = l
	}
	zl := log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "2006-01-02 15:04:05.000"}).With().Timestamp().Logger()
	logger := &Logger{level: logLevel, zl: zl}
	// 设置全局实例
	if globalLogger == nil {
		globalLogger = logger
	}
	return logger
}

// GetLogger 获取全局 logger
func GetLogger() *Logger {
	if globalLogger == nil {
		globalLogger = NewLogger("info")
	}
	return globalLogger
}

// L 快捷方式获取全局 logger
func L() *Logger {
	return GetLogger()
}

func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}
	msg := fmt.Sprintf(format, args...)
	switch level {
	case DEBUG:
		l.zl.Debug().Msg(msg)
	case INFO:
		l.zl.Info().Msg(msg)
	case WARN:
		l.zl.Warn().Msg(msg)
	case ERROR:
		l.zl.Error().Msg(msg)
	case FATAL:
		l.zl.Fatal().Msg(msg) // 使用 Fatal() 确保程序退出
	}
}

func (l *Logger) Debug(format string, args ...interface{}) { l.log(DEBUG, format, args...) }
func (l *Logger) Info(format string, args ...interface{})  { l.log(INFO, format, args...) }
func (l *Logger) Warn(format string, args ...interface{})  { l.log(WARN, format, args...) }
func (l *Logger) Error(format string, args ...interface{}) { l.log(ERROR, format, args...) }
func (l *Logger) Fatal(format string, args ...interface{}) { l.log(FATAL, format, args...) }

// WithField 带字段的日志
func (l *Logger) WithField(key string, value interface{}) *Logger {
	newZl := l.zl.With().Interface(key, value).Logger()
	return &Logger{level: l.level, zl: newZl}
}

// WithFields 带多个字段的日志
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	ctx := l.zl.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return &Logger{level: l.level, zl: ctx.Logger()}
}

// WithError 带错误的日志
func (l *Logger) WithError(err error) *Logger {
	newZl := l.zl.With().Err(err).Logger()
	return &Logger{level: l.level, zl: newZl}
}
