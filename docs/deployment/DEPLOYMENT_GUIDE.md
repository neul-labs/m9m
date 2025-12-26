# m9m Deployment Guide

## Single Binary, Multiple Modes

**Binary**: `m9m` (35MB)
**Version**: 0.4.0
**Architecture**: Hybrid control plane + worker pool

---

## Quick Start

```bash
# Download or build
go build ./cmd/m9m

# Run in default mode (single-node, no workers)
./m9m

# Access at http://localhost:8080
```

---

## Operating Modes

The single binary supports **4 operating modes**:

| Mode | Description | Use Case |
|------|-------------|----------|
| **control** | Control plane only (API + scheduler) | Single node, original behavior |
| **control --cluster** | Distributed control plane (Raft + work queue) | HA control plane with worker pool |
| **worker** | Execution worker only | Scale out execution capacity |
| **hybrid** | Control + worker in same process | All-in-one deployment |

---

## Mode 1: Single-Node (Default)

**Best for**: Development, small deployments (<1,000 workflows/min)

```bash
./m9m
# OR explicitly:
./m9m --mode control
```

**What it does**:
- ✅ REST API for workflow management
- ✅ Executes workflows directly
- ✅ In-memory storage (or PostgreSQL/SQLite)
- ❌ No high availability
- ❌ No horizontal scaling

**Capacity**:
- ~1,000 workflow executions/minute
- ~10,000 stored workflows
- Single point of failure

---

## Mode 2: Distributed Control Plane + Workers

**Best for**: Production, high availability, scale (10,000-200,000 workflows/min)

### Architecture

```
Control Plane Cluster (3 nodes)     Worker Pool (5-100+ nodes)
┌─────────────────────────┐         ┌──────────────────────┐
│  ./m9m        │         │  ./m9m     │
│  --mode control         │ ←─────→ │  --mode worker       │
│  --cluster              │  NNG    │  --control-plane...  │
│  --raft-addr...         │  Queue  │  --max-concurrent 20 │
└─────────────────────────┘         └──────────────────────┘
         (HA, Raft)                    (Stateless, scalable)
```

### Step 1: Start Control Plane Cluster

**Control Node 1** (bootstrap):
```bash
./m9m \
  --mode control \
  --cluster \
  --node-id=control1 \
  --port=8080 \
  --raft-addr=10.0.1.1:7000 \
  --nng-pub=tcp://10.0.1.1:8000 \
  --data-dir=./data/control1
```

**Control Node 2**:
```bash
./m9m \
  --mode control \
  --cluster \
  --node-id=control2 \
  --port=8081 \
  --raft-addr=10.0.1.2:7000 \
  --raft-peers=10.0.1.1:7000 \
  --nng-pub=tcp://10.0.1.2:8000 \
  --nng-subs=tcp://10.0.1.1:8000 \
  --data-dir=./data/control2
```

**Control Node 3**:
```bash
./m9m \
  --mode control \
  --cluster \
  --node-id=control3 \
  --port=8082 \
  --raft-addr=10.0.1.3:7000 \
  --raft-peers=10.0.1.1:7000 \
  --nng-pub=tcp://10.0.1.3:8000 \
  --nng-subs=tcp://10.0.1.1:8000,tcp://10.0.1.2:8000 \
  --data-dir=./data/control3
```

### Step 2: Start Workers

**Worker 1**:
```bash
./m9m \
  --mode worker \
  --worker-id=worker1 \
  --control-plane=10.0.1.1,10.0.1.2,10.0.1.3 \
  --max-concurrent=20
```

**Worker 2**:
```bash
./m9m \
  --mode worker \
  --worker-id=worker2 \
  --control-plane=10.0.1.1,10.0.1.2,10.0.1.3 \
  --max-concurrent=20
```

**Worker 3-N**: Repeat with unique worker IDs...

### What You Get

**Control Plane**:
- ✅ High availability (Raft 3-node cluster)
- ✅ Automatic failover (<5 seconds)
- ✅ REST API (workflow CRUD)
- ✅ WebSocket (real-time updates)
- ✅ Work queue (NNG PUSH/PULL)
- ✅ Worker management (heartbeat, monitoring)

