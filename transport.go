package http_raw

import (
	"bufio"
	"container/list"
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

type incomparable [0]func()

type connectMethod struct {
	_            incomparable
	proxyURL     *url.URL
	targetScheme string

	targetAddr string
	onlyH1     bool
}

type connectMethodKey struct {
	proxy, scheme, addr string
	onlyH1              bool
}

type wantConn struct {
	cm    connectMethod
	key   connectMethodKey
	ctx   context.Context
	ready chan struct{}

	beforeDial func()
	afterDial  func()

	mu  sync.Mutex
	pc  *persistConn
	err error
}

type responseAndError struct {
	_   incomparable
	res *http.Response
	err error
}

type requestAndChan struct {
	_         incomparable
	req       *http.Request
	cancelKey cancelKey
	ch        chan responseAndError

	addedGzip bool

	continueCh chan<- struct{}

	callerGone <-chan struct{}
}

type transportRequest struct {
	*http.Request
	extra     http.Header
	trace     *httptrace.ClientTrace
	cancelKey cancelKey

	mu  sync.Mutex
	err error
}

type writeRequest struct {
	req *transportRequest
	ch  chan<- error

	continueCh <-chan struct{}
}

type wantConnQueue struct {
	head    []*wantConn
	headPos int
	tail    []*wantConn
}

type persistConn struct {
	alt http.RoundTripper

	t         *http.Transport
	cacheKey  connectMethodKey
	conn      net.Conn
	tlsState  *tls.ConnectionState
	br        *bufio.Reader
	bw        *bufio.Writer
	nwrite    int64
	reqch     chan requestAndChan
	writech   chan writeRequest
	closech   chan struct{}
	isProxy   bool
	sawEOF    bool
	readLimit int64

	writeErrCh chan error

	writeLoopDone chan struct{}

	idleAt    time.Time
	idleTimer *time.Timer

	mu                   sync.Mutex
	numExpectedResponses int
	closed               error
	canceledErr          error
	broken               bool
	reused               bool

	mutateHeaderFunc func(http.Header)
}

type connLRU struct {
	ll *list.List
	m  map[*persistConn]*list.Element
}

type h2Transport interface {
	CloseIdleConnections()
}

type cancelKey struct {
	req *http.Request
}

type Transport struct {
	idleMu       sync.Mutex
	closeIdle    bool
	idleConn     map[connectMethodKey][]*persistConn
	idleConnWait map[connectMethodKey]wantConnQueue
	idleLRU      connLRU

	reqMu       sync.Mutex
	reqCanceler map[cancelKey]func(error)

	altMu    sync.Mutex
	altProto atomic.Value

	connsPerHostMu sync.Mutex
	connsPerHost   map[connectMethodKey]int
	Proxy          func(*http.Request) (*url.URL, error)

	OnProxyConnectResponse func(ctx context.Context, proxyURL *url.URL, connectReq *http.Request, connectRes *http.Response) error

	DialContext    func(ctx context.Context, network, addr string) (net.Conn, error)
	Dial           func(network, addr string) (net.Conn, error)
	DialTLSContext func(ctx context.Context, network, addr string) (net.Conn, error)
	DialTLS        func(network, addr string) (net.Conn, error)

	TLSClientConfig *tls.Config

	TLSHandshakeTimeout time.Duration

	DisableKeepAlives bool

	DisableCompression bool

	MaxIdleConns int

	MaxIdleConnsPerHost int

	MaxConnsPerHost int

	IdleConnTimeout time.Duration

	ResponseHeaderTimeout time.Duration

	ExpectContinueTimeout time.Duration

	TLSNextProto map[string]func(authority string, c *tls.Conn) RoundTripper

	ProxyConnectHeader http.Header

	GetProxyConnectHeader func(ctx context.Context, proxyURL *url.URL, target string) (http.Header, error)

	MaxResponseHeaderBytes int64

	WriteBufferSize int

	ReadBufferSize int

	nextProtoOnce      sync.Once
	h2transport        h2Transport
	tlsNextProtoWasNil bool

	ForceAttemptHTTP2 bool
}

var DefaultTransport RoundTripper = &Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: defaultTransportDialContext(&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}),
	ForceAttemptHTTP2:     true,
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}

var omitBundledHTTP2 bool

func defaultTransportDialContext(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
	return dialer.DialContext
}

func (t *Transport) Clone() *http.Transport {
	transport := &http.Transport{
		Proxy:                  t.Proxy,
		OnProxyConnectResponse: t.OnProxyConnectResponse,
		DialContext:            t.DialContext,
		Dial:                   t.Dial,
		DialTLS:                t.DialTLS,
		DialTLSContext:         t.DialTLSContext,
		TLSHandshakeTimeout:    t.TLSHandshakeTimeout,
		DisableKeepAlives:      t.DisableKeepAlives,
		DisableCompression:     t.DisableCompression,
		MaxIdleConns:           t.MaxIdleConns,
		MaxIdleConnsPerHost:    t.MaxIdleConnsPerHost,
		MaxConnsPerHost:        t.MaxConnsPerHost,
		IdleConnTimeout:        t.IdleConnTimeout,
		ResponseHeaderTimeout:  t.ResponseHeaderTimeout,
		ExpectContinueTimeout:  t.ExpectContinueTimeout,
		ProxyConnectHeader:     t.ProxyConnectHeader.Clone(),
		GetProxyConnectHeader:  t.GetProxyConnectHeader,
		MaxResponseHeaderBytes: t.MaxResponseHeaderBytes,
		ForceAttemptHTTP2:      t.ForceAttemptHTTP2,
		WriteBufferSize:        t.WriteBufferSize,
		ReadBufferSize:         t.ReadBufferSize,
	}
	return transport
}
