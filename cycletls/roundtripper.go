package cycletls

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	http "github.com/Danny-Dasilva/fhttp"
	http2 "github.com/Danny-Dasilva/fhttp/http2"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	uquic "github.com/refraction-networking/uquic"
	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/proxy"
	"net"
	stdhttp "net/http"
	"strings"
	"sync"
	"time"
)

var errProtocolNegotiated = errors.New("protocol negotiated")

type roundTripper struct {
	sync.RWMutex

	// Per-address semaphores for context-aware transport creation serialization.
	// Channel-based instead of sync.Mutex so that goroutines blocked on lock
	// can be unblocked when the request context is cancelled, preventing
	// goroutine leaks when doCycleTLS times out.
	addressSems     map[string]chan struct{}
	addressSemsLock sync.Mutex

	// TLS fingerprinting options
	JA3              string
	JA4r             string // JA4 raw format with explicit cipher/extension values
	HTTP2Fingerprint string
	QUICFingerprint  string
	USpec            *uquic.QUICSpec // UQuic QUIC specification for HTTP3 fingerprinting
	DisableGrease    bool

	// Browser identification
	UserAgent   string
	HeaderOrder []string

	// Connection options
	TLSConfig          *utls.Config
	InsecureSkipVerify bool
	ServerName         string
	Cookies            []Cookie
	ForceHTTP1         bool
	ForceHTTP3         bool

	// TLS 1.3 specific options
	TLS13AutoRetry bool

	// Caching
	cachedConnections map[string]net.Conn
	cachedTransports  map[string]http.RoundTripper

	dialer proxy.ContextDialer
}

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Apply cookies to the request
	for _, properties := range rt.Cookies {
		cookie := &http.Cookie{
			Name:       properties.Name,
			Value:      properties.Value,
			Path:       properties.Path,
			Domain:     properties.Domain,
			Expires:    properties.JSONExpires.Time,
			RawExpires: properties.RawExpires,
			MaxAge:     properties.MaxAge,
			HttpOnly:   properties.HTTPOnly,
			Secure:     properties.Secure,
			Raw:        properties.Raw,
			Unparsed:   properties.Unparsed,
		}
		req.AddCookie(cookie)
	}

	// Apply user agent
	req.Header.Set("User-Agent", rt.UserAgent)

	// Apply header order if specified (for regular headers, not pseudo-headers)
	if len(rt.HeaderOrder) > 0 {
		req.Header = ConvertHttpHeader(MarshalHeader(req.Header, rt.HeaderOrder))

		// Note: rt.HeaderOrder contains regular headers like "cache-control", "accept", etc.
		// Do NOT overwrite http.PHeaderOrderKey which contains pseudo-headers like ":method", ":path"
		// The pseudo-header order is already set correctly in index.go based on UserAgent parsing
	}

	// Get address for dialing
	addr := rt.getDialTLSAddr(req)

	// Check if we need HTTP/3 - matches reference implementation pattern
	if rt.ForceHTTP3 {
		// Extract host and port from request
		host := req.URL.Hostname()
		port := req.URL.Port()
		if port == "" {
			port = "443" // Default HTTPS port
		}

		// Check for USpec (matches reference implementation logic)
		if rt.USpec != nil {
			// Use UQuic-based HTTP/3 dialing
			conn, err := rt.uhttp3Dial(req.Context(), rt.USpec, host, port)
			if err != nil {
				return nil, fmt.Errorf("uhttp3 dial failed: %w", err)
			}
			defer func() {
				if conn.RawConn != nil {
					conn.RawConn.Close()
				}
				// Close the QUIC connection based on its type
				if conn.QuicConn != nil {
					if conn.IsUQuic {
						if uquicConn, ok := conn.QuicConn.(interface{ CloseWithError(uint64, string) error }); ok {
							uquicConn.CloseWithError(0, "request completed")
						}
					} else {
						if quicConn, ok := conn.QuicConn.(interface{ CloseWithError(uint64, string) error }); ok {
							quicConn.CloseWithError(0, "request completed")
						}
					}
				}
			}()

			// Use the HTTP/3 connection to make the request
			return rt.makeHTTP3Request(req, conn)
		}

		// Fall back to standard HTTP/3 dialing
		conn, err := rt.ghttp3Dial(req.Context(), host, port)
		if err != nil {
			return nil, fmt.Errorf("ghttp3 dial failed: %w", err)
		}
		defer func() {
			if conn.RawConn != nil {
				conn.RawConn.Close()
			}
			// Close the QUIC connection based on its type
			if conn.QuicConn != nil {
				if conn.IsUQuic {
					if uquicConn, ok := conn.QuicConn.(interface{ CloseWithError(uint64, string) error }); ok {
						uquicConn.CloseWithError(0, "request completed")
					}
				} else {
					if quicConn, ok := conn.QuicConn.(interface{ CloseWithError(uint64, string) error }); ok {
						quicConn.CloseWithError(0, "request completed")
					}
				}
			}
		}()

		// Use the HTTP/3 connection to make the request
		return rt.makeHTTP3Request(req, conn)
	}

	// Use cached transport if available, otherwise create a new one
	rt.RLock()
	transport, ok := rt.cachedTransports[addr]
	rt.RUnlock()

	if !ok {
		if err := rt.getTransport(req, addr); err != nil {
			return nil, err
		}
		rt.RLock()
		transport = rt.cachedTransports[addr]
		rt.RUnlock()
	}

	if transport == nil {
		return nil, fmt.Errorf("no transport available for %s", addr)
	}

	// Perform the request
	resp, err := transport.RoundTrip(req)
	if err != nil {
		// If the request failed with a network error, evict the cached transport
		// so the next request creates a fresh connection instead of reusing a dead one.
		errStr := err.Error()
		if strings.Contains(errStr, "use of closed network connection") ||
			strings.Contains(errStr, "connection reset") ||
			strings.Contains(errStr, "broken pipe") ||
			strings.Contains(errStr, "stream error") ||
			strings.Contains(errStr, "GOAWAY") ||
			strings.Contains(errStr, "tls:") ||
			strings.Contains(errStr, "EOF") {
			rt.Lock()
			delete(rt.cachedTransports, addr)
			if conn, exists := rt.cachedConnections[addr]; exists {
				_ = conn.Close()
				delete(rt.cachedConnections, addr)
			}
			rt.Unlock()
		}
	}
	return resp, err
}

