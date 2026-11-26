package httpmw_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Screener-masterskaya/Screener_backend/internal/httpmw"
)

// serveRequestID runs one request through the middleware and returns
// the ID seen by the handler and the ID sent back to the client.
func serveRequestID(header string) (inContext, inResponse string) {
	handler := httpmw.RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		inContext = httpmw.RequestIDFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if header != "" {
		req.Header.Set(httpmw.HeaderRequestID, header)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	return inContext, rec.Header().Get(httpmw.HeaderRequestID)
}

func TestRequestID(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		wantKept bool
	}{
		{name: "valid ID is kept", header: "abc-123_XYZ", wantKept: true},
		{name: "max length is kept", header: strings.Repeat("a", 64), wantKept: true},
		{name: "missing header", header: ""},
		{name: "too long", header: strings.Repeat("a", 65)},
		{name: "space", header: "abc 123"},
		{name: "line break", header: "abc\nlevel=ERROR"},
		{name: "non-ASCII", header: "идентификатор"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inContext, inResponse := serveRequestID(tt.header)

			require.NotEmpty(t, inContext)
			require.Equal(t, inContext, inResponse)
			if tt.wantKept {
				require.Equal(t, tt.header, inContext)
			} else {
				require.NotEqual(t, tt.header, inContext)
			}
		})
	}
}

func TestRequestID_GeneratesUniqueIDs(t *testing.T) {
	first, _ := serveRequestID("")
	second, _ := serveRequestID("")

	require.NotEqual(t, first, second)
}

func TestRequestIDFromContext_Missing(t *testing.T) {
	require.Empty(t, httpmw.RequestIDFromContext(t.Context()))
}
