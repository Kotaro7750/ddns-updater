# DDNS Updater
![Docker Version](https://img.shields.io/docker/v/7750koutarou/ddns-updater.svg?style=flat-square)

A lightweight Dynamic DNS (DDNS) updater for Cloudflare that automatically updates your DNS records when your public IP address changes.

## Prerequisites

- Cloudflare account with API access
- Domain managed by Cloudflare
- Cloudflare API token with Zone DNS Read/Edit permissions

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `CRON_EXPRESSION` | No | `*/5 * * * *` | Cron schedule for IP checks (every 5 minutes) |
| `ZONE_NAME` | Yes | - | Your domain name (e.g., `example.com`) |
| `RECORD_NAME` | Yes | - | DNS record name to update (e.g., `home`, `ddns`) |
| `CLOUDFLARE_API_TOKEN` | Yes | - | Cloudflare API token with required permissions |

## Usage

1. Build the Docker image:
```bash
docker build -t ddns-updater .
```

2. Run the container:
```bash
docker run -d \
  --name ddns-updater \
  --net host \
  -e ZONE_NAME=example.com \
  -e RECORD_NAME=home \
  -e CLOUDFLARE_API_TOKEN=your_api_token_here \
  -e CRON_EXPRESSION="*/5 * * * *" \
  ddns-updater
```

Note: `--net host` is required to access the host's IPv6 address. Without this option, the container may not have access to IPv6 connectivity, which can cause errors when trying to get the IPv6 address from ipify.

