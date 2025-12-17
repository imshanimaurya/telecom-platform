package logger

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const headerRequestID = "X-Request-Id"

// Middleware returns a Gin middleware that injects request_id and logs request summaries.
func Middleware(l *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		rid := c.GetHeader(headerRequestID)
		if rid == "" {
			rid = uuid.NewString()
		}
		c.Writer.Header().Set(headerRequestID, rid)

		// attach request_id logger
		reqLogger := l.With("request_id", rid)
		c.Set("logger", reqLogger)

		c.Next()

		dur := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		attrs := []any{
			"method", method,
			"path", path,
			"status", status,
			"duration_ms", float64(dur.Milliseconds()),
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, "errors", c.Errors.String())
			reqLogger.Error("request", attrs...)
			return
		}
		reqLogger.Info("request", attrs...)
	}
}

// FromGin pulls the request-scoped logger from Gin context.
func FromGin(c *gin.Context) *slog.Logger {
	if v, ok := c.Get("logger"); ok {
		if l, ok := v.(*slog.Logger); ok && l != nil {
			return l
		}
	}
	return slog.Default()
}
