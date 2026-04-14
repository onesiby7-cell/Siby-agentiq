#!/usr/bin/env node

const { spawn } = require('child_process');
const path = require('path');
const os = require('os');
const fs = require('fs');

const VERSION = '2.0.0';
const BINARY_NAME = process.platform === 'win32' ? 'siby-agentiq.exe' : 'siby-agentiq';

function getBinaryPath() {
    const platform = process.platform;
    const arch = process.arch;
    
    const binaryDir = path.join(__dirname, 'bin', `${platform}-${arch}`);
    const binaryPath = path.join(binaryDir, BINARY_NAME);
    
    if (fs.existsSync(binaryPath)) {
        return binaryPath;
    }
    
    const npmPrefix = process.env.NODE_PATH || path.join(os.homedir(), '.siby', 'bin');
    const fallbackPath = path.join(npmPrefix, BINARY_NAME);
    
    if (fs.existsSync(fallbackPath)) {
        return fallbackPath;
    }
    
    console.error('\x1b[91m[SIBY-AGENTIQ]\x1b[0m Binary not found. Run: npm install -g siby-agentiq');
    process.exit(1);
}

function showBanner() {
    console.log(`
\x1b[93m   _____ _____ ____  __  __   ___ _    _____ _  __
  |_   _| ____|  _ \\|  \\/  | |_ _|__|___ /| |/ /___ _   _____ _ __ 
    | | |  _| | |_) | |\\/| |  | |/ __|_ \\| ' // _ \\ | / / _ \\ '__|
    | | | |___|  _ <| |  | |  | |\\__ \\__) | . \\  __/ | \\ \\  __/ |   
    |_| |_____|_| \\_\\_|  |_| |___|___/____|_|\\_\\___|_| \\_|\\___|_|   
\x1b[0m                                                                        
\x1b[96m                      The Last Agent You Will Ever Need\x1b[0m
\x1b[93m                      Built with ❤️ by Ibrahim Siby 🦂\x1b[0m
    `);
}

function showHelp() {
    console.log(`
\x1b[96mSIBY-AGENTIQ Commands:\x1b[0m

  \x1b[92msiby\x1b[0m                    Start interactive TUI mode
  \x1b[92msiby --help\x1b[0m             Show this help
  \x1b[92msiby --version\x1b[0m          Show version
  \x1b[92msiby scan\x1b[0m                Scan current project
  \x1b[92msiby ask <question>\x1b[0m     Ask a question
  \x1b[92msiby init\x1b[0m                Initialize new project
  \x1b[92msiby evolve\x1b[0m             Run evolution-core analysis
  \x1b[92msiby leader-siby\x1b[0m        Activate GOD-IA mode (secret)

\x1b[96mExamples:\x1b[0m
  \x1b[90m$ siby scan\x1b[0m
  \x1b[90m$ siby ask "How do I optimize this API?"\x1b[0m
  \x1b[90m$ siby evolve --nightly\x1b[0m

\x1b[96mDocumentation:\x1b[0m https://docs.siby-agentiq.io
\x1b[96mSupport:\x1b[0m https://github.com/siby-agentiq/siby-agentiq
    `);
}

function main() {
    const args = process.argv.slice(2);
    
    if (args.includes('--help') || args.includes('-h')) {
        showBanner();
        showHelp();
        return;
    }
    
    if (args.includes('--version') || args.includes('-v')) {
        console.log(`\x1b[92msiby-agentiq v${VERSION}\x1b[0m`);
        console.log(`\x1b[90mBuilt with ❤️ by Ibrahim Siby 🦂\x1b[0m`);
        return;
    }
    
    if (args.includes('leader-siby')) {
        console.log('\x1b[93m');
        console.log('╔══════════════════════════════════════════════════════════════╗');
        console.log('║  🦂🦂🦂 GOD-IA OMNISCIENT MODE ACTIVATED 🦂🦂🦂              ║');
        console.log('║                                                              ║');
        console.log('║  Welcome, Creator Ibrahim Siby.                               ║');
        console.log('║  All seeing. All knowing. Sovereign mind engaged.               ║');
        console.log('╚══════════════════════════════════════════════════════════════╝');
        console.log('\x1b[0m');
    } else if (args.length === 0 || !args[0].startsWith('-')) {
        showBanner();
    }
    
    const binaryPath = getBinaryPath();
    
    const childEnv = {
        ...process.env,
        SIBY_VERSION: VERSION,
        SIBY_CREATOR: 'Ibrahim Siby',
        SIBY_SIGNATURE: 'Built with ❤️ by Ibrahim Siby 🦂'
    };
    
    const child = spawn(binaryPath, args, {
        stdio: 'inherit',
        env: childEnv,
        shell: process.platform === 'win32'
    });
    
    child.on('exit', (code) => {
        process.exit(code || 0);
    });
    
    child.on('error', (err) => {
        console.error(`\x1b[91m[SIBY-AGENTIQ]\x1b[0m Error: ${err.message}`);
        process.exit(1);
    });
}

if (require.main === module) {
    main();
}

module.exports = { main, showBanner, showHelp };
