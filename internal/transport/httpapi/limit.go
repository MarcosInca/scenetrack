package httpapi

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	perMinute = 6   // pedidos por minuto por IP
	dailyMax  = 100 // pedidos totales por día (cuida la cuota de AudD)
)

type visitor struct {
	lim  *rate.Limiter
	seen time.Time
}

// limiter limita los pedidos por IP y el total por día.
type limiter struct {
	mu         sync.Mutex
	visitors   map[string]*visitor
	day        time.Time // día UTC en curso
	count      int       // pedidos aceptados hoy
	trustProxy bool
}

func newLimiter(trustProxy bool) *limiter {
	return &limiter{visitors: map[string]*visitor{}, trustProxy: trustProxy}
}

// wrap deja pasar el pedido o responde 429.
func (l *limiter) wrap(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(l.clientIP(r)) {
			w.Header().Set("Retry-After", "60")
			writeError(w, http.StatusTooManyRequests, "demasiados pedidos, probá en un rato")
			return
		}
		next(w, r)
	})
}

func (l *limiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()

	if today := now.UTC().Truncate(24 * time.Hour); !today.Equal(l.day) {
		l.day, l.count = today, 0 // nuevo día: reinicia el contador
	}
	if l.count >= dailyMax {
		return false
	}

	v, ok := l.visitors[ip]
	if !ok {
		v = &visitor{lim: rate.NewLimiter(rate.Every(time.Minute/perMinute), perMinute)}
		l.visitors[ip] = v
	}
	v.seen = now
	if !v.lim.Allow() {
		return false
	}
	l.count++

	// Limpieza: con muchas IPs guardadas, borra las inactivas.
	if len(l.visitors) > 1000 {
		for k, old := range l.visitors {
			if now.Sub(old.seen) > 10*time.Minute {
				delete(l.visitors, k)
			}
		}
	}
	return true
}

func (l *limiter) clientIP(r *http.Request) string {
	if l.trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			return strings.TrimSpace(parts[len(parts)-1]) // lo agrega el proxy del hosting
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
