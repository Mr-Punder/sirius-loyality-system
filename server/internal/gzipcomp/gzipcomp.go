package gzipcomp

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
)

// GzipCompressWiter allows to use http.ResponseWriter with gzip compression
type GzipCompressWiter struct {
	http.ResponseWriter
	zw *gzip.Writer
}

func NewGzipCompressWriter(w http.ResponseWriter) *GzipCompressWiter {
	return &GzipCompressWiter{
		ResponseWriter: w,
		zw:             gzip.NewWriter(w),
	}
}

func NewEmptyGzipCompressWriter() *GzipCompressWiter {
	return &GzipCompressWiter{}
}

func (gw *GzipCompressWiter) SetResponseWriter(w http.ResponseWriter) {
	gw.ResponseWriter = w
}

func (gw *GzipCompressWiter) Header() http.Header {
	return gw.ResponseWriter.Header()
}

func (gw *GzipCompressWiter) Write(b []byte) (int, error) {
	gw.WriteHeader(200)
	return gw.zw.Write(b)
}

func (gw *GzipCompressWiter) WriteHeader(StatusCode int) {
	if StatusCode < 300 {
		gw.ResponseWriter.Header().Set("Content-Encoding", "gzip")
	}
	gw.ResponseWriter.WriteHeader(StatusCode)
}

func (gw *GzipCompressWiter) Close() error {
	return gw.zw.Close()
}

// GzipResponseWriter stores response before possible compression
type GzipResponseWriter struct {
	w      http.ResponseWriter
	buffer *bytes.Buffer
}

func NewGzipResponseWriter(w http.ResponseWriter) *GzipResponseWriter {
	return &GzipResponseWriter{
		w:      w,
		buffer: bytes.NewBuffer(nil),
	}
}
func (rw *GzipResponseWriter) WriteTo(wr http.ResponseWriter) {
	rw.buffer.WriteTo(wr)
}

func (rw *GzipResponseWriter) Header() http.Header {
	return rw.w.Header()
}

func (rw *GzipResponseWriter) Write(data []byte) (int, error) {
	return rw.buffer.Write(data)
}

func (rw *GzipResponseWriter) WriteHeader(statusCode int) {
	rw.w.WriteHeader(statusCode)
}

// GzipCompressReader is Readcloser with gzip decompression
type GzipCompressReader struct {
	io.ReadCloser
	zr *gzip.Reader
}

func NewGzipCompressReader(r io.ReadCloser) (*GzipCompressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &GzipCompressReader{
		ReadCloser: r,
		zr:         zr,
	}, nil
}

func NewEmptyGzipCompressReader() *GzipCompressReader {

	return &GzipCompressReader{}
}

func (gr *GzipCompressReader) SetReader(r io.ReadCloser) error {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	gr.ReadCloser = r
	gr.zr = zr
	return nil
}

func (gr *GzipCompressReader) Read(b []byte) (int, error) {
	return gr.zr.Read(b)
}

func (gr *GzipCompressReader) Close() error {
	if err := gr.ReadCloser.Close(); err != nil {
		return err
	}
	return gr.zr.Close()
}