func (rt *roundTripper) getTransport(req *http.Request, addr string) error {
	switch strings.ToLower(req.URL.Scheme) {
	case "http":
		// Allow connection reuse by removing DisableKeepAlives
		rt.Lock()
		rt.cachedTransports[addr] = &http.Transport{
			DialContext: rt.dialer.DialContext,
		}
		rt.Unlock()
		return nil
	case "https":
	default:
		return fmt.Errorf("invalid URL scheme: [%v]", req.URL.Scheme)
	}

	// Context-aware semaphore acquisition to prevent goroutine leaks.
	// Unlike sync.Mutex.Lock(), this respects context cancellation so
	// timed-out requests don't pile up waiting forever.
	sem := rt.getAddressSemaphore(addr)
	select {
	case <-sem:
		// Acquired semaphore
	case <-req.Context().Done():
		return req.Context().Err()
	}
	defer func() { sem <- struct{}{} }() // Release semaphore

	// Double-check if transport was created while we were waiting (with proper locking)
	rt.RLock()
	_, exists := rt.cachedTransports[addr]
	rt.RUnlock()
	if exists {
		return nil
	}

	// Establish TLS connection
	_, err := rt.dialTLS(req.Context(), "tcp", addr)
	switch err {
	case errProtocolNegotiated:
		// Transport and connection have been negotiated and cached by dialTLS
	case nil:
		// A cached connection/transport already exists (e.g., created by another goroutine).
	default:
		return err
	}

	return nil
}

