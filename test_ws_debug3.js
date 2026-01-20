const { CycleTLS } = require('./dist/index.js');

(async () => {
  const client = new CycleTLS({ debug: true });
  await client.ensureServerRunning();

  console.log('Testing WebSocket with debug...');

  try {
    console.log('Connecting to WebSocket...');
    const ws = await client.ws('wss://ws.postman-echo.com/raw');

    ws.on('open', () => {
      console.log('OPEN event fired, readyState:', ws.readyState);
      console.log('Sending "Hello"...');
      ws.send('Hello', (err) => {
        if (err) {
          console.error('Send error:', err);
        } else {
          console.log('Send callback - no error');
        }
      });
    });

    ws.on('message', (data, isBinary) => {
      console.log('MESSAGE received:', data, 'isBinary:', isBinary);
      ws.close(1000, 'Done');
    });

    ws.on('close', (code, reason) => {
      console.log('CLOSE event:', code, reason);
      client.close();
      process.exit(0);
    });

    ws.on('error', (err) => {
      console.error('ERROR event:', err);
    });

    setTimeout(() => {
      console.log('Timeout - readyState:', ws.readyState);
      console.log('Closing manually...');
      ws.close(1000, 'Timeout');
    }, 10000);

  } catch (err) {
    console.error('Connection failed:', err);
    await client.close();
    process.exit(1);
  }
})();
