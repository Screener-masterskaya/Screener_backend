package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ВНИМАНИЕ: маршруты /api/v1/things/{id} ниже — МОКИ. Они нужны только для
// проверки metrics-middleware на параметризованном шаблоне маршрута, потому
// что реальное приложение пока не имеет маршрутов с параметрами пути.
// Замените их на реальные маршруты после появления доменного API.

// newMockInstrumentedRouter собирает chi-роутер с metrics-middleware и
// моковым параметризованным маршрутом. Возвращает роутер и реестр метрик,
// чтобы тесты могли собрать и проверить получившиеся серии.
func newMockInstrumentedRouter(t *testing.T) (http.Handler, *prometheus.Registry) {
	t.Helper()

	reg := prometheus.NewRegistry()
	httpMetrics := NewHTTP(reg)

	router := chi.NewRouter()
	router.Use(httpMetrics.Middleware)
	// Моковый маршрут: заглушка будущих параметризованных эндпоинтов API.
	router.Get("/api/v1/things/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"` + chi.URLParam(r, "id") + `"}`))
	})

	return router, reg
}

// httptestRequest выполняет один запрос к handler и возвращает рекордер.
func httptestRequest(t *testing.T, handler http.Handler, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// gatherText собирает метрики из реестра через текстовый формат
// экспозиции Prometheus и возвращает тело ответа.
func gatherText(t *testing.T, gatherer prometheus.Gatherer) string {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	Handler(gatherer).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body, err := io.ReadAll(rec.Result().Body)
	require.NoError(t, err)
	return string(body)
}

// countSeries считает строки экспозиции (не комментарии), относящиеся
// к указанному семейству метрик (точное совпадение имени, за которым
// следует "{" или " ").
func countSeries(body, family string) int {
	count := 0
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		if strings.HasPrefix(line, family+"{") || strings.HasPrefix(line, family+" ") {
			count++
		}
	}
	return count
}

func TestMiddlewareRecordsCounterAndHistogram(t *testing.T) {
	router, reg := newMockInstrumentedRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/things/42", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	body := gatherText(t, reg)

	// Счётчик и гистограмма должны быть видны с метками низкой
	// кардинальности: method, шаблон маршрута и status — без сырого URL
	// и без значения параметра пути.
	assert.Contains(t, body,
		`screener_http_requests_total{method="GET",route="/api/v1/things/{id}",status="200"} 1`)
	assert.Contains(t, body,
		`screener_http_request_duration_seconds_count{method="GET",route="/api/v1/things/{id}",status="200"} 1`)
	assert.NotContains(t, body, "things/42")
}

func TestMiddlewareSameRouteDifferentIDsShareSingleSeries(t *testing.T) {
	router, reg := newMockInstrumentedRouter(t)

	for _, id := range []string{"1", "2", "abc-def"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/things/"+id, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
	}

	body := gatherText(t, reg)

	// Все запросы с разными значениями параметра пути должны схлопнуться
	// в одну серию, определяемую шаблоном маршрута.
	assert.Equal(t, 1, countSeries(body, "screener_http_requests_total"))
	assert.Contains(t, body,
		`screener_http_requests_total{method="GET",route="/api/v1/things/{id}",status="200"} 3`)
	assert.Equal(t, 1, countSeries(body, "screener_http_request_duration_seconds_count"))
}

func TestMiddlewareNotFoundUsesUnmatchedRouteLabel(t *testing.T) {
	router, reg := newMockInstrumentedRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/no/such/route", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNotFound, rec.Code)

	body := gatherText(t, reg)

	// Запросы без совпавшего маршрута тоже не должны попадать
	// в метки сырым путём.
	assert.Contains(t, body,
		`screener_http_requests_total{method="GET",route="unmatched",status="404"} 1`)
	assert.NotContains(t, body, "/no/such/route")
}

func TestMiddlewareTracksInFlightRequests(t *testing.T) {
	reg := prometheus.NewRegistry()
	httpMetrics := NewHTTP(reg)

	entered := make(chan struct{})
	release := make(chan struct{})

	// Моковый обработчик: блокируется до release, чтобы запрос оставался активным.
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
		w.WriteHeader(http.StatusOK)
	})

	handler := httpMetrics.Middleware(mockHandler)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}()

	<-entered
	assert.Equal(t, 1.0, testutil.ToFloat64(httpMetrics.requestsInFlight))

	close(release)
	wg.Wait()
	assert.Equal(t, 0.0, testutil.ToFloat64(httpMetrics.requestsInFlight))
}

func TestMiddlewareRecordsErrorStatus(t *testing.T) {
	reg := prometheus.NewRegistry()
	httpMetrics := NewHTTP(reg)

	// Моковый обработчик: всегда завершается ошибкой 500.
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})

	handler := httpMetrics.Middleware(mockHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusInternalServerError, rec.Code)

	body := gatherText(t, reg)
	assert.Contains(t, body, `status="500"`)
}

func TestMiddlewareRecordsRecoveredPanicAs500(t *testing.T) {
	reg := prometheus.NewRegistry()
	httpMetrics := NewHTTP(reg)

	// Порядок middleware повторяет боевой роутер: метрики перед Recoverer,
	// чтобы ответ 500 после восстановленной паники попадал в метрики.
	router := chi.NewRouter()
	router.Use(httpMetrics.Middleware)
	router.Use(chimiddleware.Recoverer)
	// Моковый маршрут: хендлер всегда паникует.
	router.Get("/api/v1/things/{id}", func(w http.ResponseWriter, r *http.Request) {
		panic("mock panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/things/42", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusInternalServerError, rec.Code)

	body := gatherText(t, reg)

	// Паника должна быть учтена как 500 с шаблоном маршрута, а gauge
	// активных запросов — вернуться к нулю несмотря на панику.
	assert.Contains(t, body,
		`screener_http_requests_total{method="GET",route="/api/v1/things/{id}",status="500"} 1`)
	assert.Contains(t, body,
		`screener_http_request_duration_seconds_count{method="GET",route="/api/v1/things/{id}",status="500"} 1`)
	assert.Equal(t, 0.0, testutil.ToFloat64(httpMetrics.requestsInFlight))
}