func (rt *roundTripper) dialTLS(ctx context.Context, network, addr string) (net.Conn, error) {
	// Quick check under read lock: return cached connection if available
	rt.RLock()
	if conn := rt.cachedConnections[addr]; conn != nil {
		rt.RUnlock()
		return conn, nil
	}
	rt.RUnlock()

	// Check context before expensive dial
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Establish raw connection WITHOUT holding the lock — this can take seconds
	// on dead proxies and must not block other addresses.
	rawConn, err := rt.dialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	// Extract host from address
	var host string
	if host, _, err = net.SplitHostPort(addr); err != nil {
		host = addr
	}
	// Determine SNI to use (custom serverName takes precedence)
	serverName := host
	if rt.ServerName != "" {
		serverName = rt.ServerName
	}

	var spec *utls.ClientHelloSpec
	var proactivelyUpgraded bool // Track if we proactively upgraded TLS 1.2 to 1.3

	// Determine which fingerprint to use
	if rt.QUICFingerprint != "" {
		// Use QUIC fingerprint
		spec, err = QUICStringToSpec(rt.QUICFingerprint, rt.UserAgent, rt.ForceHTTP1)
		if err != nil {
			_ = rawConn.Close()
			return nil, err
		}
	} else if rt.JA3 != "" {
		// Check if we should proactively upgrade TLS 1.2 to TLS 1.3
		if rt.TLS13AutoRetry && strings.HasPrefix(rt.JA3, "771,") {
			// Use TLS 1.3 compatible spec to avoid retry cycle
			spec, err = StringToTLS13CompatibleSpec(rt.JA3, rt.UserAgent, rt.ForceHTTP1)
			proactivelyUpgraded = true
		} else {
			// Use original JA3 fingerprint
			spec, err = StringToSpec(rt.JA3, rt.UserAgent, rt.ForceHTTP1)
		}
		if err != nil {
			_ = rawConn.Close()
			return nil, err
		}
	} else if rt.JA4r != "" {
		// Use JA4r (raw) fingerprint
		spec, err = JA4RStringToSpec(rt.JA4r, rt.UserAgent, rt.ForceHTTP1, rt.DisableGrease, serverName)
		if err != nil {
			_ = rawConn.Close()
			return nil, err
		}
	} else {
		// Default to Chrome fingerprint
		spec, err = StringToSpec(DefaultChrome_JA3, rt.UserAgent, rt.ForceHTTP1)
		if err != nil {
			_ = rawConn.Close()
			return nil, err
		}
	}

	// Create TLS client
	conn := utls.UClient(rawConn, &utls.Config{
		ServerName:         serverName,
		OmitEmptyPsk:       true,
		InsecureSkipVerify: rt.InsecureSkipVerify,
	}, utls.HelloCustom)

	// Apply TLS fingerprint
	if err := conn.ApplyPreset(spec); err != nil {
		_ = conn.Close()
		return nil, err
	}

	// Perform TLS handshake — NO lock held, this can take seconds on slow proxies
	if err = conn.Handshake(); err != nil {
		_ = conn.Close()

		if err.Error() == "tls: CurvePreferences includes unsupported curve" {
			// Check if TLS 1.3 retry is enabled
			if rt.TLS13AutoRetry {
				// Automatically retry with TLS 1.3 compatible curves
				return rt.retryWithTLS13CompatibleCurves(ctx, network, addr, host)
			}
			return nil, fmt.Errorf("conn.Handshake() error for TLS 1.3 (retry disabled): %+v", err)
		}

		// If we proactively upgraded to TLS 1.3 and it failed, try falling back to original TLS 1.2 JA3
		if proactivelyUpgraded && rt.JA3 != "" {
			return rt.retryWithOriginalTLS12JA3(ctx, network, addr, host)
		}

		return nil, fmt.Errorf("uTlsConn.Handshake() error: %+v", err)
	}

	// Cache transport and connection under write lock
	rt.Lock()
	defer rt.Unlock()

	// If transport already exists (created by another goroutine while we were handshaking), close ours
	if rt.cachedTransports[addr] != nil {
		return conn, nil
	}

	// Create and cache transport
	rt.cacheTransportLocked(addr, conn)

	return nil, errProtocolNegotiated
}

