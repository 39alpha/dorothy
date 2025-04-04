package log

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/39alpha/dorothy/core"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	"gorm.io/gorm/logger"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
}

type Event struct {
	*zerolog.Event
	Logger *Logger
}

func (event *Event) next(e *zerolog.Event) *Event {
	return &Event{
		Event:  e,
		Logger: event.Logger,
	}
}

func (event *Event) Send() Logger {
	event.Event.Send()
	return *event.Logger
}

func (event *Event) Redirect(path string) *Event {
	return event.next(event.Str("redirect", path))
}

func (event *Event) Stack() *Event {
	return event.next(event.Event.Stack())
}

func (event *Event) Err(err error) *Event {
	return event.next(event.Event.Err(err))
}

func (event *Event) Mark() *Event {
	ctx := event.Logger.ctx
	if ctx != nil {
		requestId, _ := ctx.Locals("RequestID").(string)
		return event.next(event.Str("request_id", requestId))
	}
	return event
}

func (event *Event) Handler(handler string) *Event {
	return event.next(event.Str("handler", handler))
}

func (event *Event) Renderer(handler, renderer string) *Event {
	return event.next(event.Handler(handler).Str("renderer", renderer))
}

func (event *Event) Recover(handler string) *Event {
	return event.next(event.Handler(handler).Bool("recovery", true))
}

type Logger struct {
	zerolog.Logger
	ctx *fiber.Ctx
}

func (l *Logger) Panic(err error) *Event {
	e := &Event{
		Event:  l.Logger.Panic(),
		Logger: l,
	}
	return e.Stack().Mark().Err(err)
}

func (l *Logger) Fatal(err error) *Event {
	e := &Event{
		Event:  l.Logger.Fatal(),
		Logger: l,
	}
	return e.Stack().Mark().Err(err)
}

func (l *Logger) Error(err error) *Event {
	e := &Event{
		Event:  l.Logger.Error(),
		Logger: l,
	}
	return e.Stack().Mark().Err(err)
}

func (l *Logger) Info() *Event {
	e := &Event{
		Event:  l.Logger.Info(),
		Logger: l,
	}
	return e.Mark()
}

func (l *Logger) Debug() *Event {
	e := &Event{
		Event:  l.Logger.Debug(),
		Logger: l,
	}
	return e.Mark()
}

func (l *Logger) Trace() *Event {
	e := &Event{
		Event:  l.Logger.Trace(),
		Logger: l,
	}
	return e.Mark()
}

func (l Logger) RequestContext(ctx *fiber.Ctx) Logger {
	l.ctx = ctx
	return l
}

func NewLogger(config core.LoggerConfig) (Logger, error) {
	level := zerolog.InfoLevel
	switch config.Level {
	case core.LogPanic:
		level = zerolog.PanicLevel
	case core.LogFatal:
		level = zerolog.FatalLevel
	case core.LogError:
		level = zerolog.ErrorLevel
	case core.LogWarn:
		level = zerolog.WarnLevel
	case core.LogInfo:
		level = zerolog.InfoLevel
	case core.LogDebug:
		level = zerolog.DebugLevel
	case core.LogTrace:
		level = zerolog.TraceLevel
	case core.LogDisable:
		level = zerolog.Disabled
	}

	logger := Logger{
		Logger: zerolog.
			New(zerolog.ConsoleWriter{Out: os.Stderr}).
			With().
			Timestamp().
			Logger().
			Level(level),
		ctx: nil,
	}

	if config.Path != "" {
		w, err := os.OpenFile(config.Path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0700)
		if err != nil {
			logger.
				Error(err).
				Str("path", config.Path).
				Msg("falling back to stderr logs")
			return logger, err
		}

		logger = Logger{
			Logger: zerolog.New(w).Level(level),
			ctx:    nil,
		}
	}

	return logger, nil
}

func NewGormLogger(config core.LoggerConfig) (logger.Interface, error) {
	conf := logger.Config{
		SlowThreshold:             50 * time.Millisecond,
		LogLevel:                  logger.Warn,
		IgnoreRecordNotFoundError: false,
		Colorful:                  false,
	}

	switch config.Level {
	case core.LogPanic:
		conf.LogLevel = logger.Error
	case core.LogFatal:
		conf.LogLevel = logger.Error
	case core.LogError:
		conf.LogLevel = logger.Error
	case core.LogWarn:
		conf.LogLevel = logger.Warn
	case core.LogInfo:
		conf.LogLevel = logger.Info
	case core.LogDebug:
		conf.LogLevel = logger.Info
	case core.LogTrace:
		conf.LogLevel = logger.Info
	case core.LogDisable:
		conf.LogLevel = logger.Silent
	}

	newLogger := logger.New(log.New(os.Stderr, "\r\n", log.LstdFlags), conf)
	if config.Path != "" {
		w, err := os.OpenFile(config.Path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0700)
		if err != nil {
			newLogger.Warn(context.Background(), "falling back to stderr logs")
			return newLogger, err
		}

		newLogger = logger.New(log.New(w, "\r\n", log.LstdFlags), conf)
	}

	return newLogger, nil
}
