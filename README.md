# 🟣 EMBER

A terminal-based text embeddings demonstration tool that showcases how semantic similarity works using OpenAI's text embedding models.

Uses OpenAI's API to embed and compare similarity of text using the `text-embedding-3-small` model and cosine similarity.


## Screenshots


## Installation

### Option 1: Download Prebuilt Binary

Download the latest release for your platform from the [Releases page](../../releases).

Available for:
- Linux (x64, ARM64)
- macOS (Intel, Apple Silicon)
- Windows (x64)

### Option 2: Build from Source

**Prerequisites:**
- Go 1.21 or later

**Build:**
```bash
git clone <this-repo>
cd ember
go build -o ember .
```

## Usage

### Setup

You need an OpenAI API key to generate embeddings:

```bash
export OPENAI_API_KEY="your-api-key-here"
```

Get an API key from [OpenAI's platform](https://platform.openai.com/api-keys).

### Running

```bash
ember
```

## License

MIT License - see [LICENSE](LICENSE) file for details.

