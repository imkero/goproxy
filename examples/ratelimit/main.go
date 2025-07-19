package main

import (
	"io"
	"log"
	"net"
	"net/http"

	"github.com/elazarl/goproxy"
	"github.com/juju/ratelimit"
)

const (
	DownloadRateLimit = 100 * 1024 // 100 KB/s
)

var (
	readBucket = ratelimit.NewBucketWithRate(float64(DownloadRateLimit), DownloadRateLimit)
)

type rateLimitedReadCloser struct {
	io.Reader
	closer io.Closer
}

func (r rateLimitedReadCloser) Close() error {
	return r.closer.Close()
}

func NewRateLimitedReadCloser(readCloser io.ReadCloser) rateLimitedReadCloser {
	return rateLimitedReadCloser{
		Reader: ratelimit.Reader(readCloser, readBucket),
		closer: readCloser,
	}
}

type rateLimitedConn struct {
	net.Conn
	reader io.Reader
}

func NewRateLimitedConn(conn net.Conn, readLimit int64) rateLimitedConn {
	return rateLimitedConn{
		Conn:   conn,
		reader: ratelimit.Reader(conn, readBucket),
	}
}

func (r rateLimitedConn) Read(b []byte) (int, error) {
	return r.reader.Read(b)
}

func main() {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = true

	// Rate limit HTTPS connections
	proxy.ConnectDial = func(network string, addr string) (net.Conn, error) {
		conn, err := net.Dial(network, addr)
		if err != nil {
			return conn, err
		}

		return NewRateLimitedConn(conn, DownloadRateLimit), nil
	}

	// Rate limit HTTP responses
	proxy.OnResponse().DoFunc(
		func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
			resp.Body = NewRateLimitedReadCloser(resp.Body)
			return resp
		})

	log.Fatal(http.ListenAndServe(":8080", proxy))
}
