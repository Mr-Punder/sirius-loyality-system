package middleware

// import (
// 	"bytes"
// 	"compress/gzip"
// 	"io"
// 	"net/http"
// 	"net/http/httptest"
// 	"testing"

// 	"github.com/Mr-Punder/go-alerting-service/internal/handlers"
// 	"github.com/Mr-Punder/go-alerting-service/internal/logger"
// 	"github.com/Mr-Punder/go-alerting-service/internal/metrics"
// 	"github.com/Mr-Punder/go-alerting-service/internal/storage"

// 	"github.com/stretchr/testify/require"
// )

// type testLogger interface {
// 	Info(mes string)
// 	Errorf(str string, arg ...any)
// 	Error(mess string)
// 	Infof(str string, arg ...any)
// 	Debug(mess string)
// }

// func testRequest(t *testing.T, ts *httptest.Server, method, path string, sentbody *bytes.Buffer, sentheaders map[string]string, log testLogger) *http.Response {
// 	req, err := http.NewRequest(method, ts.URL+path, sentbody)
// 	require.NoError(t, err)

// 	for key, value := range sentheaders {
// 		req.Header.Set(key, value)
// 	}

// 	resp, err := ts.Client().Do(req)
// 	log.Info("request sent")
// 	if err != nil {
// 		log.Error(err.Error())
// 	}
// 	//require.NoError(t, err)
// 	log.Infof("finish request with status %s", resp.Status)

// 	return resp
// }

// func TestGzipCompressor(t *testing.T) {

// 	simpleValue := func() *float64 { var v = 1.5; return &v }
// 	simpleDelta := func() *int64 { var d int64 = 2; return &d }

// 	metrics := map[string]metrics.Metrics{
// 		"gaugeMetric": {
// 			ID:    "gaugeMetric",
// 			MType: "gauge",
// 			Value: simpleValue(),
// 		},
// 		"counterMetric": {
// 			ID:    "counterMetric",
// 			MType: "counter",
// 			Delta: simpleDelta(),
// 		},
// 	}

// 	Log, err := logger.NewZapLogger("info", "./log.txt")
// 	require.NoError(t, err)
// 	stor, err := storage.NewMemStorage(metrics, false, "", Log)
// 	require.NoError(t, err)
// 	// Log, err := logger.NewLogZap("info", "./log.txt", "stderr")

// 	comp := NewGzipCompressor(Log)
// 	hlog := NewHTTPLoger(Log)

// 	ts := httptest.NewServer(hlog.HTTPLogHandler(comp.CompressHandler(handlers.NewMetricRouter(stor, Log))))
// 	defer ts.Close()

// 	requestBody := "{\"id\":\"g\",\"type\":\"gauge\",\"value\":5.2}"
// 	wantBody := "{\"id\":\"g\",\"type\":\"gauge\",\"value\":5.2}"
// 	// compressedBody := `\x1f\x8b\b\x00\x00\x00\x00\x00\x00\xff\xaaV\xcaLQ\xb2RJW\xd2Q*\xa9,H\x051\x13K\xd3S\x95t\x94\xca\x12sJS\x95\xacL\xf5\x8cj\x01\x01\x00\x00\xff\xffR\x1b\xc1\xe9%\x00\x00\x00`

// 	t.Run("sending_gzip", func(t *testing.T) {
// 		Log.Info("sending test started")

// 		buf := bytes.NewBuffer(nil)
// 		zb := gzip.NewWriter(buf)
// 		_, err := zb.Write([]byte(requestBody))
// 		require.NoError(t, err)
// 		err = zb.Close()
// 		require.NoError(t, err)

// 		headers := map[string]string{
// 			"Content-Type":     "application/json",
// 			"Content-Encoding": "gzip",
// 		}

// 		resp := testRequest(t, ts, http.MethodPost, "/update", buf, headers, Log)

// 		require.Equal(t, http.StatusOK, resp.StatusCode)
// 		defer resp.Body.Close()
// 		strbody, err := io.ReadAll(resp.Body)
// 		require.NoError(t, err)

// 		Log.Infof("Recieved str: %s", strbody)

// 		require.JSONEq(t, wantBody, string(strbody))

// 	})
// 	t.Run("receiving_gzip", func(t *testing.T) {
// 		buf := bytes.NewBufferString(requestBody)
// 		headers := map[string]string{
// 			"Content-Type":    "application/json",
// 			"Accept-Encoding": "gzip",
// 		}
// 		resp := testRequest(t, ts, http.MethodPost, "/update", buf, headers, Log)
// 		require.Equal(t, http.StatusOK, resp.StatusCode)

// 		strbody, err := io.ReadAll(resp.Body)
// 		resp.Body.Close()
// 		require.NoError(t, err)
// 		Log.Infof("Recieved str: %s", strbody)

// 		zr, err := gzip.NewReader(bytes.NewReader(strbody))
// 		require.NoError(t, err)

// 		b, err := io.ReadAll(zr)
// 		require.NoError(t, err)
// 		Log.Infof("Decompressed response: %s", b)
// 		Log.Infof("Want response: %s", wantBody)

// 		require.JSONEq(t, wantBody, string(b))
// 		// require.Equal(t, wantBody, string(strbody))

// 	})
// }
