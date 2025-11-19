# Kubernetes 部署指南

本指南介绍如何在 Kubernetes 集群上部署 TokenginX。

## 前置要求

- Kubernetes 1.20+
- kubectl 命令行工具
- Helm 3.0+ (可选)
- 持久化存储类（StorageClass）
- 至少 3 个工作节点（生产环境）

## 快速开始

### 一键部署

```bash
# 克隆仓库
git clone https://github.com/your-org/tokenginx.git
cd tokenginx/deploy/kubernetes

# 创建 namespace 和资源
kubectl apply -f namespace.yaml
kubectl apply -f rbac.yaml
kubectl apply -f configmap.yaml
kubectl create secret generic tokenginx-secret \
  --from-literal=master_key=your-secret-key-here \
  -n tokenginx
kubectl apply -f pvc.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml

# 检查部署状态
kubectl get pods -n tokenginx
kubectl get svc -n tokenginx
```

### 使用 kustomize

```bash
# 部署
kubectl apply -k deploy/kubernetes/

# 查看状态
kubectl get all -n tokenginx
```

## 配置说明

### Namespace

```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: tokenginx
```

### ConfigMap

```yaml
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: tokenginx-config
  namespace: tokenginx
data:
  config.yaml: |
    server:
      tcp_addr: "0.0.0.0:6380"
      grpc_addr: "0.0.0.0:9090"
      http_addr: "0.0.0.0:8080"
    storage:
      enable_persistence: true
    # ... 更多配置
```

### Secret

```bash
# 创建 Secret
kubectl create secret generic tokenginx-secret \
  --from-literal=master_key=$(openssl rand -hex 32) \
  -n tokenginx

# 从文件创建
kubectl create secret generic tokenginx-secret \
  --from-file=master_key=./master.key \
  -n tokenginx

# TLS 证书
kubectl create secret tls tokenginx-tls \
  --cert=server.crt \
  --key=server.key \
  -n tokenginx
```

### PersistentVolumeClaim

```yaml
# pvc.yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: tokenginx-data
  namespace: tokenginx
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
  storageClassName: fast-ssd
```

### Deployment

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tokenginx
  namespace: tokenginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: tokenginx
  template:
    metadata:
      labels:
        app: tokenginx
    spec:
      containers:
      - name: tokenginx
        image: tokenginx/tokenginx:latest
        ports:
        - containerPort: 6380
        - containerPort: 9090
        - containerPort: 8080
        resources:
          requests:
            cpu: 500m
            memory: 512Mi
          limits:
            cpu: 2000m
            memory: 2Gi
        livenessProbe:
          httpGet:
            path: /health
            port: 8081
        readinessProbe:
          httpGet:
            path: /health
            port: 8081
```

### Service

```yaml
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: tokenginx
  namespace: tokenginx
spec:
  type: ClusterIP
  ports:
  - name: tcp-resp
    port: 6380
    targetPort: 6380
  - name: grpc
    port: 9090
    targetPort: 9090
  - name: http
    port: 8080
    targetPort: 8080
  selector:
    app: tokenginx
```

## 高可用配置

### 多副本部署

```yaml
spec:
  replicas: 3  # 最少 3 个副本
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0  # 确保无停机
```

### Pod 反亲和性

```yaml
spec:
  template:
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchExpressions:
              - key: app
                operator: In
                values:
                - tokenginx
            topologyKey: kubernetes.io/hostname
```

### HPA 自动扩缩容

```yaml
# hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: tokenginx-hpa
  namespace: tokenginx
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: tokenginx
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
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 50
        periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 0
      policies:
      - type: Percent
        value: 100
        periodSeconds: 15
```

## Ingress 配置

### NGINX Ingress

```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: tokenginx-ingress
  namespace: tokenginx
  annotations:
    kubernetes.io/ingress.class: nginx
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/backend-protocol: "HTTP"
spec:
  tls:
  - hosts:
    - tokenginx.example.com
    secretName: tokenginx-tls-secret
  rules:
  - host: tokenginx.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: tokenginx
            port:
              number: 8080
```

### Traefik Ingress

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: tokenginx-ingress
  namespace: tokenginx
  annotations:
    kubernetes.io/ingress.class: traefik
    traefik.ingress.kubernetes.io/router.tls: "true"
spec:
  rules:
  - host: tokenginx.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: tokenginx
            port:
              number: 8080
```

## 监控集成

### ServiceMonitor (Prometheus Operator)

```yaml
# servicemonitor.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: tokenginx
  namespace: tokenginx
  labels:
    app: tokenginx
spec:
  selector:
    matchLabels:
      app: tokenginx
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
```

### PodMonitor

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PodMonitor
metadata:
  name: tokenginx
  namespace: tokenginx
