package middlewares

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RateLimiter aplica token bucket por IP nas rotas públicas (spec §6).
// Simples e em memória: suficiente para uma instância única atrás do proxy.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	rps     float64
	burst   float64
}

type tokenBucket struct {
	tokens float64
	last   time.Time
}

func NewRateLimiter(rps, burst float64) *RateLimiter {
	limiter := &RateLimiter{
		buckets: make(map[string]*tokenBucket),
		rps:     rps,
		burst:   burst,
	}
	go limiter.cleanupLoop()
	return limiter
}

func (l *RateLimiter) Limit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(clientIP(r)) {
			w.Header().Set("Retry-After", "1")
			http.Error(w, `{"error": "Muitas requisições — tente novamente em instantes"}`, http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}

func (l *RateLimiter) allow(ip string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	bucket, ok := l.buckets[ip]
	if !ok {
		l.buckets[ip] = &tokenBucket{tokens: l.burst - 1, last: now}
		return true
	}

	bucket.tokens += now.Sub(bucket.last).Seconds() * l.rps
	if bucket.tokens > l.burst {
		bucket.tokens = l.burst
	}
	bucket.last = now

	if bucket.tokens < 1 {
		return false
	}
	bucket.tokens--
	return true
}

// cleanupLoop descarta buckets ociosos para o mapa não crescer indefinidamente
func (l *RateLimiter) cleanupLoop() {
	for range time.Tick(5 * time.Minute) {
		cutoff := time.Now().Add(-10 * time.Minute)
		l.mu.Lock()
		for ip, bucket := range l.buckets {
			if bucket.last.Before(cutoff) {
				delete(l.buckets, ip)
			}
		}
		l.mu.Unlock()
	}
}

// clientIP resolve o IP real atrás do reverse proxy da VPS
func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	if real := r.Header.Get("X-Real-Ip"); real != "" {
		return real
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
