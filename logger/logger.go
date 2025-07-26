// Package logger
package logger

import (
	"io"
	"maps"
	"os"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Logger interface {
	Init(path string)
	InitMultiWriter(path string)

	Info(msg string)
	Warn(msg string)
	Error(msg string)
	Fatal(msg string)
	Debug(msg string)
	Panic(msg string)

	WithStr(key, value string) Logger
	WithBool(key string, value bool) Logger
	WithInt(key string, value int) Logger
	WithAny(key string, value any) Logger
}

type logger struct {
	base    zerolog.Logger
	context map[string]any
	path    string
}

func New() Logger {
	return &logger{
		path: "./logs/log.txt",
	}
}

func (l *logger) Init(path string) {
	logWriter := &lumberjack.Logger{
		Filename:   l.path,
		MaxSize:    5,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
	}

	l.base = zerolog.New(logWriter).
		With().
		Timestamp().
		Logger()
}

func (l *logger) InitMultiWriter(path string) {
	fileWriter := &lumberjack.Logger{
		Filename:   l.path,
		MaxSize:    5,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
	}

	multi := io.MultiWriter(os.Stdout, fileWriter)

	l.base = zerolog.New(multi).
		With().
		Timestamp().
		Logger()
}

func (l *logger) Info(msg string) {
	l.base.Info().Msg(msg)
}

func (l *logger) Warn(msg string) {
	l.base.Warn().Msg(msg)
}

func (l *logger) Fatal(msg string) {
	l.base.Fatal().Msg(msg)
}

func (l *logger) Error(msg string) {
	l.base.Error().Msg(msg)
}

func (l *logger) Panic(msg string) {
	l.base.Panic().Msg(msg)
}

func (l *logger) Debug(msg string) {
	l.base.Debug().Msg(msg)
}

func (l *logger) WithStr(key, value string) Logger {
	return l.withField(key, value)
}

func (l *logger) WithBool(key string, value bool) Logger {
	return l.withField(key, value)
}

func (l *logger) WithInt(key string, value int) Logger {
	return l.withField(key, value)
}

func (l *logger) WithAny(key string, value any) Logger {
	return l.withField(key, value)
}

func (l *logger) withField(key string, value any) Logger {
	newCtx := make(map[string]any, len(l.context)+1)
	maps.Copy(newCtx, l.context)
	newCtx[key] = value

	return &logger{
		base:    l.base,
		context: newCtx,
	}
}
