# Frontend Initialization and Deployment

## 1. Local initialization

Install Node.js 20 or newer first.

Then run:

```bash
cd frontend
npm install
```

Create env file:

```bash
cp .env.example .env
```

Default example:

```bash
VITE_API_BASE_URL=http://127.0.0.1:8080
```

If you want to use the Vite proxy only, you can also leave the value empty and call the backend with same-origin paths.

## 2. Local development

Start backend:

```bash
cd backend
go run ./cmd/server
```

Start frontend:

```bash
cd frontend
npm run dev
```

Default frontend URL:

- `http://127.0.0.1:5173`

## 3. Production build

```bash
cd frontend
npm install
npm run build
```

Build output:

- `frontend/dist`

That folder is the only folder you need to serve as static assets.

## 4. Nginx deployment example

Assume:

- frontend files are deployed to `/srv/kubeclaw/frontend/dist`
- backend runs at `http://127.0.0.1:8080`
- public site domain is `console.example.com`

Example Nginx config:

```nginx
server {
    listen 80;
    server_name console.example.com;

    root /srv/kubeclaw/frontend/dist;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /healthz {
        proxy_pass http://127.0.0.1:8080;
    }

    location /readyz {
        proxy_pass http://127.0.0.1:8080;
    }
}
```

## 5. Recommended production strategy

Recommended approach:

1. build frontend into static files
2. serve frontend with Nginx
3. reverse proxy `/api`, `/healthz`, `/readyz` to the Go backend
4. keep frontend and backend under the same public domain when possible

Benefits:

- no CORS pain
- simple cookie and token flow later
- easier CDN and cache control
- cleaner deployment topology

## 6. Backend deployment notes

For backend deployment:

```bash
cd backend
go build -o kubeclaw-server ./cmd/server
```

Run with environment overrides when needed:

```bash
HTTP_ADDR=:8080 \
MYSQL_HOST=36.138.61.152 \
MYSQL_PORT=30036 \
MYSQL_USER=root \
MYSQL_PASSWORD=passw0rd \
MYSQL_DATABASE=opsbrain \
./kubeclaw-server
```

On Windows PowerShell:

```powershell
$env:HTTP_ADDR=':8080'
$env:MYSQL_HOST='36.138.61.152'
$env:MYSQL_PORT='30036'
$env:MYSQL_USER='root'
$env:MYSQL_PASSWORD='passw0rd'
$env:MYSQL_DATABASE='opsbrain'
.\kubeclaw-server.exe
```

## 7. Cloud deployment checklist

- backend can reach MySQL
- backend can reach Kubernetes API server
- frontend reverse proxy points to backend
- `JWT_SECRET` is replaced in production
- `DATA_SECRET` is replaced in production
- `LOG_LEVEL` and `LOG_ENCODING` are adjusted for the environment
- HTTPS is enabled at the ingress or load balancer
- firewall allows backend to reach `36.138.61.152:6443` when cluster validation is needed

## 8. Update workflow

Recommended release workflow:

1. backend changes
2. backend tests
3. frontend changes
4. frontend lint and build
5. update docs in `docs/`
6. deploy backend
7. deploy frontend
8. run login, health, and cluster smoke checks