// cacheTransportLocked creates an appropriate transport based on negotiated protocol
// and caches it along with the connection. Must be called with rt.Lock() held.
func (rt *roundTripper) cacheTransportLocked(addr string, conn *utls.UConn) {
	switch conn.ConnectionState().NegotiatedProtocol {
	case http2.NextProtoTLS:
		parsedUserAgent := parseUserAgent(rt.UserAgent)
		http2Transport := http2.Transport{
			DialTLS:         rt.dialTLSHTTP2,
			PushHandler:     &http2.DefaultPushHandler{},
			Navigator:       parsedUserAgent.UserAgent,
			ReadIdleTimeout: 15 * time.Second,
			PingTimeout:     5 * time.Second,
		}

		if rt.HTTP2Fingerprint != "" {
			h2Fingerprint, err := NewHTTP2Fingerprint(rt.HTTP2Fingerprint)
			if err == nil {
				h2Fingerprint.Apply(&http2Transport)
			}
		}

		rt.cachedTransports[addr] = &http2Transport
	default:
		rt.cachedTransports[addr] = &http.Transport{
			DialTLSContext: rt.dialTLS,
		}
	}

	rt.cachedConnections[addr] = conn
}

// retryWithTLS13CompatibleCurves retries the TLS connection with TLS 1.3 compatible curves
func (rt *roundTripper) retryWithTLS13CompatibleCurves(ctx context.Context, network, addr, host string) (net.Conn, error) {
	// Check context before retry
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	rawConn, err := rt.dialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	var spec *utls.ClientHelloSpec

	if rt.QUICFingerprint != "" {
		spec, err = QUICStringToSpec(rt.QUICFingerprint, rt.UserAgent, rt.ForceHTTP1)
		if err != nil {
			_ = rawConn.Close()
			return nil, fmt.Errorf("failed to create QUIC spec for TLS 1.3 retry: %v", err)
		}
	} else if rt.JA3 != "" {
		spec, err = StringToTLS13CompatibleSpec(rt.JA3, rt.UserAgent, rt.ForceHTTP1)
		if err != nil {
			_ = rawConn.Close()
			return nil, fmt.Errorf("failed to create TLS 1.3 compatible JA3 spec: %v", err)
		}
	} else if rt.JA4r != "" {
		spec, err = StringToTLS13CompatibleSpec(DefaultChrome_JA3, rt.UserAgent, rt.ForceHTTP1)
		if err != nil {
			_ = rawConn.Close()
			return nil, fmt.Errorf("failed to create TLS 1.3 compatible JA4 fallback spec: %v", err)
		}
	} else {
		spec, err = StringToTLS13CompatibleSpec(DefaultChrome_JA3, rt.UserAgent, rt.ForceHTTP1)
		if err != nil {
			_ = rawConn.Close()
			return nil, fmt.Errorf("failed to create TLS 1.3 compatible default spec: %v", err)
		}
	}

	conn := utls.UClient(rawConn, &utls.Config{
		ServerName:         host,
		OmitEmptyPsk:       true,
		InsecureSkipVerify: rt.InsecureSkipVerify,
	}, utls.HelloCustom)

	if err := conn.ApplyPreset(spec); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to apply TLS 1.3 compatible preset: %v", err)
	}

	if err = conn.Handshake(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("TLS 1.3 compatible handshake failed: %+v", err)
	}

	// Cache transport under write lock
	rt.Lock()
	defer rt.Unlock()
	rt.cacheTransportLocked(addr, conn)

	return nil, errProtocolNegotiated
}

