package log

import (
	"fmt"
	"log/slog"

	"github.com/pressly/goose/v3"
)

type gooseLogger struct {
	logger *slog.Logger
}

var _ goose.Logger = (*gooseLogger)(nil)

func NewGooseLogger(logger *slog.Logger) *gooseLogger {
	return &gooseLogger{logger}
}

func (l *gooseLogger) Fatalf(format string, v ...any) {
	l.logger.Error(fmt.Sprintf(format, v...))
	panic(fmt.Sprintf(format, v...))
}

func (l *gooseLogger) Printf(format string, v ...any) {
	l.logger.Info(fmt.Sprintf(format, v...))
}
