const initCycleTLS = require('./dist/index.js').default;

async function testWebSocket() {
  console.log('🐛 Debug: Testing WebSocket implementation...\n');

  try {
    const cycleTLS = await initCycleTLS({debug: true});
    console.log('✅ CycleTLS initialized\n');

    const ws = await cycleTLS.ws('wss://echo.websocket.org');

    console.log('🐛 Debug: WebSocket object created');
    console.log('  - readyState:', ws.readyState);
    console.log('  - url:', ws.url);
    console.log();

    // Add raw event listener to see ALL events
    const originalEmit = ws.emit.bind(ws);
    ws.emit = function(event, ...args) {
      console.log(`🐛 Event emitted: "${event}"`, args.length > 0 ? args : '');
      return originalEmit(event, ...args);
    };

    ws.on('open', () => {
      console.log('✅ OPEN event received!');
      console.log('  - readyState:', ws.readyState);

      ws.send('Hello!');

      setTimeout(() => {
        ws.close();
      }, 2000);
    });

    ws.on('message', (data, isBinary) => {
      console.log('📥 MESSAGE event:', data.toString());
    });

    ws.on('close', (code, reason) => {
      console.log('🔒 CLOSE event:', code, reason);
      setTimeout(() => cycleTLS.exit().then(() => process.exit(0)), 500);
    });

    ws.on('error', (error) => {
      console.error('❌ ERROR event:', error.message);
    });

    // Wait to see if connection opens
    setTimeout(() => {
      console.log('\n⏰ After 10 seconds:');
      console.log('  - readyState:', ws.readyState);
      if (ws.readyState === 0) {
        console.log('  ❌ Connection still in CONNECTING state');
        cycleTLS.exit().then(() => process.exit(1));
      }
    }, 10000);

  } catch (error) {
    console.error(`❌ Error: ${error.message}`);
    console.error(error.stack);
    process.exit(1);
  }
}

testWebSocket();