// retryWithOriginalTLS12JA3 retries the TLS connection with the original TLS 1.2 JA3
func (rt *roundTripper) retryWithOriginalTLS12JA3(ctx context.Context, network, addr, host string) (net.Conn, error) {
	// Check context before retry
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	rawConn, err := rt.dialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	spec, err := StringToSpec(rt.JA3, rt.UserAgent, rt.ForceHTTP1)
	if err != nil {
		_ = rawConn.Close()
		return nil, fmt.Errorf("failed to create original TLS 1.2 JA3 spec: %v", err)
	}

	conn := utls.UClient(rawConn, &utls.Config{
		ServerName:         host,
		OmitEmptyPsk:       true,
		InsecureSkipVerify: rt.InsecureSkipVerify,
	}, utls.HelloCustom)

	if err := conn.ApplyPreset(spec); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to apply original TLS 1.2 preset: %v", err)
	}

	if err = conn.Handshake(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("original TLS 1.2 handshake failed: %+v", err)
	}

	// Cache transport under write lock
	rt.Lock()
	defer rt.Unlock()
	rt.cacheTransportLocked(addr, conn)

	return nil, errProtocolNegotiated
}

func (rt *roundTripper) dialTLSHTTP2(network, addr string, _ *utls.Config) (net.Conn, error) {
	// Use a context with timeout to prevent hanging forever on dead connections
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return rt.dialTLS(ctx, network, addr)
}

func (rt *roundTripper) getDialTLSAddr(req *http.Request) string {
	host, port, err := net.SplitHostPort(req.URL.Host)
	if err == nil {
		return net.JoinHostPort(host, port)
	}
	return net.JoinHostPort(req.URL.Host, "443") // Default HTTPS port
}

// getAddressSemaphore returns a channel-based semaphore for the specific address.
// The semaphore serializes transport creation while supporting context cancellation
// via select{}, unlike sync.Mutex which blocks forever.
func (rt *roundTripper) getAddressSemaphore(addr string) chan struct{} {
	rt.addressSemsLock.Lock()
	defer rt.addressSemsLock.Unlock()

	if rt.addressSems == nil {
		rt.addressSems = make(map[string]chan struct{})
	}

	if sem, exists := rt.addressSems[addr]; exists {
		return sem
	}

	sem := make(chan struct{}, 1)
	sem <- struct{}{} // Initially available
	rt.addressSems[addr] = sem
	return sem
}

// CloseIdleConnections closes connections that have been idle for too long
// If selectedAddr is provided, only close connections not matching this address
func (rt *roundTripper) CloseIdleConnections(selectedAddr ...string) {
	rt.Lock()
	defer rt.Unlock()

	// If we have a specific address to keep, only close other connections
	if len(selectedAddr) > 0 && selectedAddr[0] != "" {
		addr := selectedAddr[0]
		for connAddr, conn := range rt.cachedConnections {
			if connAddr != addr {
				_ = conn.Close()
				delete(rt.cachedConnections, connAddr)
			}
		}
	} else {
		for addr, conn := range rt.cachedConnections {
			_ = conn.Close()
			delete(rt.cachedConnections, addr)
		}
	}

	// Evict cached transports to prevent unbounded map growth (keep max 256 entries)
	const maxCachedTransports = 256
	if len(rt.cachedTransports) > maxCachedTransports {
		keepAddr := ""
		if len(selectedAddr) > 0 {
			keepAddr = selectedAddr[0]
		}
		for tAddr := range rt.cachedTransports {
			if tAddr != keepAddr {
				delete(rt.cachedTransports, tAddr)
			}
		}
	}

	// Evict stale address semaphores to prevent unbounded map growth
	rt.addressSemsLock.Lock()
	if len(rt.addressSems) > maxCachedTransports {
		rt.addressSems = make(map[string]chan struct{})
	}
	rt.addressSemsLock.Unlock()
}

