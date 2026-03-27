#!/bin/bash

# 配置
BASE_URL="http://127.0.0.1:8080"
LOGIN_USER="admin"
LOGIN_PASSWORD="admin123456"   # 可根据实际情况修改

# 检查 jq 是否可用
if ! command -v jq &> /dev/null; then
    echo "错误：需要 jq 命令来解析 JSON，请先安装 jq。"
    exit 1
fi

# 1. 登录并获取 access token
echo "1. 登录..."
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"login\":\"$LOGIN_USER\",\"password\":\"$LOGIN_PASSWORD\"}")

TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.data.tokens.accessToken')
if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
    echo "登录失败，响应：$LOGIN_RESPONSE"
    exit 1
fi
echo "登录成功，已获取 token。"

# 2. 列出 HTTP-safe capabilities
echo "2. 列出 HTTP-safe capabilities..."
curl -s -X GET "$BASE_URL/api/capabilities?audience=http" \
    -H "Authorization: Bearer $TOKEN" | jq .

# 3. 调用能力：列出 default 命名空间下的 pods
echo "3. 调用 builtin.cluster.resources 能力（列出 default 命名空间的 Pods）..."
curl -s -X POST "$BASE_URL/api/capabilities/builtin.cluster.resources/invoke" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "clusterId": 1,
        "namespace": "default",
        "payload": {
            "type": "pods"
        }
    }' | jq .

echo "完成。"