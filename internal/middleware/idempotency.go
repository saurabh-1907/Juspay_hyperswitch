package middleware

import (
	"bytes"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/saurabh/payment-routing-layer/internal/db"
)

type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w responseBodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func IdempotencyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		idempotencyKey := c.GetHeader("Idempotency-Key")

		if idempotencyKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Idempotency-Key header is required"})
			c.Abort()
			return
		}

		key := "idempotency:" + idempotencyKey

		// Check if it exists in Redis
		cachedResponse, err := db.RedisClient.Get(db.Ctx, key).Result()
		if err == nil && cachedResponse != "" {
			// Found cached response
			c.Data(http.StatusOK, "application/json", []byte(cachedResponse))
			c.Abort()
			return
		} else if err != nil && err != redis.Nil {
			// Some redis error
			c.Next()
			return
		}

		// Not found, capture the response
		w := &responseBodyWriter{body: &bytes.Buffer{}, ResponseWriter: c.Writer}
		c.Writer = w

		c.Next()

		// Cache only successful responses (2xx)
		if c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
			err = db.RedisClient.SetEx(db.Ctx, key, w.body.String(), 24*time.Hour).Err()
			if err != nil {
				// Log error, but don't fail the request
				_ = err
			}
		}
	}
}
