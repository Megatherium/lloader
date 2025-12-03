# Lload - because sometimes you just need to feed it down the llama's throat

A powerful TUI (Terminal User Interface) frontend for llama.cpp written in Go. Lloader provides an intuitive interface for managing, discovering, and running large language models with both local and remote model support.

## Confestion

The code is 95%+ generated just because I was playing with different agents.

## Features

### Core Functionality

- **Interactive Model Selection**: Browse and select from local llama.cpp models
- **Dual Mode Operation**: Run models in server mode or interactive CLI mode
- **Real-time Output**: Live output display from subprocesses with scrolling support
- **Session Configuration**: Runtime overrides for GPU layers (NGL) and context size

### HuggingFace Integration

- **Model Discovery**: Search and browse thousands of models on HuggingFace Hub
- **Quantization Selection**: Choose from available GGUF quantizations (Q4_K_M, IQ4_NL, F16, etc.)
- **Model Information**: Detailed model metadata including downloads, likes, architecture, and licensing
- **Automatic Downloads**: Models are downloaded automatically when selected

### User Interface

- **Tabbed Interface**: Switch between Local and HuggingFace model sources
- **Modal Dialogs**: Configuration modals, quantization selection, and model details
- **Keyboard Navigation**: Full keyboard control with intuitive shortcuts
- **Responsive Design**: Adapts to terminal window size

### Configuration & Logging

- **YAML Configuration**: Flexible configuration with environment variable support
- **Structured Logging**: Powered by Uber's zap logger with configurable levels
- **Modern CLI**: Built with Cobra for a professional command-line interface

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/Megatherium/lloader.git
cd lloader

# Build the binary
make build

# Or install to GOPATH
make install
```

### Using Go Install

```bash
go install github.com/Megatherium/lloader@latest
```

### Pre-built Binaries

Download pre-built binaries for Linux, macOS, and Windows from the [releases page](https://github.com/Megatherium/lloader/releases).

## Configuration

Lloader supports multiple configuration methods (in order of precedence):

1. **Command-line flags**: Override any setting for the current session
2. **Environment variables**: `LLOADER_MODELS_DIR`, `LLOADER_LOG_LEVEL`, etc.
3. **Config file**: `~/.config/lloader/config.yaml` or `/etc/lloader/config.yaml`

### Configuration Options

```yaml
# Models directory (can be overridden with --models-dir flag)
models_dir: "/home/user/models"

# Default GPU layers (0 = CPU only)
default_ngl: 99

# Default context size (0 = model default)
default_ctx_size: 0

# Logging configuration
log_level: "info" # debug, info, warn, error
log_file: "" # empty = stderr

# Command templates for llama.cpp
server_template: "llama-server -m {model_path} -ngl {ngl} -c {ctx_size}"
cli_template: "llama-cli -m {model_path} -ngl {ngl} -c {ctx_size}"
```

See `config/config.yaml.example` for a complete example.

## Usage

### Interactive TUI Mode (Default)

```bash
lload
```

This launches the full-featured terminal interface with two main tabs:

#### Local Models Tab (Tab 1)

- Navigate local models with `↑/↓` arrow keys
- Press `Enter` to start llama-server mode
- Press `c` to start interactive CLI mode
- Press `e` to configure session parameters (NGL, context size)

#### HuggingFace Models Tab (Tab 2)

- Press `/` to search for models on HuggingFace Hub
- Browse search results with `↑/↓` arrows
- Press `Enter` or `c` to select a model and choose quantization
- Press `i` to view detailed model information
- Models are automatically downloaded when selected

### Global Controls

- `1/2` - Switch between Local and HuggingFace tabs
- `Tab` - Switch focus between model list and output panes
- `Ctrl+L` - Clear output pane
- `Ctrl+C` or `q` - Quit application

### Interactive CLI Mode

When running in CLI mode:

- Type your prompts and press `Enter` to send
- Press `Esc` to exit CLI mode
- Full conversation history is maintained

### Command Line Commands

```bash
# Interactive TUI (default)
lload

# List available local models
lload list

# Show current configuration
lload config

# Show version information
lload version

# Custom models directory
lload --models-dir /path/to/models

# Custom config file
lload --config /path/to/config.yaml

# Verbose logging
lload --verbose
```

## Dependencies

### Core Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Elegant terminal user interface framework
- [Cobra](https://github.com/spf13/cobra) - CLI applications with commands and flags
- [Viper](https://github.com/spf13/viper) - Configuration management with multiple sources
- [Zap](https://github.com/uber-go/zap) - High-performance structured logging
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Terminal styling and layout
- [Bubbles](https://github.com/charmbracelet/bubbles) - Common Bubble Tea components

### HuggingFace Integration

- [hf-go](https://github.com/Megatherium/hf-go) - Go client for HuggingFace Hub API

## License

GPL v3

## Acknowledgments

- [llama.cpp](https://github.com/ggerganov/llama.cpp) - The underlying inference engine
- [HuggingFace](https://huggingface.co) - Model hosting and discovery platform
- [Charm](https://charm.sh) - Terminal UI libraries and tools
- [OpenCode](https://github.com/sst/opencode) - AI-powered software engineering assistant
- [KiloCode](https://github.com/Kilo-Org/kilocode) - Advanced code intelligence platform
- [Crush](https://github.com/charmbracelet/crush) - The prettiest AI agent on the planet
- [Droid](https://factory.ai) - Another nice one
- Grok Fast Code - High-performance code generation model
- Claude Opus - Advanced language model by Anthropic
- MiniMax-M2 - Efficient multimodal AI model
- Deepseek V3.2-Special - Specialized deep learning model
