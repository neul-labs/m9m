# Kubernetes Deployment

Deploy m9m on Kubernetes for production-grade scalability.

## Prerequisites

- Kubernetes cluster (1.24+)
- kubectl configured
- Helm 3.x (optional)

## Quick Start

### Basic Deployment

```yaml
# m9m-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: m9m
  labels:
    app: m9m
spec:
  replicas: 3
  selector:
    matchLabels:
      app: m9m
  template:
    metadata:
      labels:
        app: m9m
    spec:
      containers:
      - name: m9m
        image: m9m/m9m:latest
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9090
          name: metrics
        env:
        - name: M9M_DB_URL
          valueFrom:
            secretKeyRef:
              name: m9m-secrets
              key: database-url
        - name: M9M_QUEUE_URL
          valueFrom:
            secretKeyRef:
              name: m9m-secrets
              key: redis-url
        resources:
          limits:
            cpu: "1"
            memory: 1Gi
          requests:
            cpu: 250m
            memory: 256Mi
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 15
          periodSeconds: 20
```

### Service

```yaml
# m9m-service.yaml
apiVersion: v1
kind: Service
metadata:
  name: m9m
  labels:
    app: m9m
spec:
  type: ClusterIP
  ports:
  - port: 8080
    targetPort: 8080
    name: http
  - port: 9090
    targetPort: 9090
    name: metrics
  selector:
    app: m9m
```

### Ingress

```yaml
# m9m-ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: m9m
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
  - hosts:
    - m9m.example.com
    secretName: m9m-tls
  rules:
  - host: m9m.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: m9m
            port:
              number: 8080
```

## Complete Setup

### Namespace

```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: m9m
```

### Secrets

```yaml
# secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: m9m-secrets
  namespace: m9m
type: Opaque
stringData:
  database-url: "postgres://m9m:password@postgres:5432/m9m"
  redis-url: "redis://:password@redis:6379"
  encryption-key: "your-32-character-encryption-key"
```

### ConfigMap

```yaml
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: m9m-config
  namespace: m9m
data:
  config.yaml: |
    server:
      port: 8080
    queue:
      maxWorkers: 10
    monitoring:
      metricsPort: 9090
```

### PostgreSQL

```yaml
# postgres.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
  namespace: m9m
spec:
  serviceName: postgres
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:15-alpine
        ports:
        - containerPort: 5432
        env:
        - name: POSTGRES_USER
          value: m9m
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: m9m-secrets
              key: db-password
        - name: POSTGRES_DB
          value: m9m
        volumeMounts:
        - name: data
          mountPath: /var/lib/postgresql/data
        resources:
          limits:
            cpu: "1"
            memory: 1Gi
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 10Gi
---
apiVersion: v1
kind: Service
metadata:
  name: postgres
  namespace: m9m
spec:
  ports:
  - port: 5432
  selector:
    app: postgres
```

### Redis

```yaml
# redis.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
  namespace: m9m
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
      - name: redis
        image: redis:7-alpine
        ports:
        - containerPort: 6379
        command:
        - redis-server
        - --requirepass
        - $(REDIS_PASSWORD)
        - --appendonly
        - "yes"
        env:
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: m9m-secrets
              key: redis-password
        resources:
          limits:
            cpu: 500m
            memory: 512Mi
---
apiVersion: v1
kind: Service
metadata:
  name: redis
  namespace: m9m
spec:
  ports:
  - port: 6379
  selector:
    app: redis
```

### Full m9m Deployment

```yaml
# m9m-full.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: m9m
  namespace: m9m
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 1
  selector:
    matchLabels:
      app: m9m
  template:
    metadata:
      labels:
        app: m9m
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9090"
    spec:
      serviceAccountName: m9m
      containers:
      - name: m9m
        image: m9m/m9m:latest
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9090
          name: metrics
        env:
        - name: M9M_DB_TYPE
          value: postgres
        - name: M9M_DB_URL
          valueFrom:
            secretKeyRef:
              name: m9m-secrets
              key: database-url
        - name: M9M_QUEUE_TYPE
          value: redis
        - name: M9M_QUEUE_URL
          valueFrom:
            secretKeyRef:
              name: m9m-secrets
              key: redis-url
        - name: M9M_ENCRYPTION_KEY
          valueFrom:
            secretKeyRef:
              name: m9m-secrets
              key: encryption-key
        volumeMounts:
        - name: config
          mountPath: /app/config
        resources:
          limits:
            cpu: "2"
            memory: 2Gi
          requests:
            cpu: 500m
            memory: 512Mi
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
          failureThreshold: 3
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 15
          periodSeconds: 20
          failureThreshold: 3
      volumes:
      - name: config
        configMap:
          name: m9m-config
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchLabels:
                  app: m9m
              topologyKey: kubernetes.io/hostname
```

## Helm Chart

### Install

```bash
helm repo add m9m https://charts.m9m.io
helm install m9m m9m/m9m \
  --namespace m9m \
  --create-namespace \
  --set postgresql.enabled=true \
  --set redis.enabled=true
```

### values.yaml

```yaml
# values.yaml
replicaCount: 3

image:
  repository: m9m/m9m
  tag: latest
  pullPolicy: IfNotPresent

service:
  type: ClusterIP
  port: 8080

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: m9m.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: m9m-tls
      hosts:
        - m9m.example.com

resources:
  limits:
    cpu: 2
    memory: 2Gi
  requests:
    cpu: 500m
    memory: 512Mi

postgresql:
  enabled: true
  auth:
    database: m9m
    username: m9m
    password: password
  primary:
    persistence:
      size: 10Gi

redis:
  enabled: true
  auth:
    password: password

config:
  queue:
    maxWorkers: 10
  monitoring:
    enabled: true
```

## Horizontal Pod Autoscaler

```yaml
# hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: m9m
  namespace: m9m
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: m9m
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

## PodDisruptionBudget

```yaml
# pdb.yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: m9m
  namespace: m9m
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: m9m
```

## Network Policy

```yaml
# network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: m9m
  namespace: m9m
spec:
  podSelector:
    matchLabels:
      app: m9m
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - port: 8080
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: postgres
    ports:
    - port: 5432
  - to:
    - podSelector:
        matchLabels:
          app: redis
    ports:
    - port: 6379
```

## ServiceMonitor (Prometheus)

```yaml
# servicemonitor.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: m9m
  namespace: m9m
spec:
  selector:
    matchLabels:
      app: m9m
  endpoints:
  - port: metrics
    interval: 30s
```

## Deploy

```bash
# Apply all manifests
kubectl apply -f namespace.yaml
kubectl apply -f secrets.yaml
kubectl apply -f configmap.yaml
kubectl apply -f postgres.yaml
kubectl apply -f redis.yaml
kubectl apply -f m9m-full.yaml
kubectl apply -f m9m-service.yaml
kubectl apply -f m9m-ingress.yaml
kubectl apply -f hpa.yaml
kubectl apply -f pdb.yaml
```

## Verify

```bash
# Check pods
kubectl get pods -n m9m

# Check services
kubectl get svc -n m9m

# Check ingress
kubectl get ingress -n m9m

# View logs
kubectl logs -f -l app=m9m -n m9m
```

## Next Steps

- [Production](production.md) - Production best practices
- [Scaling](scaling.md) - Advanced scaling strategies
- [Configuration](../reference/configuration.md) - Configuration options