func newRoundTripper(browser Browser, dialer ...proxy.ContextDialer) http.RoundTripper {
	var contextDialer proxy.ContextDialer
	if len(dialer) > 0 {
		contextDialer = dialer[0]
	} else {
		contextDialer = proxy.Direct
	}

	return &roundTripper{
		dialer:             contextDialer,
		JA3:                browser.JA3,
		JA4r:               browser.JA4r,
		HTTP2Fingerprint:   browser.HTTP2Fingerprint,
		QUICFingerprint:    browser.QUICFingerprint,
		USpec:              browser.USpec,
		DisableGrease:      browser.DisableGrease,
		UserAgent:          browser.UserAgent,
		HeaderOrder:        browser.HeaderOrder,
		TLSConfig:          browser.TLSConfig,
		ServerName:         browser.ServerName,
		Cookies:            browser.Cookies,
		cachedTransports:   make(map[string]http.RoundTripper),
		cachedConnections:  make(map[string]net.Conn),
		InsecureSkipVerify: browser.InsecureSkipVerify,
		ForceHTTP1:         browser.ForceHTTP1,
		ForceHTTP3:         browser.ForceHTTP3,
		TLS13AutoRetry:     browser.TLS13AutoRetry,
	}
}

// makeHTTP3Request performs an HTTP/3 request using the provided HTTP/3 connection
func (rt *roundTripper) makeHTTP3Request(req *http.Request, conn *HTTP3Connection) (*http.Response, error) {
	tlsConfig := ConvertUtlsConfig(rt.TLSConfig)
	if tlsConfig == nil {
		tlsConfig = &tls.Config{}
	}
	if rt.ServerName != "" {
		tlsConfig.ServerName = rt.ServerName
	}

	roundTripper := &http3.Transport{
		TLSClientConfig: tlsConfig,
		QUICConfig: &quic.Config{
			HandshakeIdleTimeout:           30 * time.Second,
			MaxIdleTimeout:                 90 * time.Second,
			KeepAlivePeriod:                15 * time.Second,
			InitialStreamReceiveWindow:     512 * 1024,
			MaxStreamReceiveWindow:         2 * 1024 * 1024,
			InitialConnectionReceiveWindow: 1024 * 1024,
			MaxConnectionReceiveWindow:     4 * 1024 * 1024,
			MaxIncomingStreams:              100,
			MaxIncomingUniStreams:           100,
			EnableDatagrams:                false,
			DisablePathMTUDiscovery:        false,
			Allow0RTT:                      false,
		},
	}

	stdReq := &stdhttp.Request{
		Method:           req.Method,
		URL:              req.URL,
		Proto:            req.Proto,
		ProtoMajor:       req.ProtoMajor,
		ProtoMinor:       req.ProtoMinor,
		Header:           ConvertFhttpHeader(req.Header),
		Body:             req.Body,
		GetBody:          req.GetBody,
		ContentLength:    req.ContentLength,
		TransferEncoding: req.TransferEncoding,
		Close:            req.Close,
		Host:             req.Host,
		Form:             req.Form,
		PostForm:         req.PostForm,
		MultipartForm:    req.MultipartForm,
		Trailer:          ConvertFhttpHeader(req.Trailer),
		RemoteAddr:       req.RemoteAddr,
		RequestURI:       req.RequestURI,
		TLS:              nil,
		Cancel:           req.Cancel,
		Response:         nil,
	}

	stdResp, err := roundTripper.RoundTrip(stdReq)
	if err != nil {
		return nil, err
	}

	return &http.Response{
		Status:           stdResp.Status,
		StatusCode:       stdResp.StatusCode,
		Proto:            stdResp.Proto,
		ProtoMajor:       stdResp.ProtoMajor,
		ProtoMinor:       stdResp.ProtoMinor,
		Header:           ConvertHttpHeader(stdResp.Header),
		Body:             stdResp.Body,
		ContentLength:    stdResp.ContentLength,
		TransferEncoding: stdResp.TransferEncoding,
		Close:            stdResp.Close,
		Uncompressed:     stdResp.Uncompressed,
		Trailer:          ConvertHttpHeader(stdResp.Trailer),
		Request:          req,
		TLS:              nil,
	}, nil
}

// Default JA3 fingerprint for Chrome
const DefaultChrome_JA3 = "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-13-18-51-45-43-27-17513,29-23-24,0"
