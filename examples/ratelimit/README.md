# Rate Limit Example

This example demonstrates how to rate limit HTTP and HTTPS traffic passing through a goproxy server.

## Usage

1. Run the proxy server:
   ```bash
   go run main.go
   ```

2. Configure your browser or application to use the proxy at `http://localhost:8080`.

All HTTP and HTTPS traffic will be rate-limited to 100 KB/s.
