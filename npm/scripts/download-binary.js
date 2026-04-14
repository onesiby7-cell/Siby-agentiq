#!/usr/bin/env node

const https = require('https');
const http = require('http');
const fs = require('fs');
const path = require('path');
const os = require('os');

const VERSION = '2.0.0';
const RELEASES_URL = 'https://api.github.com/repos/siby-agentiq/siby-agentiq/releases/latest';

const BINARY_NAMES = {
    'darwin-x64': 'siby-agentiq-darwin-x64',
    'darwin-arm64': 'siby-agentiq-darwin-arm64',
    'linux-x64': 'siby-agentiq-linux-x64',
    'linux-arm64': 'siby-agentiq-linux-arm64',
    'win32-x64': 'siby-agentiq-win32-x64.exe',
};

function getPlatformKey() {
    const platform = process.platform;
    const arch = process.arch;
    
    if (platform === 'darwin') {
        return arch === 'arm64' ? 'darwin-arm64' : 'darwin-x64';
    } else if (platform === 'linux') {
        return arch === 'arm64' ? 'linux-arm64' : 'linux-x64';
    } else if (platform === 'win32') {
        return 'win32-x64';
    }
    
    throw new Error(`Unsupported platform: ${platform}-${arch}`);
}

function getBinaryName() {
    const key = getPlatformKey();
    return BINARY_NAMES[key] || 'siby-agentiq';
}

function getDownloadUrl(release) {
    const platformKey = getPlatformKey();
    const binaryName = getBinaryName();
    
    for (const asset of release.assets) {
        if (asset.name === binaryName || asset.name.startsWith(binaryName.split('.')[0])) {
            return asset.browser_download_url;
        }
    }
    
    const key = getPlatformKey();
    const ext = key.startsWith('win') ? '.exe' : '';
    return `https://github.com/siby-agentiq/siby-agentiq/releases/download/v${VERSION}/siby-agentiq-${key}${ext}`;
}

function downloadFile(url, dest) {
    return new Promise((resolve, reject) => {
        const file = fs.createWriteStream(dest);
        
        const protocol = url.startsWith('https') ? https : http;
        
        const request = protocol.get(url, {
            headers: {
                'User-Agent': 'siby-agentiq-npm'
            }
        }, (response) => {
            if (response.statusCode === 302 || response.statusCode === 301) {
                file.close();
                downloadFile(response.headers.location, dest).then(resolve).catch(reject);
                return;
            }
            
            if (response.statusCode !== 200) {
                reject(new Error(`Failed to download: ${response.statusCode}`));
                return;
            }
            
            response.pipe(file);
            
            file.on('finish', () => {
                file.close();
                fs.chmodSync(dest, 0o755);
                console.log(`\x1b[92m✓\x1b[0m Downloaded: ${dest}`);
                resolve();
            });
        });
        
        request.on('error', (err) => {
            fs.unlink(dest, () => {});
            reject(err);
        });
    });
}

async function downloadBinary() {
    const binDir = path.join(__dirname, '..', 'bin');
    fs.mkdirSync(binDir, { recursive: true });
    
    const platformKey = getPlatformKey();
    const binaryName = getBinaryName();
    const destPath = path.join(binDir, binaryName);
    
    if (fs.existsSync(destPath)) {
        console.log(`\x1b[90m✓\x1b[0m Binary already exists: ${destPath}`);
        return;
    }
    
    console.log(`\x1b[96m↓\x1b[0m Downloading Siby-Agentiq v${VERSION} for ${platformKey}...`);
    
    const binaryUrl = `https://github.com/siby-agentiq/siby-agentiq/releases/download/v${VERSION}/${binaryName}`;
    
    try {
        await downloadFile(binaryUrl, destPath);
        console.log(`\x1b[92m✓\x1b[0m Installation complete!`);
        console.log(`\x1b[90m  Run: siby\x1b[0m`);
    } catch (err) {
        console.error(`\x1b[91m✗\x1b[0m Download failed: ${err.message}`);
        console.log(`\x1b[90m  Falling back to build from source...\x1b[0m`);
        console.log(`\x1b[90m  Run: git clone https://github.com/siby-agentiq/siby-agentiq && cd siby-agentiq && make install\x1b[0m`);
    }
}

if (require.main === module) {
    downloadBinary().catch(console.error);
}

module.exports = { downloadBinary, getPlatformKey };
