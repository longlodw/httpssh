# SSH HTTP Reverse Proxy

This project implements a reverse proxy that allows HTTP services to be accessed through SSH connections instead of direct HTTP.
The SSH server authenticates users and injects JWT tokens into forwarded requests to identify the user.

## Features

- SSH-based access to HTTP backends
- User authentication via public key or password
- JWT token injection for user identification
- Prometheus metrics
- Support for multiple allowed backends

## Configuration

The server is configured via a JSON file (default: `config.json`). Example configuration:

```json
{
  "ssh_listen_addr": "0.0.0.0:2222",
  "prometheus_listen_addr": "0.0.0.0:9090",
  "key_listen_addr": "0.0.0.0:8080",
  "jwt_key_path": "./jwt_key.pem",
  "ssh_key_path": "./ssh_key.pem",
  "authorization_end_point": "https://auth.example.com/authorize",
  "allowed_backends": [
    "https://google.com:443"
  ],
  "no_auth": true
}
```

### Configuration Fields

- `ssh_listen_addr`: Address for SSH server to listen on
- `prometheus_listen_addr`: Address for Prometheus metrics server
- `key_listen_addr`: Address for JWT public key server
- `jwt_key_path`: Path to Ed25519 private key for JWT signing
- `ssh_key_path`: Path to Ed25519 private key for SSH host key
- `authorization_end_point`: Authorization endpoint for user authentication, it will send username and password as query parameters or username and public key as query parameters
- `allowed_backends`: List of allowed backend URLs
- `no_auth`: Disable authentication if true

## Usage

1. Generate Ed25519 keys for SSH and JWT:
   ```bash
   openssl genpkey -algorithm Ed25519 -out jwt_key.pem
   openssl genpkey -algorithm Ed25519 -out ssh_key.pem
   ```

2. Configure `config.json` with your settings

3. Run the server:
   ```bash
   go run .
   ```

4. Connect via SSH and use port forwarding to access backends:
   ```bash
   ssh -L 8080:localhost:80 user@your-server -p 2222
   ```

   Then access `http://localhost:8080` which will proxy to the configured backend with JWT authentication.

## How It Works

1. Users connect to the SSH server
2. Authentication is performed via password/public key against the configured endpoint
3. SSH port forwarding requests are accepted for allowed backends
4. HTTP requests are forwarded to the backend with an `X-Identity` header containing a JWT token
5. The JWT token identifies the user and is signed with the configured key
6. The JWT public key is available at the `/key` endpoint for verification

## Metrics

Prometheus metrics are exposed at the configured `prometheus_listen_addr` with request counts and other metrics.

## License
MIT
