package auth

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type loginAttempt struct {
	failedCount int
	firstFailed time.Time
	blockedTill time.Time
}

type LoginRateLimiter struct {
	mu          sync.Mutex
	attempts    map[string]*loginAttempt
	maxFailed   int
	window      time.Duration
	blockPeriod time.Duration
}

func NewLoginRateLimiter(maxFailed int, window time.Duration, blockPeriod time.Duration) *LoginRateLimiter {
	return &LoginRateLimiter{
		attempts:    map[string]*loginAttempt{},
		maxFailed:   maxFailed,
		window:      window,
		blockPeriod: blockPeriod,
	}
}

func NewDefaultLoginRateLimiter() *LoginRateLimiter {
	return NewLoginRateLimiter(10, 5*time.Minute, 15*time.Minute)
}

func (l *LoginRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		if blocked, retryAfter := l.isBlocked(clientIP); blocked {
			retryAfterSeconds := int(retryAfter.Seconds())
			if retryAfterSeconds < 1 {
				retryAfterSeconds = 1
			}
			c.Header("Retry-After", strconv.Itoa(retryAfterSeconds))
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}

		c.Next()

		switch c.Writer.Status() {
		case http.StatusUnauthorized:
			l.recordFailed(clientIP)
		case http.StatusOK:
			l.clear(clientIP)
		}
	}
}

func (l *LoginRateLimiter) isBlocked(clientIP string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	attempt, ok := l.attempts[clientIP]
	if !ok {
		return false, 0
	}

	now := time.Now()
	if now.Sub(attempt.firstFailed) > l.window {
		delete(l.attempts, clientIP)
		return false, 0
	}

	if now.Before(attempt.blockedTill) {
		return true, time.Until(attempt.blockedTill)
	}

	return false, 0
}

func (l *LoginRateLimiter) recordFailed(clientIP string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	attempt, ok := l.attempts[clientIP]
	if !ok || now.Sub(attempt.firstFailed) > l.window {
		l.attempts[clientIP] = &loginAttempt{
			failedCount: 1,
			firstFailed: now,
		}
		return
	}

	attempt.failedCount++
	if attempt.failedCount >= l.maxFailed {
		attempt.blockedTill = now.Add(l.blockPeriod)
	}
}

func (l *LoginRateLimiter) clear(clientIP string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, clientIP)
}
