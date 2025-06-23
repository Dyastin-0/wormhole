package wormhole

import (
	"io"
	"os"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Logger interface {
	Init(name, path string)
	InitMultiWriter(name, path string)

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
	name    string
	path    string
}

func NewLogger() Logger {
	return &logger{
		path: "./logs/log.txt",
	}
}

func (l *logger) Init(name, path string) {
	if l.name != "" {
		l.name = name
	}

	if path != "" {
		l.path = path
	}

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
		Str("name", name).
		Logger()
}

func (l *logger) InitMultiWriter(name, path string) {
	if l.name != "" {
		l.name = name
	}

	if path != "" {
		l.path = path
	}

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
		Str("name", name).
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
	newCtx := make(map[string]interface{}, len(l.context)+1)
	for k, v := range l.context {
		newCtx[k] = v
	}
	newCtx[key] = value

	return &logger{
		base:    l.base,
		context: newCtx,
	}
}
