# Siby-Terminal

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/License-MIT-green?style=for-the-badge" alt="License">
  <img src="https://img.shields.io/badge/Platform-Linux%20%7C%20Mac%20%7C%20Windows-blue?style=for-the-badge" alt="Platform">
  <img src="https://img.shields.io/badge/AI-Ollama%20%7C%20Claude%20%7C%20GPT--4-yellow?style=for-the-badge" alt="AI">
</p>

<p align="center">
  <strong>🚀 L'agent IA de programmation qui fonctionne à 100% en local.</strong>
</p>

---

## Pourquoi Siby-Terminal est plus rapide qu'OpenCode ?

| Critère | OpenCode | Siby-Terminal |
|---------|----------|---------------|
| **Latence** | Dépendance cloud | **Zéro latence réseau** (Ollama local) |
| **Vie privée** | Données envoyées au cloud | **100% local** |
| **Mode Offline** | ❌ | **✓ Fonctionne sans internet** |
| **Personnalisation LLM** | Fixe | **Tous les modèles Ollama** |
| **Ressources** | Utilise ton GPU/API externe | **GPU local exploité** |
| **Coût** | Facture API | **Gratuit** (Ollama) |
| **Multi-fichiers** | Limité | **✓ Édition parallèle** |
| **Planification** | Basique | **Chain of Thought avancé** |

---

## 🚀 Installation en 30 secondes

### Linux / Mac

```bash
curl -fsSL https://raw.githubusercontent.com/siby-agentiq/siby-terminal/main/install.sh | bash
```

### Windows

```powershell
irm https://raw.githubusercontent.com/siby-agentiq/siby-terminal/main/install.ps1 | iex
```

### Depuis les sources

```bash
git clone https://github.com/siby-agentiq/siby-terminal.git
cd siby-terminal
make install-home
```

---

## ⚡ Démarrage rapide

### Étape 1 : Installer Ollama (optionnel)

```bash
# Linux/Mac
curl -fsSL https://ollama.ai/install.sh | sh

# Windows
winget install Ollama.Ollama

# Télécharger un modèle
ollama pull llama3.2
```

### Étape 2 : Lancer Siby

```bash
siby
```

### Étape 3 : Coder

```
❯ Crée un serveur HTTP en Go sur le port 8080
```

---

## 📋 Commandes

| Commande | Description |
|----------|-------------|
| `/help` | Affiche l'aide |
| `/clear` | Efface l'historique |
| `/model [nom]` | Change de provider |
| `/scan` | Analyse le projet |
| `/exec <cmd>` | Exécute terminal |
| `/quit` | Quitte |

---

## 🔧 Configuration

```yaml
# ~/.config/siby/config.yaml
providers:
  primary: "ollama"
  ollama:
    base_url: "http://localhost:11434"
    model: "llama3.2:latest"
chain_of_thought:
  enabled: true
  reasoning_depth: "deep"
```

---

## 🏗️ Architecture

```
siby-agentiq/
├── cmd/                  # Entry point
├── internal/
│   ├── config/          # Configuration
│   ├── provider/        # LLM providers
│   │   ├── autconfig.go # Auto-détection
│   │   ├── manager.go  # Smart switching
│   │   └── chain.go    # Chain of Thought
│   ├── scanner/        # Analyse projet
│   ├── executor/        # Multi-file editing
│   └── ui/             # TUI Bubble Tea
├── install.sh           # Linux/Mac
├── install.ps1          # Windows
└── Makefile
```

---

## 📊 Comparaison Providers

| Provider | Coût | Vitesse | Vie privée |
|----------|------|---------|------------|
| Ollama (local) | Gratuit | ⚡⚡⚡ | 100% ✓ |
| Claude 4 | Payant | ⭐⭐ | Cloud |
| GPT-4o | Payant | ⭐⭐⭐ | Cloud |

---

## 🛠️ Développement

```bash
git clone https://github.com/siby-agentiq/siby-terminal.git
cd siby-terminal

go mod tidy
make dev      # Développement
make build    # Compilation
make test     # Tests
```

---

## 📄 Licence

MIT License
