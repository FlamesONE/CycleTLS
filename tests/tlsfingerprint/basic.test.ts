/**
 * Basic TLS fingerprint tests against tlsfingerprint.com
 *
 * Tests basic HTTP methods (GET, POST, PUT, PATCH, DELETE) and
 * validates that TLS fingerprint fields are present in responses.
 *
 * Mirrors Go tests in cycletls/tests/integration/tlsfingerprint/echo_test.go
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
  EchoResponse,
} from "./helpers";

// Longer timeout for network requests
jest.setTimeout(90000);

describe("TLS Fingerprint - Basic HTTP Methods", () => {
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

  describe("GET Requests", () => {
    it("should make a GET request and return TLS fingerprint fields", async () => {
      const options = getDefaultOptions();
      const response = await client.get(`${TEST_SERVER_URL}/get?foo=bar`, options);

      expect(response.statusCode).toBe(200);

      const body = await consumeBodyAsJson<EchoResponse>(response.body);

      // Verify TLS fingerprint fields are present
      assertTLSFieldsPresent(body as unknown as Record<string, unknown>);

      // Verify query args are present
      expect(body.args).toBeDefined();
      expect(body.args?.foo).toBe("bar");
    });

    it("should include ja3 hash in response", async () => {
      const options = getDefaultOptions();
      const response = await client.get(`${TEST_SERVER_URL}/get`, options);

      expect(response.statusCode).toBe(200);

      const body = await consumeBodyAsJson<EchoResponse>(response.body);

      expect(body.ja3).toBeDefined();
      expect(body.ja3.length).toBeGreaterThan(0);
      expect(body.ja3_hash).toBeDefined();
      expect(body.ja3_hash.length).toBeGreaterThan(0);
    });

    it("should include ja4 fingerprint in response", async () => {
      const options = getDefaultOptions();
      const response = await client.get(`${TEST_SERVER_URL}/get`, options);

      expect(response.statusCode).toBe(200);

      const body = await consumeBodyAsJson<EchoResponse>(response.body);

      expect(body.ja4).toBeDefined();
      expect(body.ja4.length).toBeGreaterThan(0);
    });

    it("should include peetprint in response", async () => {
      const options = getDefaultOptions();
      const response = await client.get(`${TEST_SERVER_URL}/get`, options);

      expect(response.statusCode).toBe(200);

      const body = await consumeBodyAsJson<EchoResponse>(response.body);

      expect(body.peetprint).toBeDefined();
      expect(body.peetprint.length).toBeGreaterThan(0);
      expect(body.peetprint_hash).toBeDefined();
      expect(body.peetprint_hash.length).toBeGreaterThan(0);
    });
  });

  describe("POST Requests", () => {
    it("should make a POST request with JSON body", async () => {
      const options = getDefaultOptions();
      const response = await client.post(
        `${TEST_SERVER_URL}/post`,
        JSON.stringify({ message: "hello" }),
        {
          ...options,
          headers: { "Content-Type": "application/json" },
        }
      );

      expect(response.statusCode).toBe(200);

      const body = await consumeBodyAsJson<EchoResponse>(response.body);

      assertTLSFieldsPresent(body as unknown as Record<string, unknown>);

      // Verify the data was received (either in data or json field)
      expect(body.data || body.json).toBeTruthy();
    });
  });

  describe("PUT Requests", () => {
    it("should make a PUT request with JSON body", async () => {
      const options = getDefaultOptions();
      const response = await client.request({
        url: `${TEST_SERVER_URL}/put`,
        method: "PUT",
        body: JSON.stringify({ update: "data" }),
        headers: { "Content-Type": "application/json" },
        ...options,
      });

      expect(response.statusCode).toBe(200);

      const body = await consumeBodyAsJson<EchoResponse>(response.body);

      assertTLSFieldsPresent(body as unknown as Record<string, unknown>);
    });
  });

  describe("PATCH Requests", () => {
    it("should make a PATCH request with JSON body", async () => {
      const options = getDefaultOptions();
      const response = await client.request({
        url: `${TEST_SERVER_URL}/patch`,
        method: "PATCH",
        body: JSON.stringify({ patch: "value" }),
        headers: { "Content-Type": "application/json" },
        ...options,
      });

      expect(response.statusCode).toBe(200);

      const body = await consumeBodyAsJson<EchoResponse>(response.body);

      assertTLSFieldsPresent(body as unknown as Record<string, unknown>);
    });
  });

  describe("DELETE Requests", () => {
    it("should make a DELETE request", async () => {
      const options = getDefaultOptions();
      const response = await client.request({
        url: `${TEST_SERVER_URL}/delete`,
        method: "DELETE",
        ...options,
      });

      expect(response.statusCode).toBe(200);

      const body = await consumeBodyAsJson<EchoResponse>(response.body);

      assertTLSFieldsPresent(body as unknown as Record<string, unknown>);
    });
  });

  describe("Anything Endpoint", () => {
    it("should echo request to /anything endpoint", async () => {
      const options = getDefaultOptions();
      const response = await client.get(`${TEST_SERVER_URL}/anything`, options);

      expect(response.statusCode).toBe(200);

      const body = await consumeBodyAsJson<EchoResponse>(response.body);

      assertTLSFieldsPresent(body as unknown as Record<string, unknown>);

      // Verify method is captured
      expect(body.method).toBe("GET");
    });

    it("should capture POST method in /anything endpoint", async () => {
      const options = getDefaultOptions();
      const response = await client.post(
        `${TEST_SERVER_URL}/anything`,
        JSON.stringify({ test: "data" }),
        {
          ...options,
          headers: { "Content-Type": "application/json" },
        }
      );

      expect(response.statusCode).toBe(200);

      const body = await consumeBodyAsJson<EchoResponse>(response.body);

      assertTLSFieldsPresent(body as unknown as Record<string, unknown>);
      expect(body.method).toBe("POST");
    });
  });

  describe("Headers Endpoint", () => {
    it("should echo custom headers", async () => {
      const options = getDefaultOptions();
      const response = await client.request({
        url: `${TEST_SERVER_URL}/headers`,
        ...options,
        headers: {
          "X-Custom-Header": "TestValue123",
          Accept: "application/json",
        },
      });

      expect(response.statusCode).toBe(200);

      const body = await consumeBodyAsJson<Record<string, unknown>>(response.body);

      assertTLSFieldsPresent(body);

      // Headers should be echoed back
      const headers = body.headers as Record<string, string> | undefined;
      expect(headers).toBeDefined();
      if (headers) {
        // Header names may be normalized (case-insensitive)
        const customHeader =
          headers["X-Custom-Header"] || headers["x-custom-header"];
        expect(customHeader).toBe("TestValue123");
      }
    });
  });
});
