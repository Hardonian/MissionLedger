# Operational Runbook

## Service: missionledger

### Deployment
```bash
# Build
make build 2>/dev/null || cargo build --release 2>/dev/null || go build ./... 2>/dev/null || npm run build

# Run locally
docker compose up -d

# Deploy to prod
# (Add your deployment commands here)
```

### Health Checks
- `GET /health` - liveness probe
- `GET /ready` - readiness probe (checks DB/dependencies)

### Common Issues

| Symptom | Diagnosis | Resolution |
|---------|-----------|------------|
| 500 errors | Check traces/logs | Look for span errors |
| High latency | Check metrics | Identify slow queries/ops |
| OOM kills | Check memory metrics | Tune build flags |
| Connection pool exhausted | Check pool metrics | Increase pool size |

### Rollback
```bash
# Docker
docker tag missionledger:previous missionledger:latest
docker compose up -d --force-recreate

# Kubernetes
kubectl rollout undo deployment/missionledger
```

### Logs
```bash
# Structured JSON logs
docker logs -f missionledger | jq .

# Filter by trace_id
docker logs missionledger | grep "trace_id=\"abc123\""
```

### Database
```bash
# Run migrations
# (Add your migration commands here)
```

### Performance Tuning
- Binary size: `./scripts/binary_size_audit.sh`
- Load test: `./scripts/load_test.sh`
- Chaos test: `./scripts/chaos_test.sh`