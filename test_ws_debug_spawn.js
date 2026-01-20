const { spawn } = require('child_process');
const path = require('path');

// Start Go server with visible output first
const serverPath = path.join(__dirname, 'dist', 'index');
const env = { ...process.env, WS_PORT: '9119' };

console.log('Starting Go server manually with visible output...');
const server = spawn(serverPath, [], {
  env,
  cwd: path.join(__dirname, 'dist'),
  stdio: ['pipe', 'pipe', 'pipe']
});

server.stdout.on('data', (data) => {
  console.log('[GO STDOUT]', data.toString().trim());
});

server.stderr.on('data', (data) => {
  console.log('[GO STDERR]', data.toString().trim());
});

// Now run the same test as test_ws_bidirectional.js
setTimeout(async () => {
  const { CycleTLS } = require('./dist/index.js');

  const client = new CycleTLS({
    debug: true,
    autoSpawn: false  // Use our manually started server
  });

  console.log('Testing WebSocket bidirectional support...');

  try {
    const ws = await client.ws('wss://ws.postman-echo.com/raw');

    let messageReceived = false;
    const testTimeout = setTimeout(() => {
      if (!messageReceived) {
        console.log('Test timed out waiting for response');
        ws.terminate();
        server.kill();
        process.exit(1);
      }
    }, 15000);

    ws.on('open', () => {
      console.log('WebSocket connected!');
      console.log('Sending message...');
      ws.send('Hello from CycleTLS!');
    });

    ws.on('message', (data, isBinary) => {
      messageReceived = true;
      clearTimeout(testTimeout);
      console.log('Received:', data);
      console.log('Is binary:', isBinary);

      console.log('Closing WebSocket...');
      ws.close(1000, 'Test complete');
    });

    ws.on('close', (code, reason) => {
      console.log('WebSocket closed:', code, reason);
      console.log('\nBidirectional WebSocket test PASSED!');
      server.kill();
      process.exit(0);
    });

    ws.on('error', (err) => {
      console.error('WebSocket error:', err.message);
    });

  } catch (err) {
    console.error('Failed to connect:', err.message);
    server.kill();
    process.exit(1);
  }
}, 2000);
