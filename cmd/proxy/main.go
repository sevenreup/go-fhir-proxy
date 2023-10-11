package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samply/golang-fhir-models/fhir-models/fhir"
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

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		t := time.Now()
		c.Next()
		latency := time.Since(t)
		log.Print(latency)
		status := c.Writer.Status()
		log.Println(status)
	}
}

func main() {
	r := gin.Default()
	r.Use(Logger())
	r.GET("*path", func(c *gin.Context) {
		handleRequest(c)
	})
	r.Run(ReverseServerAddr)
}

func handleRequest(c *gin.Context) {
	req := c.Request
	proxy, err := url.Parse(getLoadBalanceAddr())
	if err != nil {
		log.Printf("error in parse addr: %v", err)
		c.String(500, "error")

	}
	isBundleReq := len(strings.Split(req.URL.Path, "/")) > 2
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

	for k, vv := range resp.Header {
		for _, v := range vv {
			c.Header(k, v)
		}
	}
	defer resp.Body.Close()
	data, err := handleResponseBody(resp, isBundleReq)
	if err != nil {
		log.Printf("error in handle request: %v", err)
		c.String(500, "error")
		return
	}
	bufio.NewReader(data).WriteTo(c.Writer)
}

func handleResponseBody(r *http.Response, isBundleReq bool) (rd io.Reader, err error) {
	if err != nil {
		return nil, fmt.Errorf("error in parse addr: %v", err)
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}
	err = r.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("error closing body: %v", err)
	}

	if isBundleReq {
		return bytes.NewReader(body), nil
	}

	bundle := &fhir.Bundle{}
	if err := json.Unmarshal(body, &bundle); err != nil {
		log.Fatal(err)
	}

	for i, entry := range bundle.Entry {
		fullUrl := serverToProxyUrl(entry.FullUrl)
		bundle.Entry[i].FullUrl = &fullUrl
	}

	for i, link := range bundle.Link {
		fullUrl := serverToProxyUrl(&link.Url)
		bundle.Link[i].Url = fullUrl
	}

	b, err := json.Marshal(bundle)
	if err != nil {
		log.Fatal(err)
	}
	return bytes.NewReader(b), nil
}

func serverToProxyUrl(requstUrl *string) string {
	fullUrl, err := url.Parse(*requstUrl)

	if err != nil {
		log.Fatal(err)
	}

	fullUrl.Host = ReverseServerAddr
	parts := strings.Split(fullUrl.Path, "/")
	path, err := url.JoinPath("", parts[2:]...)
	if err != nil {
		log.Fatal(err)
	}
	fullUrl.Path = path

	return fullUrl.String()
}
