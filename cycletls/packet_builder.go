package cycletls

import (
	"bytes"
	"encoding/json"
)

// writeU16 writes a 16-bit unsigned integer in big-endian format.
func writeU16(b *bytes.Buffer, v int) {
	b.WriteByte(byte(v >> 8))
	b.WriteByte(byte(v))
}

// writeStringWithLen writes a string with a 2-byte big-endian length prefix.
func writeStringWithLen(b *bytes.Buffer, s string) {
	writeU16(b, len(s))
	b.WriteString(s)
}

// writeRequestAndMethod writes the request ID and method name using length-prefixed format.
func writeRequestAndMethod(b *bytes.Buffer, requestID, method string) {
	writeStringWithLen(b, requestID)
	writeStringWithLen(b, method)
}

// buildErrorFrame creates an error response packet with status code and message.
func buildErrorFrame(requestID string, statusCode int, message string) []byte {
	var b bytes.Buffer
	writeRequestAndMethod(&b, requestID, "error")
	writeU16(&b, statusCode)
	writeStringWithLen(&b, message)
	return b.Bytes()
}

// buildEndFrame generates a frame signaling request completion.
func buildEndFrame(requestID string) []byte {
	var b bytes.Buffer
	writeRequestAndMethod(&b, requestID, "end")
	return b.Bytes()
}

// writeU32 writes a 32-bit unsigned integer in big-endian format.
func writeU32(b *bytes.Buffer, v int) {
	b.WriteByte(byte(v >> 24))
	b.WriteByte(byte(v >> 16))
	b.WriteByte(byte(v >> 8))
	b.WriteByte(byte(v))
}

// buildDataFrame packages response body data with a 4-byte length header.
func buildDataFrame(requestID string, body []byte) []byte {
	var b bytes.Buffer
	writeRequestAndMethod(&b, requestID, "data")
	writeU32(&b, len(body))
	b.Write(body)
	return b.Bytes()
}

// buildResponseFrame constructs the main response packet containing status code, final URL, and HTTP headers.
func buildResponseFrame(requestID string, statusCode int, finalURL string, headers map[string][]string) []byte {
	var b bytes.Buffer
	writeRequestAndMethod(&b, requestID, "response")
	writeU16(&b, statusCode)
	writeStringWithLen(&b, finalURL)

	// headers
	writeU16(&b, len(headers))
	for name, values := range headers {
		writeStringWithLen(&b, name)
		writeU16(&b, len(values))
		for _, v := range values {
			writeStringWithLen(&b, v)
		}
	}

	return b.Bytes()
}

// -----------------------------------------------------------------------------
// WebSocket Frame Builders
// -----------------------------------------------------------------------------

// buildWebSocketOpenFrame creates a ws_open frame with protocol and extensions.
// The payload is JSON encoded: {"type":"open","protocol":"...","extensions":"..."}
func buildWebSocketOpenFrame(requestID string, protocol, extensions string) []byte {
	// Create JSON payload
	openMsg := map[string]interface{}{
		"type":       "open",
		"protocol":   protocol,
		"extensions": extensions,
	}
	payload, _ := json.Marshal(openMsg)

	var b bytes.Buffer
	writeRequestAndMethod(&b, requestID, "ws_open")
	writeU32(&b, len(payload))
	b.Write(payload)
	return b.Bytes()
}

// buildWebSocketMessageFrame creates a ws_message frame.
// messageType: 1 = text, 2 = binary
func buildWebSocketMessageFrame(requestID string, messageType int, data []byte) []byte {
	var b bytes.Buffer
	writeRequestAndMethod(&b, requestID, "ws_message")
	b.WriteByte(byte(messageType))
	writeU32(&b, len(data))
	b.Write(data)
	return b.Bytes()
}

// buildWebSocketCloseFrame creates a ws_close frame.
func buildWebSocketCloseFrame(requestID string, code int, reason string) []byte {
	// Create JSON payload
	closeMsg := map[string]interface{}{
		"type":   "close",
		"code":   code,
		"reason": reason,
	}
	payload, _ := json.Marshal(closeMsg)

	var b bytes.Buffer
	writeRequestAndMethod(&b, requestID, "ws_close")
	writeU32(&b, len(payload))
	b.Write(payload)
	return b.Bytes()
}

// buildWebSocketErrorFrame creates a ws_error frame.
func buildWebSocketErrorFrame(requestID string, statusCode int, message string) []byte {
	var b bytes.Buffer
	writeRequestAndMethod(&b, requestID, "ws_error")
	writeU16(&b, statusCode)
	writeStringWithLen(&b, message)
	return b.Bytes()
}
