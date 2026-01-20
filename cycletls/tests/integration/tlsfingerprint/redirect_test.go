//go:build integration
// +build integration

package tlsfingerprint_test

import (
	"testing"
)

func TestRedirect(t *testing.T) {
	client := newClient()
	defer client.Close()

	opts := getDefaultOptions()
	resp, err := client.Do(TestServerURL+"/redirect/3", opts, "GET")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	assertStatusCode(t, 200, resp.Status)
	assertTLSFieldsPresent(t, resp.Body)

	// After following 3 redirects, we end up at /get which returns EchoResponse
	var echoResp EchoResponse
	parseJSONResponse(t, resp.Body, &echoResp)

	// Verify we got a valid response after redirect chain
	if echoResp.Method != "GET" {
		t.Errorf("Expected method=GET, got %s", echoResp.Method)
	}
}

func TestRedirectTo(t *testing.T) {
	client := newClient()
	defer client.Close()

	opts := getDefaultOptions()
	// Redirect to an internal URL so we get a JSON response with TLS fingerprint
	resp, err := client.Do(TestServerURL+"/redirect-to?url="+TestServerURL+"/get", opts, "GET")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	assertStatusCode(t, 200, resp.Status)
	assertTLSFieldsPresent(t, resp.Body)

	var echoResp EchoResponse
	parseJSONResponse(t, resp.Body, &echoResp)

	// Verify url field is present in response
	if echoResp.URL == "" {
		t.Error("Expected url field in response")
	}
}

func TestStatus(t *testing.T) {
	client := newClient()
	defer client.Close()

	opts := getDefaultOptions()
	resp, err := client.Do(TestServerURL+"/status/201", opts, "GET")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	assertStatusCode(t, 201, resp.Status)
	assertTLSFieldsPresent(t, resp.Body)

	var statusResp StatusResponse
	parseJSONResponse(t, resp.Body, &statusResp)

	// Verify status_code in response body matches
	if statusResp.StatusCode != 201 {
		t.Errorf("Expected status_code=201 in body, got %d", statusResp.StatusCode)
	}
}
