# WebSocket Implementation Summary

## Overview

WebSocket support for CycleTLS, matching the Node.js `ws` library API with TLS fingerprinting.

## Issue Addressed

- **GitHub Issue #399**: "await cycleTLS.ws() never resolves"
  - ✅ Fixed: Connection now properly emits events and resolves
  - ✅ Added: Full bidirectional communication
  - ✅ Added: Standard event-driven API

## Implementation Details

### Go Backend Changes (`cycletls/index.go`)

1. **Connection Management**
   - Added `WebSocketConnection` struct for tracking active connections
   - Implemented connection registry with thread-safe access
   - Added state tracking (CONNECTING, OPEN, CLOSING, CLOSED)

2. **Bidirectional Communication**
   - Implemented command channel for TypeScript → Go messages
   - Added goroutines for concurrent read/write operations
   - Proper cleanup and synchronization

3. **New Message Types**
   - `ws_open`: Connection established (includes protocol/extensions)
   - `ws_message`: WebSocket message received (text/binary)
   - `ws_close`: Connection closed (with code/reason)
   - `ws_error`: WebSocket error occurred
   - `end`: Connection lifecycle completed

4. **Command Handling**
   - `ws_send`: Send message (text or binary)
   - `ws_close`: Close connection gracefully
   - `ws_ping`: Send ping frame
   - `ws_pong`: Send pong frame

5. **Bug Fixes**
   - Fixed missing 'end' event (#399)
   - Proper goroutine coordination to prevent deadlocks
   - Correct message routing and cleanup

### TypeScript Frontend Changes (`src/index.ts`)

1. **CycleTLSWebSocket Class**
   - Extends EventEmitter for compatibility with ws library
   - Implements all standard WebSocket events
   - Matches ws library API signatures

2. **Properties**
   - `readyState`: Current connection state (0-3)
   - `url`: WebSocket URL
   - `protocol`: Negotiated subprotocol
   - `extensions`: Negotiated extensions
   - `bufferedAmount`: Bytes queued to send
   - `binaryType`: Binary data format
   - `status`: HTTP response status (for backward compatibility)
   - `headers`: HTTP response headers (for backward compatibility)

3. **Methods**
   - `send(data, options?, callback?)`: Send message
   - `close(code?, reason?)`: Close connection
   - `ping(data?, mask?, callback?)`: Send ping
   - `pong(data?, mask?, callback?)`: Send pong
   - `terminate()`: Forcefully close connection

4. **Events** (ws library compatible)
   - `'open'`: Connection established
   - `'message'`: Message received (data, isBinary)
   - `'close'`: Connection closed (code, reason)
   - `'error'`: Error occurred (error)
   - `'ping'`: Ping received (future)
   - `'pong'`: Pong received (future)

5. **Message Routing**
   - Extended SharedInstance to handle WebSocket-specific messages
   - Proper message parsing for all WebSocket frame types
   - Event dispatching to correct WebSocket instances

## API Examples

### Basic Connection

```javascript
const cycleTLS = await initCycleTLS();

const ws = await cycleTLS.ws('wss://echo.websocket.org', {
  ja3: '771,4865-4867-4866...',
  userAgent: 'Mozilla/5.0...'
});

ws.on('open', () => {
  console.log('Connected! ReadyState:', ws.readyState);
  ws.send('Hello WebSocket!');
});

ws.on('message', (data, isBinary) => {
  console.log('Received:', data.toString());
});

ws.on('close', (code, reason) => {
  console.log('Closed:', code, reason);
});

ws.on('error', (error) => {
  console.error('Error:', error);
});
```

### Binary Messages

```javascript
// Send binary data
const binaryData = Buffer.from([0x01, 0x02, 0x03, 0x04]);
ws.send(binaryData);

// Or explicitly specify
ws.send('text data', { binary: false });
```

### Ping/Pong

```javascript
// Send ping
ws.ping('ping data', false, (err) => {
  if (err) console.error('Ping failed:', err);
  else console.log('Ping sent');
});

// Handle pong responses (future)
ws.on('pong', (data) => {
  console.log('Pong received:', data);
});
```

### Graceful Closure

```javascript
// Close with code and reason
ws.close(1000, 'Normal closure');

// Force close
ws.terminate();
```

## Compatibility

### ws Library Compatibility

The implementation matches the Node.js `ws` library API:

| Feature | ws Library | CycleTLS |
|---------|-----------|----------|
| EventEmitter | ✅ | ✅ |
| send() | ✅ | ✅ |
| close() | ✅ | ✅ |
| ping() | ✅ | ✅ |
| pong() | ✅ | ✅ |
| terminate() | ✅ | ✅ |
| readyState | ✅ | ✅ |
| url | ✅ | ✅ |
| protocol | ✅ | ✅ |
| extensions | ✅ | ✅ |
| bufferedAmount | ✅ | ✅ |
| binaryType | ✅ | ✅ |

### Migration from Old API

**Old (Non-functional):**
```javascript
const response = await cycleTLS.ws(url, options);
// response was a CycleTLSResponse with limited WebSocket support
```

**New (Full Support):**
```javascript
const ws = await cycleTLS.ws(url, options);
// ws is a fully functional WebSocket instance

ws.on('open', () => { /* ready to send */ });
ws.on('message', (data) => { /* received message */ });
ws.send('message');
ws.close();
```

## Testing

### Manual Testing

Created test scripts:
- `test_websocket_new.js`: Comprehensive test with echo server
- `test_websocket_debug.js`: Debug version with event logging

### Integration Tests

Update needed for:
- `tests/websocket.test.ts`: Add tests for new WebSocket API
- `cycletls/tests/integration/`: Add Go-side WebSocket tests

## Known Limitations

1. **Connection Timeout**: Relies on default timeout settings
2. **Automatic Reconnection**: Not implemented (user-space feature)
3. **Compression**: Not explicitly configured (relies on underlying library)
4. **Fragmentation**: Handled by gorilla/websocket library

## Future Enhancements

1. **Connection Pooling**: Reuse WebSocket connections across requests
2. **Automatic Ping/Pong**: Built-in keepalive mechanism
3. **Compression Support**: Per-message deflate
4. **Metrics**: Connection statistics and monitoring
5. **Events**: Expose ping/pong events to user code

## Files Modified

### Go Files
- `cycletls/index.go`: Core WebSocket implementation

### TypeScript Files
- `src/index.ts`: CycleTLSWebSocket class and integration

### Test Files
- `test_websocket_new.js`: Manual test script
- `test_websocket_debug.js`: Debug test script

## Performance Considerations

1. **Goroutines**: Each WebSocket connection spawns 2 goroutines (read/write)
2. **Channels**: Buffered channels (size 100) for command queuing
3. **Memory**: Connection registry maintains references until closure
4. **Cleanup**: Proper cleanup on connection close/error

## Security

- Maintains CycleTLS TLS fingerprinting capabilities
- Supports custom SNI via `serverName` option
- All standard WebSocket security features
- Proper certificate validation (can be disabled with `insecureSkipVerify`)

## Summary

- Fixes GitHub issue #399
- Matches the ws library API
- Bidirectional communication
- TLS fingerprinting preserved
- Event-based API (open/message/close/error)
- Proper cleanup on connection close