spec:
  selector:
    matchLabels:
      app: tokenginx
  podMetricsEndpoints:
  - port: metrics
    interval: 30s
```

## Helm Chart 部署

### 安装 Chart

```bash
# 添加 Helm 仓库
helm repo add tokenginx https://charts.tokenginx.io
helm repo update

# 安装
helm install tokenginx tokenginx/tokenginx \
  --namespace tokenginx \
  --create-namespace \
  --set replicaCount=3 \
  --set persistence.enabled=true \
  --set persistence.size=10Gi

# 自定义配置
helm install tokenginx tokenginx/tokenginx \
  -f values.yaml \
  --namespace tokenginx
```

### values.yaml 示例

```yaml
# values.yaml
replicaCount: 3

image:
  repository: tokenginx/tokenginx
  tag: "latest"
  pullPolicy: IfNotPresent

service:
  type: ClusterIP
  tcpPort: 6380
  grpcPort: 9090
  httpPort: 8080

persistence:
  enabled: true
  storageClass: fast-ssd
  size: 10Gi

resources:
  requests:
    cpu: 500m
    memory: 512Mi
  limits:
    cpu: 2000m
    memory: 2Gi

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: tokenginx.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: tokenginx-tls
      hosts:
        - tokenginx.example.com

monitoring:
  enabled: true
  serviceMonitor:
    enabled: true
```

## 安全配置

### NetworkPolicy

```yaml
# networkpolicy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: tokenginx-netpol
  namespace: tokenginx
spec:
  podSelector:
    matchLabels:
      app: tokenginx
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: default
    ports:
    - protocol: TCP
      port: 6380
    - protocol: TCP
      port: 9090
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 53  # DNS
    - protocol: UDP
      port: 53
```

### PodSecurityPolicy

```yaml
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: tokenginx-psp
spec:
  privileged: false
  allowPrivilegeEscalation: false
  requiredDropCapabilities:
  - ALL
  volumes:
  - 'configMap'
  - 'emptyDir'
  - 'projected'
  - 'secret'
  - 'downwardAPI'
  - 'persistentVolumeClaim'
  runAsUser:
    rule: 'MustRunAsNonRoot'
  seLinux:
    rule: 'RunAsAny'
  fsGroup:
    rule: 'RunAsAny'
  readOnlyRootFilesystem: true
```

## 备份与恢复

### 使用 Velero

```bash
# 安装 Velero
velero install \
  --provider aws \
  --bucket tokenginx-backups \
  --secret-file ./credentials-velero

# 备份 namespace
velero backup create tokenginx-backup \
  --include-namespaces tokenginx

# 恢复
velero restore create --from-backup tokenginx-backup
```

### 手动备份

```bash
# 导出配置
kubectl get all,cm,secret,pvc -n tokenginx -o yaml > tokenginx-backup.yaml

# 备份 PV 数据
kubectl exec -n tokenginx tokenginx-0 -- \
  tar czf - /var/lib/tokenginx > tokenginx-data.tar.gz
```

## 滚动更新

```bash
# 更新镜像
kubectl set image deployment/tokenginx \
  tokenginx=tokenginx/tokenginx:v0.2.0 \
  -n tokenginx

# 查看更新状态
kubectl rollout status deployment/tokenginx -n tokenginx

# 回滚
kubectl rollout undo deployment/tokenginx -n tokenginx

# 查看历史
kubectl rollout history deployment/tokenginx -n tokenginx
```

## 故障排查

```bash
# 查看 Pod 状态
kubectl get pods -n tokenginx

# 查看 Pod 日志
kubectl logs -f deployment/tokenginx -n tokenginx

# 查看事件
kubectl get events -n tokenginx --sort-by='.lastTimestamp'

# 进入 Pod
kubectl exec -it deployment/tokenginx -n tokenginx -- /bin/sh

# 查看资源使用
kubectl top pods -n tokenginx
kubectl top nodes

# 调试
kubectl describe pod <pod-name> -n tokenginx
```

## 完整部署清单

```bash
# deploy/kubernetes/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: tokenginx

resources:
- namespace.yaml
- rbac.yaml
- configmap.yaml
- pvc.yaml
- deployment.yaml
- service.yaml
- ingress.yaml
- hpa.yaml
- servicemonitor.yaml

secretGenerator:
- name: tokenginx-secret
  literals:
  - master_key=REPLACE_WITH_YOUR_KEY

images:
- name: tokenginx/tokenginx
  newTag: latest
```

部署命令:
```bash
kubectl apply -k deploy/kubernetes/
```

## 下一步

- 查看 [Docker 部署指南](./docker.md) 了解容器化部署
- 查看 [Podman 部署指南](./podman.md) 了解 rootless 容器
- 查看 [监控配置](../production/monitoring.md) 了解完整监控方案