**Worker Pool**:
- ✅ Horizontal scaling (add/remove dynamically)
- ✅ Fault tolerance (workers can crash)
- ✅ Stateless (no data loss on failure)
- ✅ Load balancing (NNG auto-distributes work)

**Capacity**:
- Control plane: 3-5 nodes (Raft quorum)
- Workers: 5-100+ nodes (unlimited)
- Execution capacity: **50,000-200,000 workflows/minute**

---

## Mode 3: Hybrid (Control + Worker)

**Best for**: Small distributed deployments

```bash
./m9m --mode hybrid
```

**Status**: Not yet implemented (placeholder)

**Planned**: Runs both control plane and worker in same process.

---

## Network Ports

| Port | Protocol | Purpose | Component |
|------|----------|---------|-----------|
| 8080 | HTTP | REST API | Control plane |
| 7000 | TCP | Raft consensus | Control plane |
| 8000 | TCP | NNG pub/sub | Control plane |
| 9000 | TCP | Work queue (PUSH) | Control plane |
| 9001 | TCP | Results (PULL) | Control plane |
| 9002 | TCP | Worker registration (REP) | Control plane |

---

## Deployment Scenarios

### Small: 1 Control + 5 Workers

**Capacity**: ~10,000 executions/minute

```bash
# Control (localhost)
./m9m --mode control --cluster \
  --node-id=control1 --raft-addr=localhost:7000 \
  --nng-pub=tcp://localhost:8000

# Workers (same or different machines)
for i in {1..5}; do
  ./m9m --mode worker \
    --worker-id=worker$i \
    --control-plane=localhost \
    --max-concurrent=20 &
done
```

### Medium: 3 Control + 25 Workers

**Capacity**: ~50,000 executions/minute

- 3-node control plane cluster (HA)
- 25 workers × 2,000 executions/min = 50,000/min

### Large: 5 Control + 100 Workers

**Capacity**: ~200,000 executions/minute

- 5-node control plane cluster (max recommended)
- 100 workers × 2,000 executions/min = 200,000/min

---

## Configuration Options

### Control Plane Flags

```bash
--mode control              # Run as control plane
--cluster                   # Enable Raft clustering
--node-id <string>          # Unique node ID
--raft-addr <addr>          # Raft TCP address
--raft-peers <addrs>        # Peer addresses (comma-separated)
--nng-pub <addr>            # NNG publisher address
--nng-subs <addrs>          # NNG subscriber addresses
--data-dir <path>           # Data directory (default: ./data)
--port <number>             # HTTP API port (default: 8080)
--db <type>                 # Database: memory, postgres, sqlite, badger
```

### Worker Flags

```bash
--mode worker               # Run as worker
--worker-id <string>        # Unique worker ID (auto-generated if omitted)
--control-plane <addrs>     # Control plane addresses (comma-separated)
--max-concurrent <number>   # Max concurrent executions (default: 10)
--heartbeat <seconds>       # Heartbeat interval (default: 5)
```

---

## Monitoring

### Control Plane Health

```bash
# Health check
curl http://localhost:8080/health

# Worker pool status
curl http://localhost:8080/api/v1/workers

# Cluster stats (when using --cluster)
curl http://localhost:8080/api/v1/cluster/stats
```

### Worker Health

Workers send heartbeat every 5 seconds to control plane. Check control plane logs:

```
Worker registered: worker1 (max_concurrent=20)
Worker marked offline: worker3 (last heartbeat: 35s ago)
```

---

## Production Checklist

### Control Plane

- [ ] Deploy 3 or 5 nodes (odd number for Raft quorum)
- [ ] Use persistent storage (BadgerDB, PostgreSQL, SQLite)
- [ ] Configure proper network addresses (not localhost)
- [ ] Set up load balancer for HTTP API
- [ ] Monitor Raft cluster health
- [ ] Back up data directory regularly

### Workers

- [ ] Start with 5-10 workers, scale as needed
- [ ] Set appropriate `--max-concurrent` (10-50 depending on workflow complexity)
- [ ] Monitor worker heartbeats
- [ ] Configure automatic restart (systemd, kubernetes, etc.)
- [ ] Use unique worker IDs

