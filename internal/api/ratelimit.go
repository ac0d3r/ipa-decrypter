package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	searchRateMax   = 15
	versionRateMax  = 10
	coreAPIRateMax  = 5
	registerRateMax = 5
)

type rateLimiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{hits: map[string][]time.Time{}}
}

func (l *rateLimiter) allow(key string, limit int, window time.Duration) bool {
	now := time.Now()
	cutoff := now.Add(-window)

	l.mu.Lock()
	defer l.mu.Unlock()

	times := l.hits[key]
	n := 0
	for _, t := range times {
		if t.After(cutoff) {
			times[n] = t
			n++
		}
	}
	times = times[:n]
	if n == 0 {
		delete(l.hits, key)
	}
	if len(l.hits) > 512 {
		for k, ts := range l.hits {
			if k == key {
				continue
			}
			alive := false
			for _, t := range ts {
				if t.After(cutoff) {
					alive = true
					break
				}
			}
			if !alive {
				delete(l.hits, k)
			}
		}
	}
	if len(times) >= limit {
		l.hits[key] = times
		return false
	}
	l.hits[key] = append(times, now)
	return true
}

func (s *Server) rateLimit(limit int) gin.HandlerFunc {
	return func(c *gin.Context) {
		if s.limiter == nil {
			s.limiter = newRateLimiter()
		}
		key := c.FullPath() + "|" + c.ClientIP()
		if u := currentUser(c); u.ID != "" {
			key = c.FullPath() + "|" + u.ID
		}
		if !s.limiter.allow(key, limit, time.Minute) {
			writeErr(c, errStatus(http.StatusTooManyRequests, "too many requests"))
			c.Abort()
			return
		}
		c.Next()
	}
}
