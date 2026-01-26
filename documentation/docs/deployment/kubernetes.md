# Kubernetes Deployment

Deploy m9m on Kubernetes for production workloads.

## Prerequisites

- Kubernetes cluster (1.20+)
- kubectl configured
- Helm 3 (optional)

## Quick Start

### Minimal Deployment

```yaml
# m9m-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: m9m
spec:
  replicas: 1
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
        image: neullabs/m9m:latest
        ports:
        - containerPort: 8080
        env:
        - name: M9M_LOG_LEVEL
          value: "info"
---
apiVersion: v1
kind: Service
metadata:
  name: m9m
spec:
  selector:
    app: m9m
  ports:
  - port: 80
    targetPort: 8080
```

Deploy:

```bash
kubectl apply -f m9m-deployment.yaml
```

## Production Deployment

### Namespace

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: m9m
```

### ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: m9m-config
  namespace: m9m
data:
  config.yaml: |
    server:
      port: 8080

    database:
      type: postgres

    queue:
      type: redis
      workers: 5

    monitoring:
      enabled: true
      metricsPort: 9090
```

### Secrets

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: m9m-secrets
  namespace: m9m
type: Opaque
stringData:
  database-url: "postgres://user:password@postgres:5432/m9m"
  jwt-secret: "your-secure-jwt-secret"
  encryption-key: "your-32-byte-encryption-key"
```

### Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: m9m
  namespace: m9m
spec:
  replicas: 3
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
        image: neullabs/m9m:latest
        ports:
        - name: http
          containerPort: 8080
        - name: metrics
          containerPort: 9090
        env:
        - name: M9M_DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: m9m-secrets
              key: database-url
        - name: M9M_JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: m9m-secrets
              key: jwt-secret
        - name: M9M_QUEUE_TYPE
          value: "redis"
        - name: M9M_QUEUE_URL
          value: "redis://redis:6379"
        volumeMounts:
        - name: config
          mountPath: /etc/m9m
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /health
            port: http
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /ready
            port: http
          initialDelaySeconds: 5
          periodSeconds: 10
      volumes:
      - name: config
        configMap:
          name: m9m-config
```

### Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: m9m
  namespace: m9m
spec:
  selector:
    app: m9m
  ports:
  - name: http
    port: 80
    targetPort: 8080
  - name: metrics
    port: 9090
    targetPort: 9090
```

### Ingress

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: m9m
  namespace: m9m
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
              number: 80
```

### HorizontalPodAutoscaler

```yaml
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
  minReplicas: 2
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

### PodDisruptionBudget

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: m9m
  namespace: m9m
spec:
  minAvailable: 1
  selector:
    matchLabels:
      app: m9m
```

## PostgreSQL StatefulSet

```yaml
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
          value: "m9m"
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgres-secrets
              key: password
        - name: POSTGRES_DB
          value: "m9m"
        volumeMounts:
        - name: data
          mountPath: /var/lib/postgresql/data
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
  selector:
    app: postgres
  ports:
  - port: 5432
```

## Redis Deployment

```yaml
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
        command: ["redis-server", "--appendonly", "yes"]
        volumeMounts:
        - name: data
          mountPath: /data
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: redis-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: redis
  namespace: m9m
spec:
  selector:
    app: redis
  ports:
  - port: 6379
```

## Helm Chart

### Install from Repository

```bash
helm repo add m9m https://charts.neullabs.com
helm install m9m m9m/m9m -n m9m --create-namespace
```

### Custom Values

```yaml
# values.yaml
replicaCount: 3

image:
  repository: neullabs/m9m
  tag: latest

resources:
  requests:
    memory: 256Mi
    cpu: 250m
  limits:
    memory: 512Mi
    cpu: 1000m

database:
  type: postgres
  external: false

postgresql:
  enabled: true
  auth:
    password: "secretpassword"

redis:
  enabled: true

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: m9m.example.com
      paths:
        - path: /
          pathType: Prefix

autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
```

Install:

```bash
helm install m9m m9m/m9m -f values.yaml -n m9m
```

## Service Account & RBAC

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: m9m
  namespace: m9m
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: m9m
  namespace: m9m
rules:
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: m9m
  namespace: m9m
subjects:
- kind: ServiceAccount
  name: m9m
roleRef:
  kind: Role
  name: m9m
  apiGroup: rbac.authorization.k8s.io
```

## Monitoring

### ServiceMonitor (Prometheus Operator)

```yaml
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

### Grafana Dashboard

Import dashboard ID or JSON from:

```
https://grafana.com/grafana/dashboards/m9m
```

## Troubleshooting

### Check Pod Status

```bash
kubectl get pods -n m9m
kubectl describe pod m9m-xxx -n m9m
```

### View Logs

```bash
kubectl logs -f deployment/m9m -n m9m
kubectl logs -f deployment/m9m -n m9m --all-containers
```

### Shell Access

```bash
kubectl exec -it deployment/m9m -n m9m -- /bin/sh
```

### Check Events

```bash
kubectl get events -n m9m --sort-by=.metadata.creationTimestamp
```

## Upgrading

```bash
# Update image
kubectl set image deployment/m9m m9m=neullabs/m9m:1.1.0 -n m9m

# Or with Helm
helm upgrade m9m m9m/m9m --set image.tag=1.1.0 -n m9m
```

## Rollback

```bash
# Kubernetes
kubectl rollout undo deployment/m9m -n m9m

# Helm
helm rollback m9m 1 -n m9m
```
