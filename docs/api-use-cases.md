# API Use Cases

## 1. Base values

```bash
BASE_URL=http://127.0.0.1:8080
LOGIN=admin
PASSWORD=admin123456
```

## 2. Health

```bash
curl "$BASE_URL/healthz"
curl "$BASE_URL/readyz"
```

## 3. Login

```bash
curl -X POST "$BASE_URL/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "login": "admin",
    "password": "admin123456"
  }'
```

```bash
ACCESS_TOKEN=replace-with-access-token
```

## 4. Current user

```bash
curl "$BASE_URL/api/users/me" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

## 5. Models

Create a model and validate it before saving:

```bash
curl -X POST "$BASE_URL/api/models" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "DeepSeek 32B",
    "provider": "openai-compatible",
    "model": "DeepSeek-R1-Distill-Qwen-32B",
    "baseUrl": "http://10.55.139.5:38888/v1",
    "apiKey": "",
    "description": "validated before save",
    "capabilities": ["chat", "agent"],
    "isDefault": false,
    "isEnabled": true,
    "maxTokens": 2048,
    "temperature": 0.2,
    "topP": 0.9,
    "testBeforeSave": true
  }'
```

Test an existing model:

```bash
curl -X POST "$BASE_URL/api/models/1/test" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Update an existing model without forcing a save-time test:

```bash
curl -X PUT "$BASE_URL/api/models/1" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "DeepSeek 32B",
    "provider": "openai-compatible",
    "model": "DeepSeek-R1-Distill-Qwen-32B",
    "baseUrl": "http://10.55.139.5:38888/v1",
    "apiKey": "",
    "description": "update metadata only",
    "capabilities": ["chat", "agent"],
    "isDefault": false,
    "isEnabled": true,
    "maxTokens": 2048,
    "temperature": 0.2,
    "topP": 0.9,
    "testBeforeSave": false
  }'
```

Set a default model:

```bash
curl -X POST "$BASE_URL/api/models/1/set-default" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

## 6. Clusters

Create a cluster using kubeconfig plus an external API server:

```bash
curl -X POST "$BASE_URL/api/clusters" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "prod-cluster",
    "description": "Production cluster",
    "apiServer": "https://36.138.61.152:6443",
    "environment": "prod",
    "authType": "kubeconfig",
    "kubeConfig": "PASTE_FULL_KUBECONFIG_HERE",
    "token": "",
    "caCert": "",
    "credentials": "{\"namespace\":\"default\",\"serverName\":\"apiserver.cluster.local\"}",
    "isPublic": false,
    "status": "active"
  }'
```

Validate connectivity:

```bash
curl -X POST "$BASE_URL/api/clusters/1/validate" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Update cluster configuration:

```bash
curl -X PUT "$BASE_URL/api/clusters/1" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "prod-cluster",
    "description": "Updated production cluster",
    "apiServer": "https://36.138.61.152:6443",
    "environment": "prod",
    "authType": "kubeconfig",
    "kubeConfig": "",
    "isPublic": false,
    "status": "active"
  }'
```

Load the KubeSphere-style cluster overview:

```bash
curl "$BASE_URL/api/clusters/1/overview" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

List namespaces:

```bash
curl "$BASE_URL/api/clusters/1/namespaces" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

List pods:

```bash
curl "$BASE_URL/api/clusters/1/resources?type=pods&namespace=default" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Get one deployment:

```bash
curl "$BASE_URL/api/clusters/1/resources/deployments/nginx?namespace=default" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

List events:

```bash
curl "$BASE_URL/api/clusters/1/events?namespace=default" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Stream pod logs:

```bash
curl -N "$BASE_URL/api/clusters/1/pods/nginx-7dd7f7b6d6-abcde/logs?namespace=default&follow=true&tailLines=200" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Stream pod logs for a specific container:

```bash
curl -N "$BASE_URL/api/clusters/1/pods/nginx-7dd7f7b6d6-abcde/logs?namespace=default&container=nginx&follow=true&tailLines=200" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

## 7. Cluster approvals

Request resource deletion approval:

```bash
curl -X POST "$BASE_URL/api/clusters/1/actions/delete-resource" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "deployments",
    "name": "nginx",
    "namespace": "default"
  }'
```

Request deployment scaling approval:

```bash
curl -X POST "$BASE_URL/api/clusters/1/actions/scale-deployment" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "nginx",
    "namespace": "default",
    "replicas": 3
  }'
```

Request deployment restart approval:

```bash
curl -X POST "$BASE_URL/api/clusters/1/actions/restart-deployment" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "nginx",
    "namespace": "default"
  }'
```

Request YAML apply approval:

```bash
curl -X POST "$BASE_URL/api/clusters/1/actions/apply-yaml" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "default",
    "manifest": "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: demo\n  namespace: default\ndata:\n  hello: world"
  }'
```

Approve the request:

```bash
curl -X POST "$BASE_URL/api/agent/approvals/1/approve" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Reject the request:

```bash
curl -X POST "$BASE_URL/api/agent/approvals/1/reject" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

## 8. Agent and run events

Create a session:

```bash
curl -X POST "$BASE_URL/api/agent/sessions" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "生产集群排障",
    "modelId": 1,
    "clusterId": 1
  }'
```

Send a message:

```bash
curl -X POST "$BASE_URL/api/agent/sessions/1/messages" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "帮我看看 default 命名空间里异常的 Pod"
  }'
```

Replay run events:

```bash
curl "$BASE_URL/api/agent/runs/1/events" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Open the SSE stream:

```bash
curl -N "$BASE_URL/api/agent/runs/1/stream" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

## 9. Logs

List supported scopes:

```bash
curl "$BASE_URL/api/logs/scopes" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Query platform logs:

```bash
curl "$BASE_URL/api/logs?scope=runtime&cursor=0&limit=50" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Upload one client-side log:

```bash
curl -X POST "$BASE_URL/api/logs/client" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "level": "error",
    "message": "agent sse disconnected",
    "requestId": "",
    "runId": "12",
    "fields": {
      "route": "/agent",
      "reason": "network"
    }
  }'
```
