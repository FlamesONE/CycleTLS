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

	var redirectResp RedirectResponse
	parseJSONResponse(t, resp.Body, &redirectResp)

	// Verify redirect_count is 3
	if redirectResp.RedirectCount != 3 {
		t.Errorf("Expected redirect_count=3, got %d", redirectResp.RedirectCount)
	}

	// Verify location field is present
	if redirectResp.Location == "" {
		t.Error("Expected location field to be present")
	}
}

func TestRedirectTo(t *testing.T) {
	client := newClient()
	defer client.Close()

	opts := getDefaultOptions()
	resp, err := client.Do(TestServerURL+"/redirect-to?url=https://example.com", opts, "GET")
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
