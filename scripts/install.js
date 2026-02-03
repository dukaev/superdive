#!/usr/bin/env node

const https = require('https');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');

const VERSION = require('../package.json').version;
const REPO = 'dukaev/superdive';

function getPlatform() {
  const platform = os.platform();
  switch (platform) {
    case 'darwin': return 'darwin';
    case 'linux': return 'linux';
    case 'win32': return 'windows';
    default: throw new Error(`Unsupported platform: ${platform}`);
  }
}

function getArch() {
  const arch = os.arch();
  switch (arch) {
    case 'x64': return 'amd64';
    case 'arm64': return 'arm64';
    case 'ia32': return '386';
    default: throw new Error(`Unsupported architecture: ${arch}`);
  }
}

function getDownloadUrl() {
  const platform = getPlatform();
  const arch = getArch();
  const ext = platform === 'windows' ? '.zip' : '.tar.gz';
  return `https://github.com/${REPO}/releases/download/v${VERSION}/superdive_${VERSION}_${platform}_${arch}${ext}`;
}

function download(url, dest) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(dest);

    const request = (url) => {
      https.get(url, (response) => {
        if (response.statusCode === 302 || response.statusCode === 301) {
          request(response.headers.location);
          return;
        }

        if (response.statusCode !== 200) {
          reject(new Error(`Failed to download: ${response.statusCode}`));
          return;
        }

        response.pipe(file);
        file.on('finish', () => {
          file.close();
          resolve();
        });
      }).on('error', (err) => {
        fs.unlink(dest, () => {});
        reject(err);
      });
    };

    request(url);
  });
}

async function install() {
  const platform = getPlatform();
  const binDir = path.join(__dirname, '..', 'bin');
  const binaryName = platform === 'windows' ? 'superdive.exe' : 'superdive';
  const binaryPath = path.join(binDir, binaryName);

  // Skip if binary already exists
  if (fs.existsSync(binaryPath)) {
    console.log('superdive binary already exists');
    return;
  }

  const url = getDownloadUrl();
  const ext = platform === 'windows' ? '.zip' : '.tar.gz';
  const archivePath = path.join(binDir, `superdive${ext}`);

  console.log(`Downloading superdive from ${url}...`);

  try {
    await download(url, archivePath);

    console.log('Extracting...');

    if (platform === 'windows') {
      execSync(`powershell -command "Expand-Archive -Path '${archivePath}' -DestinationPath '${binDir}'"`, { stdio: 'inherit' });
    } else {
      execSync(`tar -xzf "${archivePath}" -C "${binDir}"`, { stdio: 'inherit' });
    }

    // Make binary executable
    if (platform !== 'windows') {
      fs.chmodSync(binaryPath, 0o755);
    }

    // Clean up archive
    fs.unlinkSync(archivePath);

    console.log('superdive installed successfully!');
  } catch (err) {
    console.error(`Failed to install superdive: ${err.message}`);
    console.error('Please install manually from https://github.com/dukaev/superdive/releases');
    process.exit(1);
  }
}

install();
