# m9m Hybrid Architecture Design

## Overview

This document describes the hybrid architecture that separates the **control plane** (workflow management) from the **execution plane** (workflow execution) using **NNG** for queue-based distribution.

**Goal**: Scale to **100,000+ workflow executions per minute** with zero external dependencies.

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                     CONTROL PLANE                            │
│                  (Raft Cluster - 3-5 nodes)                  │
│                                                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                  │
│  │ Control  │  │ Control  │  │ Control  │                  │
│  │  Node 1  │  │  Node 2  │  │  Node 3  │                  │
│  │ (Leader) │  │(Follower)│  │(Follower)│                  │
│  └────┬─────┘  └──────────┘  └──────────┘                  │
│       │                                                      │
│       │ Raft Replication (BadgerDB)                         │
│       │ REST API (workflow CRUD)                            │
│       │ WebSocket (real-time updates)                       │
│       │                                                      │
└───────┼──────────────────────────────────────────────────────┘
        │
        │ NNG PUSH (Work Queue)
        │ NNG PULL (Results)
        │ NNG REP (Worker Registration)
        │
        ▼
┌─────────────────────────────────────────────────────────────┐
│                    EXECUTION PLANE                           │
│                  (Worker Pool - 10-100+ nodes)               │
│                                                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │ Worker 1 │  │ Worker 2 │  │ Worker 3 │  │ Worker N │   │
│  │          │  │          │  │          │  │   ...    │   │
│  │ Stateless│  │ Stateless│  │ Stateless│  │ Stateless│   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
│                                                              │
│  - Pull work from NNG queue                                 │
│  - Execute workflows                                         │
│  - Push results back                                         │
│  - Heartbeat to control plane                               │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## Component Breakdown

### Control Plane (Raft Cluster)

**Responsibilities**:
1. ✅ Store workflow definitions (BadgerDB + Raft)
2. ✅ Provide REST API for workflow CRUD
3. ✅ Distribute work to workers (NNG PUSH)
4. ✅ Collect results from workers (NNG PULL)
5. ✅ Store execution history
6. ✅ Manage worker pool (registration, heartbeat)
7. ✅ Provide WebSocket updates to clients

**Does NOT**:
- ❌ Execute workflows (delegated to workers)
- ❌ Store large execution data (only metadata)

**Scalability**: 3-5 nodes (Raft quorum)

---

### Execution Plane (Worker Pool)

**Responsibilities**:
1. ✅ Register with control plane
2. ✅ Pull work from NNG queue (PULL socket)
3. ✅ Execute workflows independently
4. ✅ Push results back (PUSH socket)
5. ✅ Send heartbeat every 5 seconds
6. ✅ Handle failures gracefully

**Characteristics**:
- **Stateless**: No persistent storage
- **Independent**: Can crash without affecting others
- **Scalable**: Add/remove dynamically
- **Specialized**: Can have different worker types (CPU-heavy, I/O-heavy)

**Scalability**: 10-100+ nodes (horizontal scaling)

---

## NNG Communication Patterns

### Pattern 1: Work Distribution (PUSH/PULL)

**Control Plane PUSH → Workers PULL**

```
Control Plane:
- Binds PUSH socket: tcp://*:9000
- Pushes work messages to queue

Workers:
- Connect PULL socket: tcp://control-plane:9000
- Pull work messages from queue
- NNG load-balances across all workers
```

**Message Format**:
```json
{
  "type": "execute_workflow",
  "execution_id": "exec_123",
  "workflow_id": "wf_456",
  "workflow": { /* full workflow definition */ },
  "input_data": [ /* initial data */ ],
  "priority": "normal"
}
```

---

### Pattern 2: Result Collection (PUSH/PULL)

**Workers PUSH → Control Plane PULL**

```
Control Plane:
- Binds PULL socket: tcp://*:9001
- Pulls result messages from workers

Workers:
- Connect PUSH socket: tcp://control-plane:9001
- Push result messages back
```

