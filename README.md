# Sentinel - Docker Swarm DNS Failover Manager

Sentinel is a lightweight service that automatically updates DNS records when Docker Swarm leadership changes. It ensures high availability by pointing your domain to the current Swarm leader.

![Sentinel Logo](./images/logo.png)

## Features

- 🔄 Automatic DNS failover for Docker Swarm clusters
- 🔍 Real-time monitoring of Swarm leader changes
- 🌐 DNS record updates via INWX API
- 🔒 Secure and lightweight (built on scratch container)
- 🚀 Easy to deploy and configure

## How It Works

Sentinel runs on each manager node in your Docker Swarm cluster. It:

1. Monitors Docker Swarm events for leadership changes
2. Updates the DNS record to point to the leader node's IP when changes occur

## Use Cases

This project is designed for low-budget environments where virtual IPs are unavailable or when you want to avoid dependencies on third-party load balancers.

Failover time depends on your DNS record TTL, making this solution less suitable for applications requiring zero downtime.

## Supported DNS Providers

Currently, only **INWX** is supported.  
Feel free to create a pull request to add more providers.

## Quick Start

### Prerequisites

- Docker Swarm cluster with at least one manager node
- INWX account with API access
- ID of the DNS record you want to manage

### Deployment

1. Create a Docker secret for the INWX password
```bash
echo "mySecurePassword123" | docker secret create inwx_password -
```
2. Copy the docker-compose.yml file and adjust it to your needs
3. Deploy as a stack to your Swarm cluster:

```bash
docker stack deploy -c docker-compose.yml sentinel
```

### Configuration

| Environment Variable      | Description                        | Default     |
|---------------------------|------------------------------------|-------------|
| `SENTINEL_DOMAIN`         | Domain name                        | example.com |
| `SENTINEL_RECORD`         | Record name (subdomain)            | lb          |
| `SENTINEL_INWX_USER`      | INWX username                      | *required*  |
| `SENTINEL_INWX_PASSWORD`  | INWX password                      | *required*  |
| `SENTINEL_INWX_RECORD_ID` | ID of the DNS record to update     | *required*  |
| `SENTINEL_LOG_LEVEL`      | Logging level (DEBUG, INFO, ERROR) | INFO        |

#### Node labels for public IPs
Instead of setting SENTINEL_SERVER_IP Sentinel can read the public IP address of each node from a Docker Swarm node label.
Run the following command on each node to set the "public_ip" label:

```bash
PUBLIC_IP=$(curl -s https://api.ipify.org) \
NODE_ID=$(docker info --format '{{.Swarm.NodeID}}') ; \
docker node update --label-add public_ip=$PUBLIC_IP $NODE_ID
```
To verify that the label was set correctly, run:
```bash
docker node inspect $NODE_ID --format '{{ index .Spec.Labels "public_ip" }}'
```
## Development

1. Create a `.env` file with your configuration:

```
SENTINEL_DOMAIN=example.com
SENTINEL_RECORD=lb
SENTINEL_SERVER_IP=192.168.1.100
SENTINEL_INWX_USER=your_username
SENTINEL_INWX_PASSWORD=your_password
SENTINEL_INWX_RECORD_ID=12345
SENTINEL_LOG_LEVEL=INFO
```

### Local development

```bash
# Start development environment
make dev

# View logs
make dev-logs

# Start in detached mode
make dev-detach

# Clean up
make clean
```

### Building

```bash
# Build Docker image
make build
```
## Architecture

Sentinel is built with Go and designed to be lightweight and reliable:

- **Zero dependencies**: Built on scratch container
- **Minimal footprint**: Small binary size and memory usage
- **Resilient**: Automatically reconnects if Docker API connection is lost
- **Secure**: No shell or unnecessary components in the container

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgements

- [Docker Engine API](https://docs.docker.com/engine/api/)
- [INWX API](https://www.inwx.com/en/help/apidoc)