module.exports = {
  preset: 'ts-jest',
  testEnvironment: 'node',
  moduleNameMapper: {
    // Map test-utils.js to test-utils (TypeScript file)
    '^(.*)/test-utils\\.js$': '$1/test-utils',
    // Map ./helpers.js to ./helpers for tlsfingerprint tests
    '^(.*)/helpers\\.js$': '$1/helpers'
  }
};

global.performance = {
  now: () => Date.now(),
};