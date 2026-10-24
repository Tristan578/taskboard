#!/usr/bin/env node
const { spawn } = require('node:child_process');
const path = require('node:path');
const fs = require('node:fs');
const binary = path.join(__dirname, '..', 'bin', process.platform === 'win32' ? 'player2-kanban.exe' : 'player2-kanban');
if (!fs.existsSync(binary)) { console.error('Player2 binary is missing. Reinstall player2-kanban and check installer output.'); process.exit(1); }
const child = spawn(binary, process.argv.slice(2), {stdio:'inherit'});
child.on('error', error => {console.error(error.message); process.exitCode=1;});
child.on('exit', code => {process.exitCode=code ?? 1;});
for (const signal of ['SIGINT','SIGTERM']) process.on(signal,()=>child.kill(signal));
