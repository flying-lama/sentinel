# Sentinel - DNS Failover Manager

Sentinel is a lightweight service that automatically updates DNS records when container orchestration leadership changes. 
It ensures high availability by pointing your domain to the current leader node.

![Sentinel Logo](./images/logo.png)

## Features

- 🔄 Automatic DNS failover for Docker Swarm and Kubernetes clusters
- 🔍 Real-time monitoring of leader changes
- 🌐 DNS record updates
- 🔒 Secure and lightweight (built on scratch container)
- 🚀 Easy to deploy and configure

## How It Works

Sentinel runs on manager/control plane nodes in your cluster. It:

1. Monitors orchestration events for leadership changes
2. Updates the DNS record to point to the leader node's IP when changes occur

## Use Cases

This project is designed for low-budget environments where virtual IPs are unavailable or when you want to avoid dependencies on third-party load balancers.

Failover time depends on your DNS record TTL, making this solution less suitable for applications requiring zero downtime.

## Supported DNS Providers

- INWX
- Bunny DNS

Feel free to create a pull request to add more providers.

## Quick Start

### Prerequisites

- Account for one of the supported DNS providers
- A DNS zone (``SENTINEL_DOMAIN``)
- An A or AAAA record in the zone (``SENTINEL_RECORD``)

**For Docker Swarm:**  
- Docker Swarm cluster with at least one manager node

**For Kubernetes:**  
- Kubernetes cluster with at least one control plane node

### Deployment

#### Docker Swarm Deployment

1. Create a Docker secret for the INWX password
```bash
echo "mySecurePassword123" | docker secret create inwx_password -
```
2. Copy the docker-compose.yml file from ``deployment/docker-swarm`` folder and adjust it to your needs
3. Deploy as a stack to your Swarm cluster:

```bash
docker stack deploy -c docker-compose.yml sentinel
```

#### Kubernetes Deployment
1. Copy and adjust the files from the ``deployment/kubernetes`` folder.
2. Deploy via ``kubectl apply``
3. Create a Kubernetes secret for the **INWX** credentials (optional)
```bash
kubectl create secret generic sentinel-inwx-credentials \
  --namespace sentinel \
  --from-literal=username="myusername" \
  --from-literal=password="mySecurePassword123" \
  --dry-run=client -o yaml | kubectl apply -f -
```
4. Create a Kubernetes secret for the **Bunny DNS** credentials (optional)
```bash
kubectl create secret generic sentinel-bunny-credentials \
  --namespace sentinel \
  --from-literal=api_key="my-api-key" \
  --dry-run=client -o yaml | kubectl apply -f 
```

### Configuration

| Environment Variable     | Description                               | Default                              |
|--------------------------|-------------------------------------------|--------------------------------------|
| `SENTINEL_DOMAIN`        | Domain name                               | example.com                          |
| `SENTINEL_RECORD`        | Record name (subdomain)                   | lb                                   |
| `SENTINEL_LOG_LEVEL`     | Logging level (DEBUG, INFO, ERROR)        | INFO                                 |
| `SENTINEL_ORCHESTRATION` | Orchestration platform (swarm/kubernetes) | swarm                                |
| `SENTINEL_DNS_PROVIDER`  | Name of DNS provider (inwx/bunny)         | inwx                                 |
| `SENTINEL_INWX_USER`     | INWX username                             | *required, if dns provider is inwx*  |
| `SENTINEL_INWX_PASSWORD` | INWX password                             | *required, if dns provider is inwx*  |
| `SENTINEL_BUNNY_API_KEY` | Bunny API key                             | *required, if dns provider is bunny* |

#### Public IP configuration

**Docker Swarm**  
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

**Kubernetes**  
Without setting a label the first external IP address of the node is used.
If you want to set it to something else you can run the following command on each node to set the "public_ip" label (replace ``mynode`` with your node name)
```bash
PUBLIC_IP=$(curl -s https://api.ipify.org) \
kubectl label nodes mynode public_ip=$PUBLIC_IP
```

## Development

```bash
# Copy and adjust to fit your setup
cp .env.dist .env

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
- [Kubernetes](https://kubernetes.io/)
- [libdns](https://github.com/libdns/libdns)
- [INWX API](https://www.inwx.com/en/help/apidoc)
- [Bunny API](https://docs.bunny.net/reference/bunnynet-api-overview)