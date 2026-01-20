import CycleTLS from "../dist/index.js";
import { withCycleTLS, streamToJson } from "./test-utils";

test("Should Handle URL Encoded Form Data Correctly", async () => {
  await withCycleTLS({ port: 9200 }, async (cycleTLS) => {
    const urlEncodedData = new URLSearchParams();
    urlEncodedData.append("key1", "value1");
    urlEncodedData.append("key2", "value2");

    const response = await cycleTLS.post(
      "http://httpbin.org/post",
      urlEncodedData.toString(),
      {
        headers: {
          "Content-Type": "application/x-www-form-urlencoded",
        },
      }
    );
    const responseBody = await streamToJson<{ form: Record<string, string> }>(response.body);

    // Validate the 'form' part of the response
    expect(responseBody.form).toEqual({
      key1: "value1",
      key2: "value2",
    });
  });
});
