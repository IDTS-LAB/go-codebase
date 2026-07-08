package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

type idempotencyKey struct{}

func IdempotencyKeyFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(idempotencyKey{}).(string); ok {
		return v
	}
	return ""
}

func Idempotency(rdb *redis.Client, ttl time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}

			key := r.Header.Get("Idempotency-Key")
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			redisKey := fmt.Sprintf("idempotency:%s", key)
			ctx := r.Context()

			cached, err := rdb.Get(ctx, redisKey).Result()
			if err == nil && cached != "" {
				var entry idempotencyEntry
				if json.Unmarshal([]byte(cached), &entry) == nil {
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("Idempotency-Key", key)
					w.WriteHeader(entry.StatusCode)
					w.Write(entry.Body)
					return
				}
			}

			capture := &idempotencyCapture{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(capture, r)

			if capture.statusCode >= 200 && capture.statusCode < 300 {
				entry := idempotencyEntry{
					StatusCode: capture.statusCode,
					Body:       capture.body.Bytes(),
				}
				if data, err := json.Marshal(entry); err == nil {
					rdb.Set(ctx, redisKey, data, ttl)
				}
			}

			w.Header().Set("Idempotency-Key", key)
		})
	}
}

type idempotencyEntry struct {
	StatusCode int    `json:"status_code"`
	Body       []byte `json:"body"`
}

type idempotencyCapture struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (c *idempotencyCapture) WriteHeader(code int) {
	c.statusCode = code
	c.ResponseWriter.WriteHeader(code)
}

func (c *idempotencyCapture) Write(b []byte) (int, error) {
	c.body.Write(b)
	return c.ResponseWriter.Write(b)
}
