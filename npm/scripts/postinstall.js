#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const os = require('os');

console.log(`
\x1b[93m╔════════════════════════════════════════════════════════════════╗
\x1b[93m║                                                                ║
\x1b[93m║   🦂 SIBY-AGENTIQ v2.0.0 INSTALLATION 🦂                      ║
\x1b[93m║                                                                ║
\x1b[93m║   "The Last Agent You Will Ever Need"                          ║
\x1b[93m║                                                                ║
\x1b[93m║   Built with ❤️ by Ibrahim Siby 🇬🇳                           ║
\x1b[93m║                                                                ║
\x1b[93m╚════════════════════════════════════════════════════════════════╝
\x1b[0m`);

const homeDir = os.homedir();
const sibyDir = path.join(homeDir, '.siby');
const configFile = path.join(sibyDir, 'config.json');

if (!fs.existsSync(sibyDir)) {
    fs.mkdirSync(sibyDir, { recursive: true });
}

if (!fs.existsSync(configFile)) {
    const defaultConfig = {
        version: "2.0.0",
        creator: "Ibrahim Siby",
        signature: "Built with ❤️ by Ibrahim Siby 🦂",
        providers: {
            default: "ollama",
            fallback: ["groq", "anthropic", "openai"]
        },
        evolution: {
            enabled: true,
            nightlyMode: false,
            autoLearn: true
        },
        godIA: {
            secretCommand: "leader-siby"
        },
        scorpion: {
            enabled: true,
            providers: ["gemini", "gpt-4o", "perplexity"]
        },
        hologram: {
            enabled: false,
            theme: "cyberpunk"
        },
        voice: {
            enabled: false,
            wakeWord: "Siby"
        },
        cloudSync: {
            enabled: false,
            encryption: "aes-256-gcm"
        }
    };
    
    fs.writeFileSync(configFile, JSON.stringify(defaultConfig, null, 2));
    console.log('\x1b[92m✓\x1b[0m Configuration created at: ' + configFile);
}

console.log(`
\x1b[96mQuick Start:\x1b[0m
  \x1b[90m$ siby\x1b[0m                  # Start interactive mode
  \x1b[90m$ siby ask "Hello!"\x1b[0m     # Ask a question
  \x1b[90m$ siby --help\x1b[0m            # Show all commands

\x1b[96mFeatures:\x1b[0m
  🦂 Scorpion: Deep web search with multi-API queries
  🧬 Evolution-Core: Self-learning from every interaction
  👁️ GOD-IA: Type \x1b[93mleader-siby\x1b[0m to activate (secret mode)
  🌈 Hologram: Visual ASCII art mode
  🎤 Voice: Voice commands (coming soon)
  ☁️ Cloud Sync: Sync memory across devices

\x1b[96mLearn More:\x1b[0m
  \x1b[90mhttps://docs.siby-agentiq.io\x1b[0m
  \x1b[90mhttps://github.com/siby-agentiq/siby-agentiq\x1b[0m

\x1b[93m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  🦂 Built with ❤️ by Ibrahim Siby • République de Guinée 🇬🇳
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\x1b[0m
`);

module.exports = {};