**Message Format**:
```json
{
  "type": "execution_result",
  "execution_id": "exec_123",
  "status": "success",
  "result_data": [ /* output data */ ],
  "error": null,
  "duration_ms": 1234,
  "worker_id": "worker-1"
}
```

---

### Pattern 3: Worker Registration (REQ/REP)

**Workers REQ → Control Plane REP**

```
Control Plane:
- Binds REP socket: tcp://*:9002
- Replies to registration/heartbeat

Workers:
- Connect REQ socket: tcp://control-plane:9002
- Send registration on startup
- Send heartbeat every 5 seconds
```

**Registration Message**:
```json
{
  "type": "register",
  "worker_id": "worker-1",
  "capabilities": ["http", "database", "transform"],
  "max_concurrent": 10
}
```

**Heartbeat Message**:
```json
{
  "type": "heartbeat",
  "worker_id": "worker-1",
  "active_executions": 3,
  "uptime_seconds": 12345
}
```

---

## Workflow Execution Flow

### Step-by-Step

1. **User triggers workflow** (API call to control plane)
   ```
   POST /api/v1/workflows/{id}/execute
   ```

2. **Control plane creates execution record**
   - Generates execution ID
   - Stores in BadgerDB (replicated via Raft)
   - Status: "queued"

3. **Control plane pushes to work queue**
   - Serializes workflow + input data
   - Pushes to NNG PUSH socket
   - NNG distributes to available worker

4. **Worker pulls work**
   - Receives message via PULL socket
   - Updates local status: "running"
   - Begins execution

5. **Worker executes workflow**
   - Runs all nodes sequentially
   - Collects output data
   - Handles errors

6. **Worker pushes result**
   - Sends result via PUSH socket
   - Includes execution ID, status, output

7. **Control plane receives result**
   - Updates execution record in BadgerDB
   - Broadcasts WebSocket update to clients
   - Stores final result

8. **Client receives update**
   - WebSocket message with execution status
   - Can fetch full results via API

---

## Scalability Improvements

### Before (Raft Cluster Only)

| Metric | Capacity |
|--------|----------|
| Workflow Executions | ~1,000/minute |
| Bottleneck | Single leader CPU |
| Max Workers | 1 (leader only) |

### After (Hybrid Architecture)

| Metric | Capacity |
|--------|----------|
| Workflow Executions | **100,000+/minute** |
| Bottleneck | NNG queue throughput |
| Max Workers | **100+ independent workers** |

**Example**:
- 50 workers × 2,000 executions/min each = **100,000 executions/min**
- 100 workers × 2,000 executions/min each = **200,000 executions/min**

---

## Worker Specialization (Optional)

Workers can be specialized for different workload types:

### Worker Types

**CPU-Heavy Workers**:
```bash
./m9m-worker --type=cpu --max-concurrent=5
```
- Execute compute-intensive workflows
- Lower concurrency, higher CPU allocation

**I/O-Heavy Workers**:
```bash
./m9m-worker --type=io --max-concurrent=50
```
- Execute HTTP requests, database queries
- Higher concurrency, lower CPU needs

**Mixed Workers** (default):
```bash
./m9m-worker --max-concurrent=20
```
- Handle all workflow types

---

## Fault Tolerance

### Control Plane Failures

**Scenario**: Leader node crashes
- **Action**: Raft elects new leader (<5 seconds)
- **Impact**: Brief pause in work distribution
- **Recovery**: New leader resumes pushing work

**Scenario**: All control nodes crash
- **Action**: Workers continue processing current work
- **Impact**: No new work distributed, results buffered
- **Recovery**: Workers reconnect when control plane restarts

### Worker Failures

**Scenario**: Worker crashes during execution
- **Action**: Work message not acknowledged
- **Impact**: Execution marked as "timeout" after 5 minutes
- **Recovery**: Can retry execution manually or automatically

