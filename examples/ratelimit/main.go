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
	UploadRateLimit   = 50 * 1024  // 50 KB/s
)

func main() {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = true

	// Rate limit HTTP responses (download only)
	downloadBucket := ratelimit.NewBucketWithRate(DownloadRateLimit, DownloadRateLimit)
	proxy.OnResponse().DoFunc(
		func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
			resp.Body = ratelimit.Reader(resp.Body, downloadBucket)
			return resp
		})

	// Rate limit HTTPS connections
	proxy.OnRequest().HandleConnect(func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
		return &goproxy.ConnectAction{
			Action: goproxy.ConnectHijack,
			Hijack: func(clientConn net.Conn) {
				destConn, err := net.Dial("tcp", host)
				if err != nil {
					clientConn.Close()
					return
				}
				rdBucket := ratelimit.NewBucketWithRate(DownloadRateLimit, DownloadRateLimit)
				wrBucket := ratelimit.NewBucketWithRate(UploadRateLimit, UploadRateLimit)

				go func() {
					io.Copy(ratelimit.Writer(destConn, wrBucket), clientConn)
					destConn.Close()
					clientConn.Close()
				}()
				go func() {
					io.Copy(ratelimit.Writer(clientConn, rdBucket), destConn)
					destConn.Close()
					clientConn.Close()
				}()
			},
		}, host
	})

	log.Fatal(http.ListenAndServe(":8080", proxy))
}
