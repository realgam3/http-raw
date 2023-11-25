package http_raw

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

type RoundTripper interface {
	RoundTrip(*http.Request) (*http.Response, error)
}

type Client struct {
	Transport     RoundTripper
	CheckRedirect func(req *http.Request, via []*http.Request) error
	Jar           http.CookieJar
	Timeout       time.Duration
}

var DefaultClient = &Client{}

func (c *Client) Clone() *http.Client {
	client := &http.Client{
		Transport:     c.Transport,
		CheckRedirect: c.CheckRedirect,
		Jar:           c.Jar,
		Timeout:       c.Timeout,
	}
	return client
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.Clone().Do(req)
}

func NewRequest(method, url string, body io.Reader) (*http.Request, error) {
	return http.NewRequestWithContext(context.Background(), method, url, body)
}

func extendMap(original, extension map[string]string) {
	for key, value := range extension {
		original[key] = value
	}
}

func (c *Client) Request(method, url string, body io.Reader, header map[string]string) (resp *http.Response, err error) {
	req, err := NewRequest(strings.ToUpper(method), url, body)
	if err != nil {
		return nil, err
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	return c.Do(req)
}

func (c *Client) request(method, url string, args ...any) (resp *http.Response, err error) {
	var body io.Reader = nil
	var header map[string]string = nil

	for _, arg := range args {
		switch v := arg.(type) {
		case io.Reader:
			body = v
		case map[string]string:
			header = v
		}
	}

	return c.Request(method, url, body, header)
}

func (c *Client) Get(url string, args ...any) (resp *http.Response, err error) {
	return c.request("GET", url, args...)
}

func (c *Client) Trace(url string, args ...any) (resp *http.Response, err error) {
	return c.request("TRACE", url, args...)
}

func (c *Client) Connect(url string) (resp *http.Response, err error) {
	return c.Request("CONNECT", url, nil, nil)
}

func (c *Client) Post(url, contentType string, body io.Reader, args ...any) (resp *http.Response, err error) {
	headers := map[string]string{`Content-Type`: contentType}
	if len(args) > 0 {
		if len(args) > 1 {
			return nil, errors.New(`too many arguments`)
		}
		extendMap(headers, args[0].(map[string]string))
	}
	return c.Request("POST", url, body, headers)
}

func (c *Client) Put(url, contentType string, body io.Reader, args ...any) (resp *http.Response, err error) {
	headers := map[string]string{`Content-Type`: contentType}
	if len(args) > 0 {
		if len(args) > 1 {
			return nil, errors.New(`too many arguments`)
		}
		extendMap(headers, args[0].(map[string]string))
	}
	return c.Request("PUT", url, body, headers)

}

func (c *Client) Patch(url, contentType string, body io.Reader, args ...any) (resp *http.Response, err error) {
	headers := map[string]string{`Content-Type`: contentType}
	if len(args) > 0 {
		if len(args) > 1 {
			return nil, errors.New(`too many arguments`)
		}
		extendMap(headers, args[0].(map[string]string))
	}
	return c.Request("PATCH", url, body, headers)

}

func (c *Client) Delete(url string, args ...any) (resp *http.Response, err error) {
	return c.request("DELETE", url, args...)
}

func (c *Client) Head(url string, args ...any) (resp *http.Response, err error) {
	return c.request("HEAD", url, args...)
}

func (c *Client) Options(url string, args ...any) (resp *http.Response, err error) {
	return c.request("OPTIONS", url, args...)
}

func (c *Client) Raw(url string, body io.Reader) (resp *http.Response, err error) {
	return c.Request("RAW", url, body, nil)
}
