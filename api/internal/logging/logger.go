package logging

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	appctx "github.com/dokkiitech/ashiato/api/internal/middleware"
)

func NewLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func RequestMiddleware(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			traceID := c.Request().Header.Get("X-Trace-Id")
			if traceID == "" {
				traceID = uuid.NewString()
			}

			ctx := appctx.WithTraceID(c.Request().Context(), traceID)
			ctx = appctx.WithRequestIP(ctx, c.RealIP())
			c.SetRequest(c.Request().WithContext(ctx))
			c.Response().Header().Set("X-Trace-Id", traceID)

			err := next(c)
			if err != nil {
				c.Error(err)
			}

			fields := []any{
				"timestamp", time.Now().UTC().Format(time.RFC3339),
				"level", "INFO",
				"service", "api",
				"trace_id", traceID,
				"method", c.Request().Method,
				"path", c.Path(),
				"status", c.Response().Status,
				"latency_ms", time.Since(start).Milliseconds(),
				"ip", c.RealIP(),
				"user_agent", c.Request().UserAgent(),
			}

			if actor, ok := appctx.ActorFromContext(c.Request().Context()); ok {
				fields = append(fields,
					"user_id", actor.UserID.String(),
					"tenant_id", actor.OrganizationID.String(),
				)
			}

			logger.LogAttrs(context.Background(), slog.LevelInfo, "request completed", toAttrs(fields...)...)
			return err
		}
	}
}

func toAttrs(kv ...any) []slog.Attr {
	attrs := make([]slog.Attr, 0, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		key, _ := kv[i].(string)
		attrs = append(attrs, slog.Any(key, kv[i+1]))
	}
	return attrs
}
