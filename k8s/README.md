# Kubernetes deployment

This folder deploys the three Go services. Kafka, Elasticsearch, and PostgreSQL
are installed with Helm because writing production StatefulSets is outside this
beginner project.

## 1. Start a local cluster

```powershell
kind create cluster --name logstream
```

## 2. Install the stateful dependencies

```powershell
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo add elastic https://helm.elastic.co
helm repo update
helm install kafka bitnami/kafka --namespace logstream --create-namespace --set listeners.client.protocol=PLAINTEXT
helm install postgresql bitnami/postgresql --namespace logstream --set auth.username=logstream --set auth.password=logstream --set auth.database=logstream
helm install elasticsearch elastic/elasticsearch --namespace logstream --set replicas=1 --set minimumMasterNodes=1
```

The ConfigMap uses the service names produced by these releases. If a chart
version uses different names, check `kubectl get svc -n logstream` and update
`configmap.yaml` and `secret.yaml` before applying the app.

## 3. Build and load the local images

```powershell
docker build -f Dockerfile.ingestion -t logstream-ingestion:latest .
docker build -f Dockerfile.consumer -t logstream-consumer:latest .
docker build -f Dockerfile.search -t logstream-search:latest .
kind load docker-image logstream-ingestion:latest logstream-consumer:latest logstream-search:latest --name logstream
kubectl apply -k k8s/base
kubectl get pods -n logstream -w
```

## 4. Exercise the pipeline

```powershell
kubectl port-forward -n logstream service/ingestion 8080:8080
curl.exe -X POST http://localhost:8080/ingest -H "Content-Type: application/json" -d "[{\"service\":\"demo\",\"level\":\"info\",\"message\":\"hello from Kubernetes\",\"metadata\":{}}]"
kubectl port-forward -n logstream service/search 8084:8084
curl.exe http://localhost:8084/search?service=demo
```

The consumer has no Service because it receives records from Kafka rather than
HTTP traffic. Its metrics can be viewed with `kubectl port-forward deployment/consumer 9090:9090 -n logstream` and `curl.exe http://localhost:9090/metrics`.
