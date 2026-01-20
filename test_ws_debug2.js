const { CycleTLS } = require('./dist/index.js');

(async () => {
  const client = new CycleTLS({ debug: true });
  await client.ensureServerRunning();

  console.log('Testing WebSocket with debug...');

  try {
    console.log('Connecting to WebSocket...');
    const wsPromise = client.ws('wss://ws.postman-echo.com/raw');

    // Register handlers BEFORE await
    wsPromise.then(ws => {
      console.log('Promise resolved, registering handlers...');
      console.log('Current readyState:', ws.readyState);

      ws.on('open', () => {
        console.log('=== OPEN EVENT FIRED ===');
        ws.send('Hello');
      });

      ws.on('message', (data) => {
        console.log('message:', data);
        ws.close();
      });

      ws.on('close', (code, reason) => {
        console.log('close:', code, reason);
        client.close();
        process.exit(0);
      });

      ws.on('error', (err) => {
        console.error('error:', err);
      });

      // If readyState is already OPEN, the open event was missed
      if (ws.readyState === 1) {
        console.log('WebSocket already open, sending manually...');
        ws.send('Hello from manual send');
      }
    });

    // Timeout
    setTimeout(() => {
      console.log('Test timeout');
      client.close();
      process.exit(1);
    }, 10000);

  } catch (err) {
    console.error('Connection failed:', err);
    await client.close();
    process.exit(1);
  }
})();
