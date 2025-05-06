# Executron

Executron is a secure code execution service that allows running untrusted JavaScript and Python code in isolated containers.

## Architecture

Executron consists of two main components:

1. **HTTP Server**: A Go-based HTTP server that accepts code execution requests and returns the results.
2. **Container Runtime**: Uses Docker with gVisor (runsc) for secure isolation of code execution.

The application follows these steps to execute code:
1. Receives code via HTTP POST request
2. Creates temporary files for the user code and wrapper script
3. Launches a Docker container with gVisor isolation
4. Executes the code with strict resource limits
5. Captures the output and returns it as JSON

## Security Features

- **gVisor Runtime**: Uses Google's gVisor (runsc) container runtime for enhanced isolation between the host and containers
- **Resource Limits**: Strict memory (128MB), CPU (0.5 cores), and process limits for script execution
- **Read-only Filesystem**: Containers run with a read-only root filesystem
- **Capability Dropping**: All Linux capabilities are dropped
- **Execution Timeout**: Code execution is limited to 5 seconds

## Example Usage

### Request

```bash
curl -X POST http://localhost:8080/run \
  -H "Content-Type: application/json" \
  -d '{
    "language": "javascript",
    "code": "const request = await fetch(\"https://7tv.io/v3/emote-sets/01JF0D8JKKR4MZBMPN14FBAR77\"); const response = await request.json(); const randomEmotes = _.sampleSize(response.emotes, 3); return randomEmotes.map(e => e.name).join(\" \");"
  }'
```

### Response

```json
{
  "output": "Stare HACKERMANS PETTHEPEEPO"
}
```

## Deployment

See `PRODUCTION.md` for deployment instructions, including gVisor configuration.