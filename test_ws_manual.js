const { spawn } = require('child_process');
const path = require('path');
const WebSocket = require('ws');

// Start Go server manually
const serverPath = path.join(__dirname, 'dist', 'index');
const env = { ...process.env, WS_PORT: '9119' };

console.log('Starting Go server...');
const server = spawn(serverPath, [], {
  env,
  cwd: path.join(__dirname, 'dist'),
  stdio: ['pipe', 'pipe', 'pipe']
});

server.stdout.on('data', (data) => {
  console.log('[GO STDOUT]', data.toString());
});

server.stderr.on('data', (data) => {
  console.log('[GO STDERR]', data.toString());
});

server.on('error', (err) => {
  console.error('Server error:', err);
});

// Wait for server to start, then run test
setTimeout(async () => {
  console.log('Connecting to Go server...');

  const { CycleTLS } = require('./dist/index.js');

  const client = new CycleTLS({
    debug: true,
    autoSpawn: false  // Don't spawn, use our manual server
  });

  try {
    const ws = await client.ws('wss://ws.postman-echo.com/raw');
    console.log('WebSocket connected!');

    ws.on('open', () => {
      console.log('OPEN event');
      ws.send('Hello');
    });

    ws.on('message', (data) => {
      console.log('MESSAGE:', data);
      ws.close();
    });

    ws.on('close', () => {
      console.log('CLOSE');
      server.kill();
      process.exit(0);
    });

    setTimeout(() => {
      console.log('Test timeout');
      server.kill();
      process.exit(1);
    }, 10000);

  } catch (err) {
    console.error('Error:', err);
    server.kill();
    process.exit(1);
  }
}, 2000);
