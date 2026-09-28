const { spawn } = require('node:child_process');
const { prepare } = require('./build.cjs');
prepare()
  .then((build) => {
    const child = spawn(require('electron'), [build.main], { env: build.env, stdio: 'inherit' });
    child.once('exit', () => build.cleanup().catch(console.error));
    child.once('error', (error) => {
      console.error(error);
      build.cleanup().catch(console.error);
      process.exitCode = 1;
    });
  })
  .catch((error) => {
    console.error(error);
    process.exitCode = 1;
  });
