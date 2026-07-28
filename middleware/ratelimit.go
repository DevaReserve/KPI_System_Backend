package middleware

import (
	"KPI_System_Backend/global_var"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// IPRateLimiter stores the rate limiter for each IP
type IPRateLimiter struct {
	ips sync.Map
	r   rate.Limit
	b   int
}

// NewIPRateLimiter creates a new rate limiter that will allow r requests per second with a burst of b
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		r: r,
		b: b,
	}
}

// GetLimiter returns the rate limiter for the given IP or creates a new one
func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	limiter, exists := i.ips.Load(ip)
	if !exists {
		newLimiter := rate.NewLimiter(i.r, i.b)
		i.ips.Store(ip, newLimiter)
		return newLimiter
	}

	return limiter.(*rate.Limiter)
}

// Global variable for the auth rate limiter
// Example: 10 requests per minute = 10 / 60 = 0.166 requests per second
// We can use rate.Every(time.Minute / 10) to be precise, with a burst of 10.
var authLimiter = NewIPRateLimiter(rate.Every(time.Minute/10), 10)

// RateLimitMiddleware applies rate limiting based on client IP
func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := authLimiter.GetLimiter(ip)

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, global_var.ResponseFormat{
				Status:  http.StatusTooManyRequests,
				Message: "Terlalu banyak request. Silakan coba beberapa saat lagi.",
				Data:    nil,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
