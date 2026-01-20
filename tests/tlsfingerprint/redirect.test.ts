/**
 * Redirect tests against tlsfingerprint.com
 *
 * Tests redirect following behavior and status code handling.
 *
 * Mirrors Go tests in cycletls/tests/integration/tlsfingerprint/redirect_test.go
 *
 * PREREQUISITES:
 * - Go binary must be rebuilt with V2 protocol support
 * - Run: npm run build:go (or platform-specific variant)
 */

import CycleTLS from "../../dist/index.js";
import {
  TEST_SERVER_URL,
  getDefaultOptions,
  assertTLSFieldsPresent,
  consumeBodyAsJson,
  consumeBody,
  EchoResponse,
  StatusResponse,
} from "./helpers";

// Longer timeout for redirect chains
jest.setTimeout(90000);

describe("TLS Fingerprint - Redirects", () => {
  let client: CycleTLS;

  beforeEach(() => {
    client = new CycleTLS({
      port: 9119,
      debug: false,
      timeout: 30000,
      autoSpawn: true,
    });
  });

  afterEach(async () => {
    await client.close();
  });

  describe("Redirect Following", () => {
    it("should follow /redirect/3 chain and end at /get", async () => {
      const options = getDefaultOptions();
      const response = await client.get(`${TEST_SERVER_URL}/redirect/3`, options);

      expect(response.statusCode).toBe(200);

      const body = await consumeBodyAsJson<EchoResponse>(response.body);

      // Verify TLS fields present after redirect chain
      assertTLSFieldsPresent(body as unknown as Record<string, unknown>);

      // After following 3 redirects, we end up at /get which returns EchoResponse
      expect(body.method).toBe("GET");
    });

    it("should follow /redirect/1 single redirect", async () => {
      const options = getDefaultOptions();
      const response = await client.get(`${TEST_SERVER_URL}/redirect/1`, options);

      expect(response.statusCode).toBe(200);

      const body = await consumeBodyAsJson<EchoResponse>(response.body);

      assertTLSFieldsPresent(body as unknown as Record<string, unknown>);
    });

    it("should follow /redirect/5 multiple redirects", async () => {
      const options = getDefaultOptions();
      const response = await client.get(`${TEST_SERVER_URL}/redirect/5`, options);

      expect(response.statusCode).toBe(200);

      const body = await consumeBodyAsJson<EchoResponse>(response.body);

      assertTLSFieldsPresent(body as unknown as Record<string, unknown>);
    });
  });

  describe("Redirect-To Endpoint", () => {
    it("should follow redirect-to internal URL", async () => {
      const options = getDefaultOptions();
      const targetUrl = encodeURIComponent(`${TEST_SERVER_URL}/get`);
      const response = await client.get(
        `${TEST_SERVER_URL}/redirect-to?url=${targetUrl}`,
        options
      );

      expect(response.statusCode).toBe(200);

      const body = await consumeBodyAsJson<EchoResponse>(response.body);

      assertTLSFieldsPresent(body as unknown as Record<string, unknown>);

      // Verify url field is present in response
      expect(body.url).toBeDefined();
    });
  });

  describe("Disable Redirect", () => {
    it("should return 302 when redirects are disabled", async () => {
      const options = getDefaultOptions();
      const response = await client.request({
        url: `${TEST_SERVER_URL}/redirect/1`,
        ...options,
        disableRedirect: true,
      });

      expect(response.statusCode).toBe(302);

      // Consume body to complete request
      await consumeBody(response.body);
    });

    it("should return redirect location header when disabled", async () => {
      const options = getDefaultOptions();
      const response = await client.request({
        url: `${TEST_SERVER_URL}/redirect/1`,
        ...options,
        disableRedirect: true,
      });

      expect(response.statusCode).toBe(302);
      expect(response.headers).toBeDefined();

      // Location header should be present
      const location =
        response.headers.Location ||
        response.headers.location ||
        response.headers["Location"] ||
        response.headers["location"];
      expect(location).toBeDefined();

      await consumeBody(response.body);
    });
  });

  describe("Status Codes", () => {
    it("should return 201 Created status", async () => {
      const options = getDefaultOptions();
      const response = await client.get(`${TEST_SERVER_URL}/status/201`, options);

      expect(response.statusCode).toBe(201);

      const body = await consumeBodyAsJson<StatusResponse>(response.body);

      assertTLSFieldsPresent(body as unknown as Record<string, unknown>);

      // Verify status_code in response body matches
      expect(body.status_code).toBe(201);
    });

    it("should return 400 Bad Request status", async () => {
      const options = getDefaultOptions();
      const response = await client.get(`${TEST_SERVER_URL}/status/400`, options);

      expect(response.statusCode).toBe(400);

      const body = await consumeBodyAsJson<StatusResponse>(response.body);

      assertTLSFieldsPresent(body as unknown as Record<string, unknown>);
      expect(body.status_code).toBe(400);
    });

    it("should return 404 Not Found status", async () => {
      const options = getDefaultOptions();
      const response = await client.get(`${TEST_SERVER_URL}/status/404`, options);

      expect(response.statusCode).toBe(404);

      const body = await consumeBodyAsJson<StatusResponse>(response.body);

      assertTLSFieldsPresent(body as unknown as Record<string, unknown>);
      expect(body.status_code).toBe(404);
    });

    it("should return 500 Internal Server Error status", async () => {
      const options = getDefaultOptions();
      const response = await client.get(`${TEST_SERVER_URL}/status/500`, options);

      expect(response.statusCode).toBe(500);

      const body = await consumeBodyAsJson<StatusResponse>(response.body);

      assertTLSFieldsPresent(body as unknown as Record<string, unknown>);
      expect(body.status_code).toBe(500);
    });

    it("should return 204 No Content status", async () => {
      const options = getDefaultOptions();
      const response = await client.get(`${TEST_SERVER_URL}/status/204`, options);

      expect(response.statusCode).toBe(204);

      // 204 has no body, just consume to complete
      await consumeBody(response.body);
    });
  });

  // Skipped: /absolute-redirect endpoint not available on tlsfingerprint.com
  describe.skip("Absolute Redirect", () => {
    it("should follow absolute-redirect endpoint", async () => {
      const options = getDefaultOptions();
      const response = await client.get(
        `${TEST_SERVER_URL}/absolute-redirect/2`,
        options
      );

      expect(response.statusCode).toBe(200);

      const body = await consumeBodyAsJson<EchoResponse>(response.body);

      assertTLSFieldsPresent(body as unknown as Record<string, unknown>);
    });
  });

  // Skipped: /relative-redirect endpoint not available on tlsfingerprint.com
  describe.skip("Relative Redirect", () => {
    it("should follow relative-redirect endpoint", async () => {
      const options = getDefaultOptions();
      const response = await client.get(
        `${TEST_SERVER_URL}/relative-redirect/2`,
        options
      );

      expect(response.statusCode).toBe(200);

      const body = await consumeBodyAsJson<EchoResponse>(response.body);

      assertTLSFieldsPresent(body as unknown as Record<string, unknown>);
    });
  });
});
