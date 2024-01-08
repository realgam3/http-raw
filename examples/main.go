package main

import (
	"bytes"
	"crypto/tls"
	"github.com/realgam3/http-raw"
	"io"
	"log"
	"time"
)

func main() {
	transport := http_raw.Transport{
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     true,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	}

	client := &http_raw.Client{
		Timeout:   15 * time.Second,
		Transport: &transport,
	}

	res, err := client.Raw("https://httpbin.org:443/", bytes.NewReader([]byte("GET /get?a=a HTTP/1.1\r\nHost: httpbin.org\r\n\r\n")))
	if err != nil {
		log.Fatalln("Error sending request:", err)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalln("Error reading body:", err)
	}

	log.Println(string(body))
}
