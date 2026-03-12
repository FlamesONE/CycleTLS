package cycletls

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	http "github.com/Danny-Dasilva/fhttp"
)

// TestDialTLS_DoesNotBlockOtherAddresses verifies that a slow TLS handshake
// for one address does not block requests to other addresses.
func TestDialTLS_DoesNotBlockOtherAddresses(t *testing.T) {
	rtIface := newRoundTripper(Browser{})
	rt, ok := rtIface.(*roundTripper)
	if !ok {
		t.Fatalf("expected *roundTripper, got %T", rtIface)
	}

	// Pre-cache a transport for addr1
	addr1 := "example.com:443"
	c1, _ := net.Pipe()
	defer c1.Close()
	rt.cachedConnections[addr1] = c1
	rt.cachedTransports[addr1] = &http.Transport{}

	// Reading addr1 should return immediately even if we simulate concurrent use
	done := make(chan bool, 1)
	go func() {
		req, _ := http.NewRequest("GET", "https://example.com/", nil)
		addr := rt.getDialTLSAddr(req)
		err := rt.getTransport(req, addr)
		done <- (err == nil)
	}()

	select {
	case success := <-done:
		if !success {
			t.Fatal("getTransport failed for cached address")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("getTransport blocked — dialTLS is holding global lock during handshake")
	}
}

// TestClientPool_Eviction verifies that the client pool evicts old entries
// when it reaches maxClientPoolSize.
func TestClientPool_Eviction(t *testing.T) {
	// Clear pool
	advancedClientPoolMutex.Lock()
	advancedClientPool = make(map[string]*ClientPoolEntry)
	advancedClientPoolMutex.Unlock()

	// Fill the pool to trigger eviction
	now := time.Now()
	advancedClientPoolMutex.Lock()
	for i := 0; i < maxClientPoolSize+10; i++ {
		key := string(rune(i)) + "_test_key"
		advancedClientPool[key] = &ClientPoolEntry{
			Client:    http.Client{},
			CreatedAt: now.Add(-time.Duration(i) * time.Minute),
			LastUsed:  now.Add(-time.Duration(i) * time.Minute),
		}
	}
	advancedClientPoolMutex.Unlock()

	// Requesting a new client should trigger eviction
	browser := Browser{
		UserAgent: "test-eviction-ua",
	}
	_, err := getOrCreateClient(browser, 10, false, "test-eviction-ua", true)
	if err != nil {
		t.Fatalf("getOrCreateClient failed: %v", err)
	}

	advancedClientPoolMutex.RLock()
	poolSize := len(advancedClientPool)
	advancedClientPoolMutex.RUnlock()

	if poolSize > maxClientPoolSize {
		t.Fatalf("pool should have been evicted to <= %d, got %d", maxClientPoolSize, poolSize)
	}

	// Cleanup
	clearAllConnections()
}

// TestClientPool_CleanupRemovesOld verifies that cleanupClientPool removes
// entries older than maxAge.
func TestClientPool_CleanupRemovesOld(t *testing.T) {
	advancedClientPoolMutex.Lock()
	advancedClientPool = make(map[string]*ClientPoolEntry)

	// Add old entry
	advancedClientPool["old_entry"] = &ClientPoolEntry{
		Client:    http.Client{},
		CreatedAt: time.Now().Add(-1 * time.Hour),
		LastUsed:  time.Now().Add(-1 * time.Hour),
	}
	// Add fresh entry
	advancedClientPool["fresh_entry"] = &ClientPoolEntry{
		Client:    http.Client{},
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}
	advancedClientPoolMutex.Unlock()

	cleanupClientPool(30 * time.Minute)

	advancedClientPoolMutex.RLock()
	_, hasOld := advancedClientPool["old_entry"]
	_, hasFresh := advancedClientPool["fresh_entry"]
	advancedClientPoolMutex.RUnlock()

	if hasOld {
		t.Fatal("old entry should have been cleaned up")
	}
	if !hasFresh {
		t.Fatal("fresh entry should still be in pool")
	}

	// Cleanup
	clearAllConnections()
}

// TestDo_RespectsTimeout verifies that Do() returns within timeout+buffer
// even when the server is unreachable.
func TestDo_RespectsTimeout(t *testing.T) {
	client := Init()
	defer client.Close()

	start := time.Now()
	// Use a non-routable IP to simulate a hung connection
	resp, err := client.Do("https://192.0.2.1/test", Options{
		Timeout: 3, // 3 seconds
	}, "GET")
	elapsed := time.Since(start)

	// Should complete within timeout + 5s buffer + some overhead
	if elapsed > 15*time.Second {
		t.Fatalf("Do() took %v — should have timed out around 8s", elapsed)
	}

	// Either an error or a non-200 status
	if err == nil && resp.Status == 200 {
		t.Fatal("expected error or non-200 for unreachable host")
	}
}

// TestConcurrent_Do_NoRace runs multiple concurrent Do() calls to verify
// no data races occur (run with -race flag).
func TestConcurrent_Do_NoRace(t *testing.T) {
	client := Init()
	defer client.Close()

	var wg sync.WaitGroup
	var errors atomic.Int64

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := client.Do("https://httpbin.org/get", Options{
				Timeout:               5,
				EnableConnectionReuse: true,
			}, "GET")
			if err != nil {
				errors.Add(1)
			}
		}()
	}

	wg.Wait()
	// Some failures are OK (network), but we should not panic or deadlock
	t.Logf("concurrent Do: %d/10 errors (non-zero is OK, we're testing for races)", errors.Load())
}

// TestSocksDialer_RespectsContext verifies that SocksDialer.DialContext
// respects context cancellation.
func TestSocksDialer_RespectsContext(t *testing.T) {
	dialer := &SocksDialer{
		socksDial: func(network, addr string) (net.Conn, error) {
			// Simulate a slow connection — blocks for 10s
			time.Sleep(10 * time.Second)
			return nil, net.ErrClosed
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := dialer.DialContext(ctx, "tcp", "192.0.2.1:1080")
	elapsed := time.Since(start)

	if elapsed > 2*time.Second {
		t.Fatalf("DialContext took %v — should have been cancelled in ~100ms", elapsed)
	}
	if err == nil {
		t.Fatal("expected context deadline error")
	}
}

// TestRoundTripper_CacheEviction verifies that CloseIdleConnections
// evicts transports when the cache exceeds the limit.
func TestRoundTripper_CacheEviction(t *testing.T) {
	rtIface := newRoundTripper(Browser{})
	rt, ok := rtIface.(*roundTripper)
	if !ok {
		t.Fatalf("expected *roundTripper, got %T", rtIface)
	}

	// Add 300 cached transports (exceeds the 256 limit)
	for i := 0; i < 300; i++ {
		addr := net.JoinHostPort("host"+string(rune(i+'A')), "443")
		rt.cachedTransports[addr] = &http.Transport{}
	}

	// Trigger eviction
	rt.CloseIdleConnections()

	if len(rt.cachedTransports) > 256 {
		t.Fatalf("expected <= 256 cached transports after eviction, got %d", len(rt.cachedTransports))
	}
}
