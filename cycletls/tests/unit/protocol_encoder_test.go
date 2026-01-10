//go:build !integration

package unit

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/Danny-Dasilva/CycleTLS/cycletls/server/protocol"
)

func readUint16BE(data []byte, offset int) uint16 {
	return binary.BigEndian.Uint16(data[offset:])
}

func readUint32BE(data []byte, offset int) uint32 {
	return binary.BigEndian.Uint32(data[offset:])
}

func TestEncodeError(t *testing.T) {
	tests := []struct {
		name       string
		requestID  string
		statusCode int
		message    string
	}{
		{"basic error", "req-123", 500, "Internal server error"},
		{"empty message", "req-456", 404, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := protocol.EncodeError(tt.requestID, tt.statusCode, tt.message)
			if len(result) == 0 {
				t.Fatal("EncodeError returned empty slice")
			}

			// Verify request ID
			reqIDLen := int(readUint16BE(result, 0))
			if reqIDLen != len(tt.requestID) {
				t.Errorf("request ID length mismatch: got %d, want %d", reqIDLen, len(tt.requestID))
			}

			reqID := string(result[2 : 2+reqIDLen])
			if reqID != tt.requestID {
				t.Errorf("request ID mismatch: got %q, want %q", reqID, tt.requestID)
			}

			// Verify separator and frame type
			offset := 2 + reqIDLen
			if result[offset] != 0 {
				t.Errorf("separator byte mismatch: got %d, want 0", result[offset])
			}
			offset++

			frameTypeLen := int(result[offset])
			offset++
			frameType := string(result[offset : offset+frameTypeLen])
			if frameType != protocol.FrameTypeError {
				t.Errorf("frame type mismatch: got %q, want %q", frameType, protocol.FrameTypeError)
			}
		})
	}
}

func TestEncodeResponse(t *testing.T) {
	requestID := "req-123"
	statusCode := 200
	url := "https://example.com"
	headers := map[string][]string{
		"Content-Type": {"application/json"},
	}

	result := protocol.EncodeResponse(requestID, statusCode, url, headers)
	if len(result) == 0 {
		t.Fatal("EncodeResponse returned empty slice")
	}

	// Verify request ID
	reqIDLen := int(readUint16BE(result, 0))
	reqID := string(result[2 : 2+reqIDLen])
	if reqID != requestID {
		t.Errorf("request ID mismatch: got %q, want %q", reqID, requestID)
	}

	// Verify separator and frame type
	offset := 2 + reqIDLen
	if result[offset] != 0 {
		t.Errorf("separator byte mismatch: got %d, want 0", result[offset])
	}
	offset++

	frameTypeLen := int(result[offset])
	offset++
	frameType := string(result[offset : offset+frameTypeLen])
	if frameType != protocol.FrameTypeResponse {
		t.Errorf("frame type mismatch: got %q, want %q", frameType, protocol.FrameTypeResponse)
	}
}

func TestEncodeData(t *testing.T) {
	tests := []struct {
		name      string
		requestID string
		data      []byte
	}{
		{"basic data", "req-123", []byte("Hello, World!")},
		{"binary data", "req-789", []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := protocol.EncodeData(tt.requestID, tt.data)
			if len(result) == 0 {
				t.Fatal("EncodeData returned empty slice")
			}

			// Verify request ID
			reqIDLen := int(readUint16BE(result, 0))
			reqID := string(result[2 : 2+reqIDLen])
			if reqID != tt.requestID {
				t.Errorf("request ID mismatch: got %q, want %q", reqID, tt.requestID)
			}

			// Calculate offset to data length
			offset := 2 + len(tt.requestID) + 1 + 1 + len(protocol.FrameTypeData)
			dataLen := int(readUint32BE(result, offset))
			if dataLen != len(tt.data) {
				t.Errorf("data length mismatch: got %d, want %d", dataLen, len(tt.data))
			}

			// Verify data content
			dataStart := offset + 4
			if !bytes.Equal(result[dataStart:dataStart+dataLen], tt.data) {
				t.Error("data content mismatch")
			}
		})
	}
}

func TestEncodeEnd(t *testing.T) {
	requestID := "req-123"
	result := protocol.EncodeEnd(requestID)

	if len(result) == 0 {
		t.Fatal("EncodeEnd returned empty slice")
	}

	// Verify request ID
	reqIDLen := int(readUint16BE(result, 0))
	reqID := string(result[2 : 2+reqIDLen])
	if reqID != requestID {
		t.Errorf("request ID mismatch: got %q, want %q", reqID, requestID)
	}

	// Verify frame length
	expectedLen := 2 + len(requestID) + 1 + 1 + len(protocol.FrameTypeEnd)
	if len(result) != expectedLen {
		t.Errorf("frame length mismatch: got %d, want %d", len(result), expectedLen)
	}
}

func TestFrameTypeConstants(t *testing.T) {
	// Just verify constants are non-empty
	if protocol.FrameTypeError == "" {
		t.Error("FrameTypeError is empty")
	}
	if protocol.FrameTypeResponse == "" {
		t.Error("FrameTypeResponse is empty")
	}
	if protocol.FrameTypeData == "" {
		t.Error("FrameTypeData is empty")
	}
	if protocol.FrameTypeEnd == "" {
		t.Error("FrameTypeEnd is empty")
	}
}
