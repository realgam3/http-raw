package http_raw

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"strings"
)

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.ToUpper(req.Method) != "RAW" {
		return t.Clone().RoundTrip(req)
	}

	// Extract the raw request from the body
	rawRequest, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	err = req.Body.Close()
	if err != nil {
		return nil, err
	}

	req.Body = io.NopCloser(bytes.NewReader([]byte("")))

	// Check if the request is HTTPS and use TLS
	var conn net.Conn
	if req.URL.Scheme == "https" {
		// Open a TCP connection to the server
		conn, err = tls.Dial("tcp", req.URL.Host, t.TLSClientConfig)
		if err != nil {
			return nil, err
		}
	} else {
		// Open a TCP connection to the server
		conn, err = net.Dial("tcp", req.URL.Host)
		if err != nil {
			return nil, err
		}
	}
	defer conn.Close()

	// Write the raw request to the connection
	_, err = conn.Write(rawRequest)
	if err != nil {
		return nil, err
	}

	// Read the response
	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		return nil, err
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Restore the request body to its original state
	err = resp.Body.Close()
	if err != nil {
		return nil, err
	}
	resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	return resp, nil
}
