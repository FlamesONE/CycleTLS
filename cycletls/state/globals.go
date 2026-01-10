// Package state provides shared global state for the cycletls package.
// This includes pooled buffers and debug logging utilities used across
// multiple components.
package state

import (
	"bytes"
	"log"
	"os"
	"sync"
)

// DebugLogger provides a logger for debug output with timestamp and file info.
var DebugLogger = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

// bufferPool provides reusable bytes.Buffer instances to reduce GC pressure
var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// GetBuffer retrieves a buffer from the pool and resets it for reuse.
func GetBuffer() *bytes.Buffer {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// PutBuffer returns a buffer to the pool for reuse.
func PutBuffer(buf *bytes.Buffer) {
	bufferPool.Put(buf)
}
