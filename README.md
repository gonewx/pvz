# Plants vs. Zombies AI Recreation

> **A backend developer with ZERO game development experience** recreated "Plants vs. Zombies" using [BMAD-METHOD](https://github.com/bmad-code-org/BMAD-METHOD) + AI (Claude Code), achieving **120,000+ lines of Go code** with **99.9% AI-generated content**.

| Metric | Value |
|--------|-------|
| Developer Background | Backend engineer, no game dev experience |
| AI Code Generation Rate | 99.9% |
| Go Source Code | 120,000+ lines (388 files) |
| Development Approach | BMAD-METHOD workflow + Claude Code |
| Manual Intervention | Architecture decisions, prompting, review |

This project demonstrates that with proper AI workflow management (BMAD-METHOD), even developers without domain expertise can build complex, production-quality applications. The BMAD-METHOD provides structured PRD → Architecture → Epic → Story workflows that guide AI to generate consistent, maintainable code.

[![CI](https://github.com/gonewx/pvz/actions/workflows/ci.yml/badge.svg)](https://github.com/gonewx/pvz/actions/workflows/ci.yml)
[![Release](https://github.com/gonewx/pvz/actions/workflows/release.yml/badge.svg)](https://github.com/gonewx/pvz/actions/workflows/release.yml)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

[简体中文](README_zh.md) | English

---

## 🎮 Introduction

This project is a faithful recreation of the classic tower defense game "Plants vs. Zombies" (PC Chinese Anniversary Edition), focusing on learning and practicing Go game development. It adopts the modern Entity-Component-System (ECS) architecture pattern, implements type-safe game logic using Go generics, and aims for 100% reproduction of the original front yard daytime level experience.

**Key Features:**
- 🏗️ **ECS Architecture** - Type-safe ECS framework based on Go generics
- 🎨 **Original Animations** - Complete implementation of the Reanim skeletal animation system
- ✨ **Particle Effects** - XML configuration-driven particle system
- 📊 **Data-Driven** - YAML configuration files for game data management
- 🎯 **High Fidelity** - Precise recreation of original game values and behaviors

## ✨ Features

### Implemented Features

#### Core Systems
- ✅ **Game Framework** - Scene management, state machine, main loop
- ✅ **Resource Management** - Unified image, audio, and configuration loading system
- ✅ **Sun System** - Sky drops and sunflower production
- ✅ **Plant System** - Planting, cooldown, card selection
- ✅ **Zombie System** - AI, movement, attack, health
- ✅ **Combat System** - Projectile collision, damage calculation
- ✅ **Level System** - Wave management, progress bar, win/lose detection

#### Animations & Effects
- ✅ **Reanim Animation System** - Original skeletal animation with part transformations
- ✅ **Particle Effects System** - Explosions, splashes, dirt particles, etc.
- ✅ **Animation Composition** - Multi-animation overlay, track binding, parent-child offsets
- ✅ **Configuration-Driven Animations** - YAML configuration for animation compositions

#### Plants (Chapter 1)
- 🌻 Sunflower
- 🌱 Peashooter
- 🛡️ Wall-nut
- 💣 Cherry Bomb
- 💥 Potato Mine
- ❄️ Snow Pea
- 🌱 Repeater
- 🪴 Chomper

#### Zombie Types (Chapter 1)
- 🧟 Normal Zombie
- 🚧 Conehead Zombie
- 🪣 Buckethead Zombie
- 🚩 Flag Zombie
- 📰 Newspaper Zombie
- 🏃 Pole Vaulting Zombie

#### Level Content
- ✅ **Chapter 1 (Front Yard Daytime)** - Levels 1-1 to 1-10
- ✅ **Tutorial System** - 1-1 single-row lawn guidance
- ✅ **Special Level** - 1-5 Wall-nut Bowling
- ✅ **Conveyor Level** - 1-10 Conveyor belt mechanics
- ✅ **Opening Animation** - Camera pan, zombie preview
- ✅ **Card Selection Screen** - Plant selection, unlock system

#### UI & Experience
- ✅ **Main Menu System** - Start Adventure, Quit Game
- ✅ **Pause Menu** - Continue, Restart, Return to Main Menu
- ✅ **Shovel Tool** - Remove plants
- ✅ **Lawn Mower Defense** - Last line of defense mechanism
- ✅ **Level Progress Bar** - Flags, final wave notification

## 🚀 Quick Start

### Requirements

- **Go Version**: 1.24 or higher
- **Operating System**: Windows / macOS / Linux / Android / WASM
- **Memory**: At least 2GB RAM
- **Graphics**: OpenGL 2.1+ support

### Installation & Running

```bash
# 1. Clone the repository
git clone https://github.com/gonewx/pvz
cd pvz

# 2. Download dependencies
go mod download

# 3. Run the game
go run .
```

The game will launch in an 800x600 window.

### Building Executables

```bash
# Build using Makefile (recommended)
make build                # Build for current platform
make build-linux          # Build for Linux (amd64 + arm64)
make build-windows        # Build for Windows (amd64 + arm64)
make build-darwin         # Build for macOS (requires macOS host)
make build-wasm           # Build for WebAssembly

# Manual build
go build -o pvz-go .

# Build optimized version (smaller size)
go build -ldflags="-s -w" -o pvz-go .
```

### Building Release Version with Icons

```bash
# Generate Windows icon resources (.syso)
make generate-icons

# Package Linux release (with icons and .desktop)
make package-linux

# Build macOS .app bundle (requires macOS)
make build-darwin-app

# Build Android APK
make build-apk

# View iOS icon usage instructions
make ios-icons-info
```

See **[Quick Start Guide](docs/quickstart.md)** for detailed instructions.

## 📖 Documentation

### User Documentation
- **[Quick Start Guide](docs/quickstart.md)** - Get started in 5 minutes
- **[User Manual](docs/user-guide.md)** - Game controls and features

### Developer Documentation
- **[Development Guide](docs/development.md)** - Code contribution and development guidelines
- **[Product Requirements Document (PRD)](docs/prd.md)** - Complete feature specifications
- **[Architecture Document](docs/architecture.md)** - Technical architecture design

> **Note**: `CLAUDE.md` provides development context for the Claude Code AI tool, containing technical details about ECS architecture, Reanim system, etc., primarily for developers.

## 🏗️ Project Structure

```
pvz/
├── main.go                 # Game entry point
├── assets/                 # Game resources
│   ├── images/             # Image resources (spritesheets)
│   ├── audio/              # Audio resources
│   ├── fonts/              # Font files
│   ├── effect/             # Particle configurations
│   └── icons/              # Application icons (multi-platform)
│       ├── windows/        # Windows ico and png
│       ├── macos/          # macOS iconset
│       ├── linux/          # Linux multi-size png
│       ├── ios/            # iOS AppIcon.appiconset
│       ├── android/        # Android mipmap icons
│       └── web/            # Web favicon and PWA icons
├── data/                   # Externalized game data
│   ├── levels/             # Level configurations (YAML)
│   ├── reanim/             # Reanim animation definitions
│   └── reanim_config.yaml  # Animation configuration
├── pkg/                    # Core code library
│   ├── components/         # All component definitions
│   ├── entities/           # Entity factory functions
│   ├── systems/            # All system implementations
│   ├── scenes/             # Game scenes
│   ├── ecs/                # ECS framework core
│   ├── game/               # Game core managers
│   ├── utils/              # Common utility functions
│   └── config/             # Configuration loading and management
├── scripts/                # Build scripts
│   ├── build-apk.sh        # Android APK build
│   ├── Info.plist          # macOS app configuration
│   └── pvz.desktop         # Linux desktop entry
├── docs/                   # Documentation
└── .meta/                  # Reference materials and metadata
```

## 🎯 Tech Stack

- **Language**: Go 1.21+
- **Game Engine**: [Ebitengine v2](https://ebiten.org/)
- **Architecture Pattern**: Entity-Component-System (ECS)
- **Configuration Format**: YAML
- **Testing Framework**: Go native testing

### Core Technical Highlights

1. **Go Generics ECS** - Compile-time type safety, 10-30% performance improvement
2. **Reanim Skeletal Animation** - 100% reproduction of original animation system
3. **Data-Driven Design** - All game values externalized in configuration
4. **High-Performance Particle System** - DrawTriangles batch rendering

## 🎮 Game Controls

### Basic Controls
- **Left Mouse Button** - Collect sun, select plants, place plants
- **Right Mouse Button** - Cancel plant selection
- **ESC Key** - Pause/Resume game
- **--verbose** - Enable verbose logging (debug)

### Gameplay Flow
1. Select "Start Adventure" from the main menu
2. Choose plants on the card selection screen (up to 6-10)
3. Wait for sun, select a plant card
4. Click on lawn grid to plant
5. Defend against zombies, complete all waves

See **[User Manual](docs/user-guide.md)** for detailed instructions.

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 🤝 Contributing

Contributions are welcome! This project is primarily for learning Go game development.

### Contribution Process
1. Fork this repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Create a Pull Request

See **[Development Guide](docs/development.md)** for detailed guidelines.

## 📊 Project Status

### MVP Scope
✅ **Completed** - Full front yard daytime experience (all 10 levels in Chapter 1)

### Completed Epics
- ✅ Epic 1: Game Basic Framework and Main Loop
- ✅ Epic 2: Core Resources and Player Interaction
- ✅ Epic 3: Plant System and Deployment
- ✅ Epic 4: Basic Zombies and Combat Logic
- ✅ Epic 5: Game Flow and Advanced Units
- ✅ Epic 6: Reanim Animation System Migration
- ✅ Epic 7: Particle Effects System
- ✅ Epic 8: Chapter 1 Level Implementation
- ✅ Epic 9: ECS Framework Generics Refactoring
- ✅ Epic 10: Game Experience Improvements
- ✅ Epic 11: Level UI Enhancement
- ✅ Epic 12: Main Menu System
- ✅ Epic 13: Reanim Animation System Modern Refactoring

### Future Plans
- 🔄 Epic 14+: More levels and features (to be planned)

## 📜 License

This project is for learning and technical research purposes only.

See [DISCLAIMER.md](DISCLAIMER.md) for important legal notices.

## 🙏 Acknowledgments

- **Workflow Engine**: [BMAD-METHOD](https://github.com/bmad-code-org/BMAD-METHOD)
- **Original Game**: "Plants vs. Zombies" by PopCap Games
- **Game Engine**: [Ebitengine](https://ebiten.org/) team
- **Development Tool**: Claude Code AI

## 📞 Contact

For questions or suggestions, please contact us through:
- Submit an [Issue](../../issues)
- Start a [Discussion](../../discussions)

---

**Notice**: This project is for learning and technical research only, not for commercial use. All game resource copyrights belong to their original authors.
