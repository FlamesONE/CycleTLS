import CycleTLS from "../dist/index.js";
import { withCycleTLS } from "./test-utils.js";
jest.setTimeout(30000);

test("Should return a timeout error", async () => {
  // V2 API: timeout is set at client level, not per-request
  // Use a 2-second client timeout to trigger timeout on /delay/4 (4 second delay)
  await withCycleTLS({ port: 9116, timeout: 2000 }, async (client) => {
    const ja3 =
      "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-13-18-51-45-43-27-21,29-23-24,0";
    const userAgent =
      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/78.0.3904.97 Safari/537.36";

    // V2 API throws CycleTLSError on timeout
    await expect(
      client.get("https://httpbin.org/delay/4", {
        ja3: ja3,
        userAgent: userAgent,
      })
    ).rejects.toThrow(/timeout/i);
  });

  // Separate client with longer timeout for the success test
  await withCycleTLS({ port: 9116, timeout: 30000 }, async (client) => {
    const ja3 =
      "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-13-18-51-45-43-27-21,29-23-24,0";
    const userAgent =
      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/78.0.3904.97 Safari/537.36";

    const normalResponse = await client.get("https://httpbin.org/delay/1", {
      ja3: ja3,
      userAgent: userAgent,
    });

    expect(normalResponse.status).toBe(200);
  });
});
