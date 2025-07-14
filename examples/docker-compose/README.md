# Docker Compose with Caddy Reverse Proxy

## Quick Start

1. **Navigate to the directory:**
   ```bash
   cd examples/docker-compose
   ```

2. **Set up environment:**
   ```bash
   # Edit the .env file
   nano .env
   # Set your actual domain: DOMAIN=twitch-alerts.yourdomain.com
   
   # Edit the config.toml file
   nano config.toml
   # Update twitch credentials and your domain
   ```

3. **Start services:**
   ```bash
   docker-compose up -d
   ```

4. **Verify deployment:**
   ```bash
   # Check service health
   curl https://your-domain.com/health
   
   # Check Caddy logs
   docker-compose logs caddy
   
   # Check itsjustintv logs
   docker-compose logs itsjustintv
   ```

## Directory Structure

```
examples/docker-compose/
├── docker-compose.yml          # Main compose file
├── caddy/
│   └── Caddyfile              # Caddy configuration
├── config.toml               # itsjustintv configuration
├── .env                      # Environment variables
└── README.md                 # This file
```

## Features

- **Automatic HTTPS** with Let's Encrypt
- **Reverse proxy** with Caddy
- **Health checks** for both services
- **Volume persistence** for data and certificates
- **Network isolation** with dedicated bridge network
- **Environment variable** support for configuration

## Commands

```bash
# Start services
docker-compose up -d

# Stop services
docker-compose down

# View logs
docker-compose logs -f

# Restart services
docker-compose restart

# Update to latest image
docker-compose pull
docker-compose up -d

# Access shell inside container
docker-compose exec itsjustintv sh

# View Caddy logs
docker-compose logs -f caddy

# View itsjustintv logs
docker-compose logs -f itsjustintv
```

## Production Deployment

1. **DNS Setup:** Point your domain to your server IP
2. **Firewall:** Ensure ports 80 and 443 are open
3. **Configuration:** Update `.env` and `config.toml` with your domain
4. **SSL:** Caddy will automatically obtain and renew certificates

## Troubleshooting

- **Port conflicts:** Change ports in `docker-compose.yml` if needed
- **Permission issues:** Ensure Docker has permission to bind ports
- **SSL issues:** Check DNS propagation and firewall settings
- **Configuration:** Use `docker-compose logs` to debug issues

## Customization

### Change Ports
Edit `docker-compose.yml` to change the external ports:
```yaml
ports:
  - "8080:80"      # HTTP
  - "8443:443"     # HTTPS
```

### Custom Caddy Configuration
Edit `caddy/Caddyfile` to customize:
- Rate limiting
- Additional headers
- Custom domains
- SSL settings

### Environment Variables
You can override any configuration using environment variables:
```bash
export ITSJUSTINTV_TWITCH_CLIENT_ID="your_client_id"
export ITSJUSTINTV_SERVER_EXTERNAL_DOMAIN="your-domain.com"
```