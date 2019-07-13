package main

import (
	"io"
	"strings"
	"time"

	echo "github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

// HTTPLogger returns a custom logging middleware that uses zerolog
func HTTPLogger(log *zerolog.Logger) echo.MiddlewareFunc {
	return getLogger(log, false, func(log *zerolog.Logger, s string) string {
		return s
	})
}

func getLogger(log *zerolog.Logger, errorsOnly bool, sanitizeURL func(*zerolog.Logger, string) string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			start := time.Now()
			err := next(ctx)
			if err != nil {
				ctx.Error(err)
			}
			stop := time.Now()
			delta := stop.Sub(start)
			code := ctx.Response().Status
			method := ctx.Request().Method
			if err != nil {
				log.
					Error().
					Err(err).
					Int("code", code).
					Str("method", method).
					Str("url", ctx.Request().URL.String()).
					Str("ip_address", ctx.RealIP()).
					Dur("request_duration", delta).
					Msg("request failed")
			} else if !errorsOnly {
				log.
					Info().
					Int("code", code).
					Str("method", method).
					Str("url", ctx.Request().URL.String()).
					Str("ip_address", ctx.RealIP()).
					Dur("request_duration", delta).
					Msg("request processed")
			}
			return err
		}
	}
}

// NewZeroLog creates a new zerolog logger
func NewZeroLog(writer io.Writer) *zerolog.Logger {
	zl := zerolog.New(writer).Output(zerolog.ConsoleWriter{Out: writer}).With().Timestamp().Logger()
	return &zl
}

// ParseLevel parses a level from string to log level
func ParseLevel(level string) zerolog.Level {
	switch strings.ToUpper(level) {
	case "FATAL":
		return zerolog.FatalLevel
	case "ERROR":
		return zerolog.ErrorLevel
	case "WARNING":
		return zerolog.WarnLevel
	case "INFO":
		return zerolog.InfoLevel
	case "DEBUG":
		return zerolog.DebugLevel
	default:
		return zerolog.DebugLevel
	}
}