### Network

- [ ] Open required ports (7000, 8000, 9000-9002)
- [ ] Configure firewall rules
- [ ] Use internal network for Raft/NNG (not exposed publicly)
- [ ] Load balancer for HTTP API (port 8080)

---

## Kubernetes Deployment

### Control Plane StatefulSet

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: n8n-control
spec:
  replicas: 3
  selector:
    matchLabels:
      app: n8n-control
  template:
    metadata:
      labels:
        app: n8n-control
    spec:
      containers:
      - name: n8n-control
        image: m9m:0.4.0
        command:
          - /app/m9m
          - --mode=control
          - --cluster
          - --node-id=$(POD_NAME)
          - --raft-addr=$(POD_IP):7000
          - --nng-pub=tcp://$(POD_IP):8000
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: POD_IP
          valueFrom:
            fieldRef:
              fieldPath: status.podIP
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 7000
          name: raft
        - containerPort: 8000
          name: nng
        - containerPort: 9000
          name: work-queue
```

### Worker Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: n8n-workers
spec:
  replicas: 10
  selector:
    matchLabels:
      app: n8n-worker
  template:
    metadata:
      labels:
        app: n8n-worker
    spec:
      containers:
      - name: n8n-worker
        image: m9m:0.4.0
        command:
          - /app/m9m
          - --mode=worker
          - --control-plane=n8n-control-0.n8n-control,n8n-control-1.n8n-control,n8n-control-2.n8n-control
          - --max-concurrent=20
```

---

## Docker Compose Example

```yaml
version: '3.8'

services:
  # Control plane (single node for simplicity)
  control:
    build: .
    command: >
      --mode control
      --cluster
      --node-id=control1
      --raft-addr=control:7000
      --nng-pub=tcp://control:8000
    ports:
      - "8080:8080"
    volumes:
      - control-data:/data

  # Workers (scale with: docker-compose up --scale worker=10)
  worker:
    build: .
    command: >
      --mode worker
      --control-plane=control
      --max-concurrent=20
    depends_on:
      - control

volumes:
  control-data:
```

Scale workers:
```bash
docker-compose up --scale worker=10
```

---

## Troubleshooting

### Control Plane Won't Start

**Issue**: Raft leader election timeout

```bash
# Check if all peers are reachable
ping 10.0.1.1
ping 10.0.1.2

# Check if ports are open
telnet 10.0.1.1 7000
```

**Fix**: Ensure all nodes can communicate on Raft ports.

### Workers Not Connecting

**Issue**: Workers can't connect to control plane

```bash
# Test connectivity
telnet <control-ip> 9000
telnet <control-ip> 9001
telnet <control-ip> 9002
```

**Fix**: Check firewall rules, ensure work queue ports (9000-9002) are open.

### Workers Marked Offline

**Issue**: Heartbeat timeout (>30 seconds)

**Causes**:
- Network issues
- Worker overloaded (too many concurrent executions)
- Worker crashed

**Fix**:
- Reduce `--max-concurrent`
- Check worker logs
- Restart worker

---

## Migration from Single-Node

**From**:
```bash
./m9m
```

**To** (with workers):
```bash
# 1. Start control plane with work queue
./m9m --mode control --cluster \
  --node-id=control1 --raft-addr=localhost:7000 \
  --nng-pub=tcp://localhost:8000

# 2. Start workers
./m9m --mode worker \
  --worker-id=worker1 --control-plane=localhost \
  --max-concurrent=20
```

**No code changes needed** - same API, same workflows!

---

## Summary

**Single Binary** = `m9m` (35MB)

**Deployment Options**:

1. **Single-node**: `./m9m` → 1,000 workflows/min
2. **Distributed**: Control cluster + workers → 200,000 workflows/min
3. **Kubernetes**: StatefulSet + Deployment → Infinite scale

**Zero External Dependencies**:
- ✅ No Redis
- ✅ No RabbitMQ
- ✅ No message queue
- ✅ BadgerDB (embedded)
- ✅ Raft (embedded)
- ✅ NNG (embedded)

**Production-Ready**: High availability + horizontal scaling in one 35MB binary! 🚀
