const initCycleTLS = require('./dist/index.js').default;

async function testWebSocket() {
  console.log('🚀 Testing new WebSocket implementation...\n');

  try {
    const cycleTLS = await initCycleTLS();

    console.log('✅ CycleTLS initialized\n');

    // Test 1: Connect to WebSocket echo server
    console.log('📡 Connecting to wss://echo.websocket.org...');
    const ws = await cycleTLS.ws('wss://echo.websocket.org', {
      headers: {
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36'
      }
    });

    console.log(`📊 Connection state: ${ws.readyState} (0=CONNECTING, 1=OPEN, 2=CLOSING, 3=CLOSED)`);
    console.log(`🔗 URL: ${ws.url}`);
    console.log(`🔌 Protocol: ${ws.protocol || 'none'}`);
    console.log();

    // Set up event handlers
    ws.on('open', () => {
      console.log('✅ WebSocket connection opened!');
      console.log(`📊 ReadyState: ${ws.readyState}`);
      console.log();

      // Send test messages
      console.log('📤 Sending message: "Hello WebSocket!"');
      ws.send('Hello WebSocket!');

      setTimeout(() => {
        console.log('📤 Sending message: "Testing bidirectional communication"');
        ws.send('Testing bidirectional communication');
      }, 1000);

      setTimeout(() => {
        console.log('📤 Sending binary message');
        ws.send(Buffer.from([0x01, 0x02, 0x03, 0x04]));
      }, 2000);

      // Close after 4 seconds
      setTimeout(() => {
        console.log('\n🔒 Closing connection...');
        ws.close(1000, 'Test complete');
      }, 4000);
    });

    ws.on('message', (data, isBinary) => {
      if (isBinary) {
        console.log(`📥 Received binary message: ${Buffer.from(data).toString('hex')}`);
      } else {
        console.log(`📥 Received message: ${data.toString()}`);
      }
    });

    ws.on('close', (code, reason) => {
      console.log(`\n🔒 WebSocket closed with code ${code}: ${reason}`);
      console.log(`📊 Final ReadyState: ${ws.readyState}`);
      console.log();
      console.log('✅ All tests completed successfully!');

      // Exit after cleanup
      setTimeout(async () => {
        await cycleTLS.exit();
        process.exit(0);
      }, 1000);
    });

    ws.on('error', (error) => {
      console.error(`❌ WebSocket error: ${error.message}`);
    });

  } catch (error) {
    console.error(`❌ Test failed: ${error.message}`);
    console.error(error.stack);
    process.exit(1);
  }
}

// Run the test
testWebSocket();
