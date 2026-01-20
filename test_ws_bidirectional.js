const { CycleTLS } = require('./dist/index.js');

(async () => {
  const client = new CycleTLS();
  await client.ensureServerRunning();

  console.log('Testing WebSocket bidirectional support...');

  try {
    const ws = await client.ws('wss://ws.postman-echo.com/raw');

    let messageReceived = false;
    const testTimeout = setTimeout(() => {
      if (!messageReceived) {
        console.log('Test timed out waiting for response');
        ws.terminate();
        client.close();
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
      client.close();
      process.exit(0);
    });

    ws.on('error', (err) => {
      console.error('WebSocket error:', err.message);
    });

  } catch (err) {
    console.error('Failed to connect:', err.message);
    await client.close();
    process.exit(1);
  }
})();
