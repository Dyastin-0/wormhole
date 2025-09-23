// Package logger
package logger

import (
	"io"
	"os"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Init initializes the global logger with file output.
func Init(path string) {
	if path == "" {
		path = "./logs/log.txt"
	}

	logWriter := &lumberjack.Logger{
		Filename:   path,
		MaxSize:    5,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logger := zerolog.New(logWriter).With().Timestamp().Logger()

	zerolog.DefaultContextLogger = &logger
}

// InitMultiWriter initializes the global logger with both file and stdout.
func InitMultiWriter(path string) {
	if path == "" {
		path = "./logs/log.txt"
	}

	fileWriter := &lumberjack.Logger{
		Filename:   path,
		MaxSize:    5,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
	}

	multi := io.MultiWriter(os.Stdout, fileWriter)

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logger := zerolog.New(multi).With().Timestamp().Logger()

	zerolog.DefaultContextLogger = &logger
}
