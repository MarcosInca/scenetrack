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
	identifyPerMin = 6   // reconocimientos por minuto por IP
	identifyPerDay = 100 // reconocimientos totales por día (cuida la cuota de AudD)
	submitPerMin   = 3   // envíos por minuto por IP
	submitPerDay   = 300 // envíos totales por día
)

type visitor struct {
	lim  *rate.Limiter
	seen time.Time
}

// limiter limita los pedidos por IP y el total por día.
type limiter struct {
	mu         sync.Mutex
	visitors   map[string]*visitor
	perMin     int
	dayMax     int
	day        time.Time // día UTC en curso
	count      int       // pedidos aceptados hoy
	trustProxy bool
}

func newLimiter(perMin, dayMax int, trustProxy bool) *limiter {
	return &limiter{
		visitors: map[string]*visitor{}, perMin: perMin, dayMax: dayMax, trustProxy: trustProxy,
	}
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
	if l.count >= l.dayMax {
		return false
	}

	v, ok := l.visitors[ip]
	if !ok {
		every := rate.Every(time.Minute / time.Duration(l.perMin))
		v = &visitor{lim: rate.NewLimiter(every, l.perMin)}
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
