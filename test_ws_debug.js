const { CycleTLS } = require('./dist/index.js');

(async () => {
  const client = new CycleTLS({ debug: true });
  await client.ensureServerRunning();

  console.log('Testing WebSocket with debug...');

  try {
    console.log('Connecting to WebSocket...');
    const ws = await client.ws('wss://ws.postman-echo.com/raw');
    console.log('ws() promise resolved - connection established');

    ws.on('open', () => {
      console.log('open event fired');
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

    // Timeout
    setTimeout(() => {
      console.log('Test timeout - ws state:', ws.readyState);
      client.close();
      process.exit(1);
    }, 10000);

  } catch (err) {
    console.error('Connection failed:', err);
    await client.close();
    process.exit(1);
  }
})();
