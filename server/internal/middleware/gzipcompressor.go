package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/MrPunder/sirius-loyality-system/internal/gzipcomp"
	"github.com/MrPunder/sirius-loyality-system/internal/logger"
)

// GzipCompressor is middleware compressor
type GzipCompressor struct {
	log logger.Logger
}

func NewGzipCompressor(log logger.Logger) *GzipCompressor {
	return &GzipCompressor{
		log: log,
	}
}

func (c *GzipCompressor) CompressHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.log.Info("Entered compressor")
		ow := w

		// Пропускаем сжатие для статических файлов админки
		if strings.HasPrefix(r.URL.Path, "/admin/") {
			next.ServeHTTP(w, r)
			return
		}

		headers := r.Header
		c.log.Info(fmt.Sprintf("Headers:  %v", headers))

		contentEncoding := r.Header.Get("Content-Encoding")
		c.log.Info(fmt.Sprintf("Content-Encoding = %s", contentEncoding))
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if sendsGzip {
			c.log.Info(fmt.Sprintf("Detected %s compression", "gzip"))

			var err error
			r.Body, err = gzipcomp.NewGzipCompressReader(r.Body)
			if err != nil {
				c.log.Error(fmt.Sprintf("Error setting read buffer for %s compressor", "gzip"))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			defer r.Body.Close()

		}

		accepEncoding := r.Header.Values("Accept-Encoding")
		c.log.Info(fmt.Sprintf("Accept-Encoding: %v", accepEncoding))

		supportGzip := false

		for _, value := range accepEncoding {
			if strings.Contains(value, "gzip") {
				supportGzip = true
				break
			}
		}

		if supportGzip {

			c.log.Info("Detected gzip support")

			rw := gzipcomp.NewGzipResponseWriter(w)
			next.ServeHTTP(rw, r)

			contentType := rw.Header().Get("Content-Type")
			if contentType == "text/html" || contentType == "application/json" {

				cw := gzipcomp.NewGzipCompressWriter(w)
				cw.Header().Set("Content-Encoding", "gzip")
				defer cw.Close()

				ow = cw
				rw.WriteTo(cw)
			}

		} else {
			next.ServeHTTP(ow, r)
		}
		c.log.Info("request served from GzipCompressor")

	})
}