**Scenario**: Worker becomes unresponsive
- **Action**: Heartbeat timeout (30 seconds)
- **Impact**: Worker marked as "offline", removed from pool
- **Recovery**: Worker re-registers on restart

---

## Monitoring & Observability

### Control Plane Metrics

```bash
GET /api/v1/cluster/metrics
```

```json
{
  "control_plane": {
    "nodes": 3,
    "leader": "node1",
    "raft_state": "healthy"
  },
  "worker_pool": {
    "total_workers": 25,
    "active_workers": 24,
    "offline_workers": 1
  },
  "work_queue": {
    "queued": 142,
    "in_progress": 87,
    "completed_last_minute": 1834
  },
  "executions": {
    "success_rate": 99.2,
    "avg_duration_ms": 234,
    "total_today": 125000
  }
}
```

### Worker Metrics

```bash
GET /api/v1/workers/{id}/metrics
```

```json
{
  "worker_id": "worker-1",
  "status": "active",
  "uptime_seconds": 86400,
  "executions": {
    "total": 4523,
    "success": 4489,
    "failed": 34
  },
  "current_load": {
    "active_executions": 8,
    "max_concurrent": 20,
    "cpu_percent": 45.2,
    "memory_mb": 512
  }
}
```

---

## Deployment Scenarios

### Small Deployment (3 control + 5 workers)

**Capacity**: ~10,000 executions/minute

```bash
# Control Plane
./m9m --cluster --node-id=control1 ...
./m9m --cluster --node-id=control2 ...
./m9m --cluster --node-id=control3 ...

# Workers
./m9m-worker --id=worker1 --control=control1:9000,control2:9000,control3:9000
./m9m-worker --id=worker2 ...
./m9m-worker --id=worker3 ...
./m9m-worker --id=worker4 ...
./m9m-worker --id=worker5 ...
```

### Medium Deployment (5 control + 25 workers)

**Capacity**: ~50,000 executions/minute

### Large Deployment (5 control + 100 workers)

**Capacity**: ~200,000 executions/minute

---

## Implementation Files

### New Components

1. **`internal/queue/nng_queue.go`**
   - Work distribution (PUSH/PULL)
   - Result collection (PUSH/PULL)
   - Worker registration (REQ/REP)

2. **`internal/worker/executor.go`**
   - Worker execution engine
   - Work polling and processing
   - Result reporting

3. **`internal/worker/registration.go`**
   - Worker registration logic
   - Heartbeat mechanism
   - Health reporting

4. **`internal/control/worker_pool.go`**
   - Worker pool management
   - Worker health monitoring
   - Load balancing

5. **`cmd/m9m-worker/main.go`**
   - Worker binary entry point
   - Configuration and initialization

---

## Benefits of This Architecture

✅ **Massive Scalability**
- 100+ workers can execute 200,000+ workflows/minute
- Horizontal scaling of execution plane

✅ **Zero External Dependencies**
- NNG replaces RabbitMQ/Redis
- Single binary for control plane
- Single binary for workers

✅ **Fault Tolerance**
- Control plane: Raft HA
- Workers: Stateless, can crash/restart

✅ **Resource Efficiency**
- Control plane stays lightweight
- Workers use full CPU/memory for execution

✅ **Flexibility**
- Add/remove workers dynamically
- Specialize workers for workload types
- Scale control and execution independently

---

## Next Steps

1. ✅ Implement NNG-based work queue
2. ✅ Implement worker executor
3. ✅ Implement worker registration system
4. ✅ Update control plane for work distribution
5. ✅ Create worker binary
6. ✅ Test hybrid architecture
7. ✅ Performance benchmarking

---

## Summary

This hybrid architecture combines the best of both worlds:

- **Control Plane** (Raft cluster): High availability, strong consistency, workflow management
- **Execution Plane** (Worker pool): Horizontal scalability, fault isolation, specialization

**Result**: A production-ready system that can scale to **100,000+ workflow executions per minute** with **zero external dependencies**.
