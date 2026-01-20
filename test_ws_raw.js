// Test raw WebSocket without CycleTLS to verify the echo server works
const WebSocket = require('ws');

const ws = new WebSocket('wss://ws.postman-echo.com/raw');

ws.on('open', () => {
  console.log('Raw WS connected');
  ws.send('Hello');
});

ws.on('message', (data) => {
  console.log('Raw WS received:', data.toString());
  ws.close();
});

ws.on('close', () => {
  console.log('Raw WS closed');
  process.exit(0);
});

ws.on('error', (err) => {
  console.error('Raw WS error:', err.message);
});

setTimeout(() => {
  console.log('Raw WS timeout');
  process.exit(1);
}, 10000);
