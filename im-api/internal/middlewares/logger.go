package middlewares

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zeromicro/go-zero/core/logc"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		latency := time.Since(start)
		ctx := c.Request.Context()

		if len(c.Errors) > 0 {
			logc.Errorf(ctx, "%s %s %d %s %v ua=%s errors=%s",
				c.Request.Method,
				c.Request.URL.Path,
				c.Writer.Status(),
				latency,
				c.ClientIP(),
				c.Request.UserAgent(),
				c.Errors.String(),
			)
			return
		}

		logc.Infof(ctx, "%s %s %d %s %s ua=%s",
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			latency,
			c.ClientIP(),
			c.Request.UserAgent(),
		)
	}
}
