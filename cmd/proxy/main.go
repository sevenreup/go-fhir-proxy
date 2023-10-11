package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

const (
	ReverseServerAddr = "0.0.0.0:9090"
)

var RealAddr = []string{
	"https://hapi.fhir.org/baseR4",
}

func getLoadBalanceAddr() string {
	return RealAddr[0]
}

func main() {
	r := gin.Default()
	r.GET("/:path", func(c *gin.Context) {
		req := c.Request
		proxy, err := url.Parse(getLoadBalanceAddr())
		if err != nil {
			log.Printf("error in parse addr: %v", err)
			c.String(500, "error")
			return
		}
		req.URL.Scheme = proxy.Scheme
		req.URL.Host = proxy.Host
		path, err := url.JoinPath(proxy.Path, req.URL.Path)
		if err != nil {
			log.Printf("error in join path: %v", err)
			c.String(500, "error")
			return
		}
		req.URL.Path = path

		transport := http.DefaultTransport
		resp, err := transport.RoundTrip(req)
		if err != nil {
			log.Printf("error in roundtrip: %v", err)
			c.String(500, "error")
			return
		}

		fmt.Println(req, proxy)
		fmt.Println(resp)

		for k, vv := range resp.Header {
			for _, v := range vv {
				c.Header(k, v)
			}
		}
		defer resp.Body.Close()
		bufio.NewReader(resp.Body).WriteTo(c.Writer)
		return
	})
	r.Run(ReverseServerAddr)
}
