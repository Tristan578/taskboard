const fs = require('fs');
const path = require('path');
const os = require('os');
const https = require('https');
const { execFileSync } = require('child_process');

const REPO = 'Tristan578/taskboard';
const BINARY_NAME = 'player2-kanban';
const VERSION = require('../package.json').version;
const BASE_URL = `https://github.com/${REPO}/releases/download/v${VERSION}`;
const crypto = require('node:crypto');

const PLATFORM_MAP = {
  linux: 'linux',
  darwin: 'darwin',
  win32: 'windows',
};

const ARCH_MAP = {
  x64: 'amd64',
  arm64: 'arm64',
};

function getBinaryName() {
  return os.platform() === 'win32' ? `${BINARY_NAME}.exe` : BINARY_NAME;
}

function getAssetName() {
  const platform = PLATFORM_MAP[os.platform()];
  const arch = ARCH_MAP[os.arch()];

  if (!platform) {
    throw new Error(`Unsupported platform: ${os.platform()}. Supported: linux, darwin, win32`);
  }
  if (!arch) {
    throw new Error(`Unsupported architecture: ${os.arch()}. Supported: x64, arm64`);
  }

  const suffix = `${platform}-${arch}`;
  const ext = platform === 'windows' ? '.zip' : '.tar.gz';
  return `${BINARY_NAME}-${suffix}${ext}`;
}

function followRedirects(url, redirects = 0) {
  if (redirects > 5 || !url.startsWith("https://")) return Promise.reject(new Error("Invalid download redirect"));
  return new Promise((resolve, reject) => {
    https.get(url, { headers: { 'User-Agent': 'player2-kanban-installer' } }, (res) => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        res.resume();
        followRedirects(new URL(res.headers.location, url).href, redirects + 1).then(resolve, reject);
        return;
      }
      if (res.statusCode === 404) {
        reject(new Error('Release asset not found (404). The release may not exist yet.'));
        return;
      }
      if (res.statusCode !== 200) {
        reject(new Error(`Download failed with status ${res.statusCode}`));
        return;
      }
      resolve(res);
    }).on('error', reject);
  });
}

function download(url, dest) {
  return new Promise((resolve, reject) => {
    followRedirects(url).then((res) => {
      const file = fs.createWriteStream(dest);
      res.pipe(file);
      file.on('finish', () => {
        file.close(resolve);
      });
      file.on('error', (err) => {
        fs.unlink(dest, () => {});
        reject(err);
      });
    }).catch(reject);
  });
}

function verifyChecksum(data, manifest, assetName) {
 const entry=manifest.split(/\r?\n/).map(line=>line.trim().split(/\s+/)).find(parts=>parts[1]===assetName);
 const actual=crypto.createHash('sha256').update(data).digest('hex');
 if(!entry || !/^[a-f0-9]{64}$/.test(entry[0]) || actual!==entry[0]) throw new Error('Release checksum verification failed');
}

function extractTarGz(archive, destDir) {
  execFileSync("tar", ["xzf", archive, "-C", destDir], {stdio:"pipe"});
}

function extractZip(archive, destDir) {
  if (os.platform() === 'win32') {
    // Paths are arguments to PowerShell, never interpolated into executable script.
    const script = "$a=$env:PLAYER2_ARCHIVE; $d=$env:PLAYER2_DEST; Expand-Archive -LiteralPath $a -DestinationPath $d -Force";
    execFileSync('powershell', ['-NoProfile','-NonInteractive','-Command',script], {stdio:'pipe',env:{...process.env,PLAYER2_ARCHIVE:archive,PLAYER2_DEST:destDir}});
  } else {
    execFileSync("unzip", ["-o",archive,"-d",destDir], {stdio:"pipe"});
  }
}

async function install() {
  const assetName = getAssetName();
  const url = `${BASE_URL}/${assetName}`;
  const binDir = path.join(__dirname, '..', 'bin');
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'player2-install-'));

  console.log(`Platform: ${os.platform()} | Arch: ${os.arch()}`);
  console.log(`Downloading ${assetName}...`);

  // Clean up and create directories
  fs.mkdirSync(binDir, { recursive: true });
  fs.mkdirSync(tmpDir, { recursive: true });

  const archivePath = path.join(tmpDir, assetName);

  try {
    await download(url, archivePath);
    const checksumPath=path.join(tmpDir,'SHA256SUMS');
    await download(`${BASE_URL}/SHA256SUMS`,checksumPath);
    verifyChecksum(fs.readFileSync(archivePath),fs.readFileSync(checksumPath,'utf8'),assetName);
  } catch (err) {
    // Clean up temp dir
    fs.rmSync(tmpDir, { recursive: true, force: true });

    if (err.message.includes('404') || err.message.includes('not found')) {
      console.error('');
      console.error('No pre-built binary found for this platform.');
      console.error('This likely means no GitHub Release has been published yet.');
      console.error('');
      console.error('To build from source:');
      console.error('  git clone https://github.com/Tristan578/taskboard.git');
      console.error('  cd taskboard');
      console.error('  make build');
      console.error('');
      throw err; // A missing executable is an installation failure.
    }
    throw err;
  }

  console.log('Extracting...');

  try {
    if (assetName.endsWith('.zip')) {
      extractZip(archivePath, tmpDir);
    } else {
      extractTarGz(archivePath, tmpDir);
    }

    // Find the binary in the extracted contents
    const binaryName = getBinaryName();
    const expectedBinary = path.join(tmpDir, binaryName);

    if (!fs.existsSync(expectedBinary)) {
      // Search one level deep in case it was extracted into a subdirectory
      const entries = fs.readdirSync(tmpDir, { withFileTypes: true });
      let found = false;
      for (const entry of entries) {
        if (entry.isDirectory()) {
          const nested = path.join(tmpDir, entry.name, binaryName);
          if (fs.existsSync(nested)) {
            fs.copyFileSync(nested, path.join(binDir, binaryName));
            found = true;
            break;
          }
        }
      }
      if (!found) {
        throw new Error(`Binary "${binaryName}" not found in archive.`);
      }
    } else {
      fs.copyFileSync(expectedBinary, path.join(binDir, binaryName));
    }

    if (os.platform() !== 'win32') fs.chmodSync(path.join(binDir,binaryName),0o755);

    console.log(`Installed ${binaryName} to ${binDir}`);
  } finally {
    // Clean up temp directory
    fs.rmSync(tmpDir, { recursive: true, force: true });
  }
}

if (require.main === module) install().catch((err) => {
  console.error(`Installation failed: ${err.message}`);
  process.exit(1);
});

module.exports = {getAssetName, getBinaryName, followRedirects, extractTarGz, extractZip, verifyChecksum};
