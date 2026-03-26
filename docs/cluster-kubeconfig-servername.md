# Kubernetes kubeconfig ServerName Handling

## 1. Problem

A common real-world kubeconfig problem looks like this:

- kubeconfig `server` uses an internal hostname
- public access must use an external IP
- the API server certificate is still issued for the internal hostname

If you connect directly to the public IP without setting TLS `ServerName`,
certificate verification can fail.

## 2. Your real example

Your kubeconfig contains:

- kubeconfig server: `https://apiserver.cluster.local:6443`
- real external endpoint: `https://36.138.61.152:6443`

That means:

- transport target should be `36.138.61.152:6443`
- TLS host verification should still use `apiserver.cluster.local`

## 3. Current backend behavior

The backend now handles this case in `backend/internal/infrastructure/kubernetes/gateway.go`.

Resolution order for TLS `ServerName`:

1. `credentials.serverName`
2. host extracted from kubeconfig `server`
3. host extracted from `apiServer`

Actual connection target:

- `apiServer` if provided
- otherwise kubeconfig `server`

This means you can safely store:

```json
{
  "apiServer": "https://36.138.61.152:6443",
  "credentials": "{\"serverName\":\"apiserver.cluster.local\",\"namespace\":\"default\"}"
}
```

while still pasting the original kubeconfig content.

## 4. Recommended cluster payload

Example cluster create payload:

```json
{
  "name": "real-prod-cluster",
  "description": "Kubernetes cluster with public API endpoint",
  "apiServer": "https://36.138.61.152:6443",
  "environment": "prod",
  "authType": "kubeconfig",
  "kubeConfig": "PASTE_FULL_KUBECONFIG_HERE",
  "token": "",
  "caCert": "",
  "credentials": "{\"namespace\":\"default\",\"serverName\":\"apiserver.cluster.local\"}",
  "isPublic": false,
  "status": "active"
}
```

## 5. Why this works

- client cert, client key, and CA data still come from kubeconfig
- the network socket connects to the public IP
- TLS SNI and certificate verification use `apiserver.cluster.local`
- certificate trust remains valid

## 6. When to use `credentials.serverName`

Use it when:

- kubeconfig `server` is internal
- you override `apiServer` with public IP or public DNS
- the certificate name must stay aligned with the original internal host

Do not use it when:

- public endpoint hostname already matches the certificate
- you intentionally skip verification with insecure TLS

## 7. Related frontend field

The clusters page already exposes:

- `API Server`
- `KubeConfig`
- `Extended credentials JSON`

Example `Extended credentials JSON`:

```json
{"namespace":"default","serverName":"apiserver.cluster.local"}
```
