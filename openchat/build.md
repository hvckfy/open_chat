# OpenChat Docker Build Guide

Build multi-arch Docker images from the `openchat/openchat` repository.

## Requirements

- Docker installed
- Docker Buildx enabled
- Logged in to Docker registry

```bash
docker login
Initialize buildx (once):
docker buildx create --use
docker buildx inspect --bootstrap
```
Build Images

Account Service
```bash
docker buildx build \
  --platform linux/amd64 \
  -f cmd/account-service/Dockerfile \
  -t hvckfy/openchat-accountservice:latest \
  --push .
```
Message Service
```bash
docker buildx build \
  --platform linux/amd64\
  -f cmd/message-service/Dockerfile \
  -t hvckfy/openchat-messageservice:latest \
  --push .
  ```
