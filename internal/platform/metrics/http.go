// Package metrics предоставляет Prometheus-инструментацию сервиса Screener:
// HTTP-middleware, коллектор pgxpool и отдельный HTTP-сервер метрик.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace = "screener"

	// routeUnmatched используется как значение метки "route", когда chi не
	// сопоставил запрос ни одному шаблону маршрута (например, ответы 404).
	// Сырые пути запросов никогда не используются как метки, чтобы
	// кардинальность метрик оставалась ограниченной.
	routeUnmatched = "unmatched"
)

// HTTP содержит коллекторы Prometheus, описывающие входящий HTTP-трафик.
type HTTP struct {
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	requestsInFlight prometheus.Gauge
}

// NewHTTP создаёт и регистрирует коллекторы HTTP-метрик в reg.
// Если reg равен nil, используется prometheus.DefaultRegisterer.
func NewHTTP(reg prometheus.Registerer) *HTTP {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}
	factory := promauto.With(reg)

	return &HTTP{
		requestsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "http",
				Name:      "requests_total",
				Help:      "Total number of HTTP requests handled, partitioned by method, route pattern and status code.",
			},
			[]string{"method", "route", "status"},
		),
		requestDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "http",
				Name:      "request_duration_seconds",
				Help:      "Duration of HTTP requests in seconds, partitioned by method, route pattern and status code.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "route", "status"},
		),
		requestsInFlight: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "http",
				Name:      "requests_in_flight",
				Help:      "Number of HTTP requests currently being served.",
			},
		),
	}
}

// Middleware инструментирует обёрнутый обработчик: отслеживает активные
// запросы и по завершении запроса фиксирует его количество и длительность.
// В метках используется шаблон маршрута chi (например, /api/v1/users/{id}),
// а не сырой URL, поэтому запросы с разными параметрами пути попадают
// в одну серию.
func (m *HTTP) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.requestsInFlight.Inc()
		defer m.requestsInFlight.Dec()

		start := time.Now()

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		route := chi.RouteContext(r.Context()).RoutePattern()
		if route == "" {
			route = routeUnmatched
		}

		status := ww.Status()
		if status == 0 {
			status = http.StatusOK
		}

		labels := prometheus.Labels{
			"method": r.Method,
			"route":  route,
			"status": strconv.Itoa(status),
		}
		m.requestsTotal.With(labels).Inc()
		m.requestDuration.With(labels).Observe(time.Since(start).Seconds())
	})
}
