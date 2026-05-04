# Deployment Plan — Robo-Advisory Platform

**Architecture:** K3s self-managed (3 VPS) + Cloudflare Full Strict SSL + Supabase PostgreSQL + CloudAMQP RabbitMQ

**Traffic flow:**
```
Browser → Cloudflare (TLS termination + CDN) → VPS-1 :443
  → Traefik ingress controller
    → /api/*  → Go Service (ClusterIP :8081)
    → /*      → Nginx Service (ClusterIP :80) → React SPA static files
```

**Services map:**
| Component | Where | Notes |
|-----------|-------|-------|
| Operational node | K8s cluster (vps-1/vps-2) | 2 replicas for rolling deploys |
| Decisional node | K8s cluster (vps-3) | RabbitMQ consumer, no HTTP |
| Nginx frontend | K8s cluster | Serves React `/dist`, pure static |
| Traefik | K8s cluster (vps-1) | Routes traffic, holds TLS cert |
| PostgreSQL | Supabase (external, free) | Session mode pooler, sslmode=require |
| RabbitMQ | CloudAMQP (external, free) | Go publishes, Python consumes |

---

## Stage 1 — External Services Setup

### 1.1 Cloudflare

Cloudflare sits in front of your VPS and handles TLS for browsers. Your VPS only sees Cloudflare IPs, never raw browser connections. You use a **Cloudflare Origin Certificate** (not Let's Encrypt) for the Cloudflare → VPS leg.

- [x] Create Cloudflare account at cloudflare.com
- [x] Add your domain → Cloudflare will give you 2 nameservers to set at your registrar
- [x] Wait for nameserver propagation (can take up to 24h, usually ~15min)
- [x] In Cloudflare dashboard → SSL/TLS → set mode to **Full (Strict)**
  - "Flexible" = no encryption to origin (bad)
  - "Full" = encrypts but doesn't verify cert (acceptable but weaker)
  - "Full Strict" = encrypts + verifies origin cert (correct choice)
- [x] Generate origin certificate: SSL/TLS → Origin Server → Create Certificate
  - Key type: RSA 2048
  - Validity: 15 years
  - Hostnames: `yourdomain.com`, `*.yourdomain.com`
  - Download both `origin.pem` (certificate) and `origin.key` (private key) — store securely, you'll need them in Stage 4

### 1.2 Supabase

- [x] Create Supabase account, create new project (pick region closest to your DO VPS region)
- [x] Wait for project provisioning (~2 min)
- [x] Settings → Database → Connection string section:
  - Copy the **Session mode** URL — it uses port **5432**
  - Format: `postgresql://postgres.[project-ref]:[password]@aws-0-[region].pooler.supabase.com:5432/postgres`
  - **Do NOT use Transaction mode (port 6543)** — it breaks GORM because GORM uses prepared statements which require session-level state
- [ ] Settings → Network → Restrict access:
  - For thesis: allow `0.0.0.0/0` (open to all, still password-protected)
  - For production: add only your 3 VPS public IPs

### 1.3 CloudAMQP

CloudAMQP is a managed RabbitMQ service. Your Go node publishes messages, Python node consumes them. Free tier gives 1M messages/month and 1 concurrent connection — fine for thesis.

- [x] Create account at cloudamqp.com
- [x] Create new instance → plan: **Little Lemur (free)**
- [x] Pick region matching your VPS region (reduces latency)
- [x] Copy the **AMQP URL** from the instance details page
  - Format: `amqps://user:pass@host/vhost` (note `amqps://` — TLS enabled)
  - This URL goes into both Go and Python secrets in Stage 4

### 1.4 GitHub Container Registry

The CI/CD pipeline builds Docker images and pushes them to ghcr.io. K8s nodes pull from there.

- [x] Go to GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic)
- [x] Generate token with scopes: `write:packages`, `read:packages`, `delete:packages`
- [x] Save the token — you'll add it to GitHub Actions secrets as `GHCR_TOKEN` in Stage 8

---

## Stage 2 — VPS Provisioning

### Node layout

| Node | Hostname | Role | Recommended DO Size |
|------|----------|------|---------------------|
| vps-1 | `k3s-master` | K3s control-plane + runs Traefik | s-2vcpu-4gb ($24/mo) |
| vps-2 | `k3s-worker-1` | K3s worker — runs Go + Nginx pods | s-2vcpu-4gb ($24/mo) |
| vps-3 | `k3s-worker-2` | K3s worker — runs Python consumer pods | s-2vcpu-2gb ($18/mo) |

All 3 in the **same DO datacenter region** (e.g., Frankfurt `fra1`) — this gives them a private network interface for inter-node communication so K3s traffic doesn't go over public internet.

### 2.1 Provision VPS

- [x] Create all 3 droplets in the same region
- [x] Select Ubuntu 22.04 LTS (K3s officially supports it)
- [x] Add your SSH public key during creation
- [x] Enable **Private Networking** on all 3 (DO panel option) — this gives each node a `10.x.x.x` private IP
- [x] Note down both public IPs and private IPs for all 3 nodes

### 2.2 Initial server setup (repeat on all 3)

```bash
# SSH in as root
ssh root@<node-public-ip>

# Update packages first
apt update && apt upgrade -y

# Set hostname (replace with k3s-master / k3s-worker-1 / k3s-worker-2)
hostnamectl set-hostname k3s-master
```

**Create non-root sudo user (do this before locking root out):**
```bash
adduser andrei
usermod -aG sudo andrei

# Copy root's authorized_keys to new user
mkdir -p /home/andrei/.ssh
cp ~/.ssh/authorized_keys /home/andrei/.ssh/
chown -R andrei:andrei /home/andrei/.ssh
chmod 700 /home/andrei/.ssh
chmod 600 /home/andrei/.ssh/authorized_keys
```

- [x] Create non-root user, copy SSH key, verify login in second terminal before continuing
- [x] **Do not close root session until new user login confirmed**

**Harden SSH:**
```bash
# Move SSH off port 22 (reduces automated scan noise by ~99%)
sed -i 's/#Port 22/Port 2222/' /etc/ssh/sshd_config

# Disable password auth
sed -i 's/#PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config

# Disable root login
sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin no/' /etc/ssh/sshd_config

# OR (if already uncommented with a different value):
# sed -i 's/PermitRootLogin yes/PermitRootLogin no/' /etc/ssh/sshd_config

# Whitelist only your user — any other account is rejected at daemon level regardless of key
echo "AllowUsers andrei" >> /etc/ssh/sshd_config

# OR (if `Permission denied`)
# echo "AllowUsers andrei" | sudo tee -a /etc/ssh/sshd_config

# Ubuntu 24.04 uses socket activation — plain `systemctl restart ssh` won't pick up port changes
systemctl daemon-reload
systemctl restart ssh.socket
```

SSH config will be finalized with Tailscale IPs in section 2.2.5 — skip for now.

**Install fail2ban (auto-bans IPs after repeated failed auth):**
```bash
apt install fail2ban -y
systemctl enable --now fail2ban
# Default config: ban after 5 failed SSH attempts for 10 min — sufficient for thesis
```

**Enable automatic security updates:**
```bash
apt install unattended-upgrades -y
dpkg-reconfigure --priority=low unattended-upgrades
# Select "Yes" — applies security-only updates automatically
```

- [x] SSH harden done (port 2222, no root, no password auth)
- [x] fail2ban running: `systemctl status fail2ban`
- [x] Unattended-upgrades enabled

### 2.2.5 Tailscale — private mesh network for SSH

Tailscale gives each node (and your laptops/machines) a stable private IP (`100.x.x.x`) that works from anywhere. Once set up, you SSH via the Tailscale IP — no need to whitelist your public IPs. Port 2222 is removed from the public firewall entirely. Tailscale authenticates devices via your account; only devices you approve can join the network.

**On each VPS (repeat on all 3):**
```bash
# Install Tailscale
curl -fsSL https://tailscale.com/install.sh | sh

# Start and authenticate — this prints a URL, open it in your browser to approve the node
tailscale up

# After auth, get this node's Tailscale IP
tailscale ip -4
# Note it down: e.g. 100.x.x.x  ← use this for SSH instead of the public IP
```

**On your local machines (every machine you SSH from):**

Install from https://tailscale.com/download and run `tailscale up`. Once authenticated, all your devices are on the same mesh.

**Update your local `~/.ssh/config` to use Tailscale IPs:**
```
Host k3s-master
    HostName 100.x.x.x        # k3s-master Tailscale IP
    User andrei
    Port 2222
    IdentityFile ~/.ssh/id_ed25519

Host k3s-worker-1
    HostName 100.x.x.x        # k3s-worker-1 Tailscale IP
    User andrei
    Port 2222
    IdentityFile ~/.ssh/id_ed25519

Host k3s-worker-2
    HostName 100.x.x.x        # k3s-worker-2 Tailscale IP
    User andrei
    Port 2222
    IdentityFile ~/.ssh/id_ed25519
```

- [x] Install Tailscale on all 3 VPS nodes and authenticate each
- [x] Install Tailscale on all your local machines
- [x] Note down each node's Tailscale IP (`tailscale ip -4`)
- [x] Update `~/.ssh/config` with Tailscale IPs
- [x] Verify SSH works via Tailscale: `ssh andrei@<tailscale-ip> -p 2222`
- [x] After confirming Tailscale SSH works — remove port 2222 from public firewall (next section)

### 2.3 Firewall rules

K3s nodes must reach each other. Use DO Cloud Firewall (panel) or `ufw` on each node.

SSH is no longer in the public firewall — it's only reachable via Tailscale (which runs over UDP 41641, handled automatically by Tailscale's own process, not ufw).

**On all 3 nodes — inbound allow:**
| Port | Protocol | Source | Purpose |
|------|----------|--------|---------|
| 41641 | UDP | 0.0.0.0/0 | Tailscale (needs to be open for mesh connectivity) |
| 2222 | TCP | Tailscale subnet (100.64.0.0/10) | SSH — not reachable from public internet |
| 6443 | TCP | 0.0.0.0/0 | K8s API server (GitHub Actions runners have dynamic IPs) |
| 8472 | UDP | other 2 nodes (private IPs) | Flannel VXLAN overlay network |
| 10250 | TCP | other 2 nodes (private IPs) | kubelet API (health checks between nodes) |
| 80 | TCP | Cloudflare IPs only | HTTP (Traefik, vps-1 only) |
| 443 | TCP | Cloudflare IPs only | HTTPS (Traefik, vps-1 only) |

```bash
# Run on all 3 nodes after Tailscale is up (section 2.2.5)

# 1. Default: deny all incoming, allow all outgoing
ufw default deny incoming
ufw default allow outgoing

# 2. Allow Tailscale UDP (required for mesh to form)
ufw allow 41641/udp

# 3. Allow SSH only from Tailscale subnet — unreachable from public internet
ufw allow from 100.64.0.0/10 to any port 2222 proto tcp

# 4. Allow K8s API (GitHub Actions needs this from public internet)
ufw allow 6443/tcp

# 5. Allow Flannel VXLAN between nodes (use private DO IPs)
ufw allow from <PRIVATE_IP_NODE_2> to any port 8472 proto udp
ufw allow from <PRIVATE_IP_NODE_3> to any port 8472 proto udp

# 6. Allow kubelet health checks between nodes (use private DO IPs)
ufw allow from <PRIVATE_IP_NODE_2> to any port 10250 proto tcp
ufw allow from <PRIVATE_IP_NODE_3> to any port 10250 proto tcp

# 7. Enable UFW
ufw --force enable
ufw status verbose
```

**Cloudflare IP ranges for ports 80/443** — restricting to these means bots hitting your VPS IP directly are blocked at the firewall level, bypassing Cloudflare's DDoS protection. Run this script on **vps-1 only** (the Traefik node):

```bash
# Add ufw rules for all Cloudflare IPv4 ranges (ports 80 + 443)
for ip in $(curl -s https://www.cloudflare.com/ips-v4); do 
  ufw allow from $ip to any port 80 proto tcp
  ufw allow from $ip to any port 443 proto tcp;
done
```

> **Note on port 6443:** Must stay open to `0.0.0.0/0` for GitHub Actions. K8s auth (kubeconfig cert) still protects it — unauthenticated requests are rejected by K8s itself.

- [x] Apply firewall rules to all 3 nodes (ufw commands above)
- [x] Apply Cloudflare ranges for 80/443 on vps-1 only
- [x] Verify `ufw status verbose` shows expected rules on each node
- [x] Verify nodes can ping each other on private IPs: `ping 10.x.x.x`
- [x] Verify SSH works via Tailscale: `ssh andrei@<tailscale-ip> -p 2222`
- [x] Verify port 2222 is **not** reachable via public IP: `ssh andrei@<public-ip> -p 2222` should time out

---

## Stage 3 — K3s Cluster Bootstrap

K3s is a lightweight Kubernetes distribution — single binary, production-grade, much easier to set up than kubeadm. K3s ships with Traefik as a default ingress controller. The master was installed with `--disable traefik` to prevent the bundled version from running; Traefik is instead installed via Helm (section 3.4) so we have full control over the version and configuration.

### 3.1 Install K3s on master

```bash
# SSH into vps-1 (k3s-master)
curl -sfL https://get.k3s.io | sh -s - server \
  --cluster-init \
  --disable traefik \
  --node-ip <vps-1-private-ip> \
  --advertise-address <vps-1-public-ip> \
  --tls-san <vps-1-public-ip>
```

- `--cluster-init` initializes embedded etcd (makes the node a control-plane)
- `--disable traefik` prevents K3s's bundled Traefik from claiming ports 80/443 — we install Traefik via Helm in section 3.4 instead
- `--node-ip` tells K3s to use the private IP for inter-node communication

- [x] Run the install command on vps-1
- [x] Verify K3s is running: `systemctl status k3s`
- [x] Copy the node join token:
  ```bash
  cat /var/lib/rancher/k3s/server/node-token
  ```

### 3.2 Join worker nodes

Run on **vps-2** and **vps-3** (replace placeholders):

```bash
curl -sfL https://get.k3s.io | K3S_URL=https://<vps-1-public-ip>:6443 \
  K3S_TOKEN=<token-from-master> \
  sh -s - agent --node-ip <this-node-private-ip>
```

- [x] Join vps-2 (`k3s-worker-1`)
- [x] Join vps-3 (`k3s-worker-2`)

### 3.3 Configure local kubectl access

```bash
# On vps-1, get the kubeconfig
cat /etc/rancher/k3s/k3s.yaml

# Copy to your local machine, replace 127.0.0.1 with vps-1 public IP
# Save as ~/.kube/config
```

- [x] Copy kubeconfig to local machine and update server IP
- [x] Verify cluster: `kubectl get nodes` — should show 3 nodes all `Ready`

```
NAME            STATUS   ROLES                  AGE
k3s-master      Ready    control-plane,master   2m
k3s-worker-1    Ready    <none>                 1m
k3s-worker-2    Ready    <none>                 1m
```

### 3.4 Install Traefik ingress controller (via Helm)

Traefik is a Kubernetes-native reverse proxy. It watches `Ingress` objects and routes traffic accordingly. It runs as a pod on the cluster and binds to ports 80/443 on vps-1.

K3s's built-in ServiceLB (klipper-lb) handles the `LoadBalancer` service type on bare metal — but because K3s was installed with `--node-ip <private-ip>`, ServiceLB binds only to the **private** IP by default. Cloudflare traffic hits the **public** IP, so we must explicitly add the public IP via `service.externalIPs` when installing Traefik.

**Install Helm first (on your local machine or on vps-1):**
```bash
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
```

**Install Traefik:**
```bash
helm repo add traefik https://helm.traefik.io/traefik
helm repo update

helm install traefik traefik/traefik \
  --namespace traefik \
  --create-namespace \
  --set "ports.web.http.redirections.entryPoint.to=websecure" \
  --set "ports.web.http.redirections.entryPoint.scheme=https" \
  --set "service.externalIPs[0]=<vps-1-public-ip>"
```

- `ports.web.http.redirections.entryPoint.to=websecure` — HTTP → HTTPS redirect at Traefik level (belt-and-suspenders alongside Cloudflare's edge redirect)
- `service.externalIPs[0]` — forces Traefik's LoadBalancer service to accept traffic on the public IP; without this, ServiceLB only binds to the private `10.x.x.x` IP and Cloudflare traffic never reaches Traefik
- `externalIPs` is handled by kube-proxy via iptables DNAT — `ss -tlnp` will **not** show ports 80/443 as listening sockets; this is expected and correct
- TLS termination is handled per-Ingress via the origin cert secret (section 4.2)

- [x] Install Helm
- [x] Add Traefik Helm repo and install
- [x] Wait for Traefik pod to be ready:
  ```bash
  kubectl wait --namespace traefik \
    --for=condition=ready pod \
    --selector=app.kubernetes.io/name=traefik \
    --timeout=120s
  ```
- [x] Verify the service shows the public IP under EXTERNAL-IP:
  ```bash
  kubectl get svc -n traefik
  # EXTERNAL-IP column should include 165.245.210.201 alongside the private IPs
  ```
- [x] Verify svclb pods are running:
  ```bash
  kubectl get pods -n kube-system | grep svclb
  ```
  Note: `ss -tlnp | grep ':80\|:443'` will show no output — this is expected. kube-proxy handles `externalIPs` via iptables DNAT, not a real socket.

---

## Stage 4 — Namespace, Secrets & Config

All application resources live in the `investpilot` namespace to keep them isolated from system pods.

### 4.1 Create namespace

```bash
kubectl create namespace investpilot
```

### 4.2 TLS secret (Cloudflare origin certificate)

This is the certificate nginx-ingress presents to Cloudflare for the encrypted Cloudflare → VPS leg. You downloaded `origin.pem` and `origin.key` in Stage 1.

```bash
kubectl create secret tls cloudflare-origin-tls \
  --cert=origin.pem \
  --key=origin.key \
  -n investpilot
```

- [ ] Create TLS secret

### 4.3 Go operational node secrets

```bash
kubectl create secret generic go-secrets \
  --from-literal=DATABASE_URL="postgresql://postgres.[ref]:[pass]@aws-0-[region].pooler.supabase.com:5432/postgres?sslmode=require" \
  --from-literal=JWT_SECRET="<strong-random-string-min-32-chars>" \
  --from-literal=STRIPE_KEY="sk_live_..." \
  --from-literal=RABBITMQ_URL="amqps://user:pass@host/vhost" \
  -n investpilot
```

These map to the env vars your Operational node already reads from `.env-operational-node`. The secret keys must match exactly what `os.Getenv()` expects in your code.

- [ ] Create go-secrets

### 4.4 Python decisional node secrets

```bash
kubectl create secret generic python-secrets \
  --from-literal=DATABASE_URL="postgresql://postgres.[ref]:[pass]@aws-0-[region].pooler.supabase.com:5432/postgres?sslmode=require" \
  --from-literal=RABBITMQ_URL="amqps://user:pass@host/vhost" \
  -n investpilot
```

- [ ] Create python-secrets

### 4.5 Verify secrets exist

```bash
kubectl get secrets -n investpilot
# Should list: cloudflare-origin-tls, go-secrets, python-secrets
```

- [ ] Verify all 3 secrets present

---

## Stage 5 — Dockerfiles & Nginx Config

### 5.1 Operational node — `operational-node/Dockerfile`

Multi-stage build: compile in a Go image, copy binary into a minimal runtime image.

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8081
CMD ["./server"]
```

- [ ] Create `operational-node/Dockerfile`
- [ ] Test build locally: `docker build -t operational-node ./operational-node`

### 5.2 Decisional node — `decisional-node/Dockerfile`

```dockerfile
FROM python:3.11-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
CMD ["python", "main.py"]
```

- [ ] Create `decisional-node/Dockerfile`
- [ ] Test build locally: `docker build -t decisional-node ./decisional-node`

### 5.3 Nginx frontend — `frontend/Dockerfile`

Two-stage: Node builds React, nginx serves the output.

```dockerfile
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:1.25-alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
```

- [ ] Create `frontend/Dockerfile`
- [ ] Test build locally: `docker build -t nginx-frontend ./frontend`

### 5.4 Nginx config — `frontend/nginx.conf`

Nginx only serves static files here. It does **not** proxy `/api/*` — that routing is handled by the Ingress controller upstream.

```nginx
server {
    listen 80;
    server_name _;
    root /usr/share/nginx/html;
    index index.html;

    # SPA fallback: serve index.html for all routes so React Router works
    location / {
        try_files $uri $uri/ /index.html;
    }

    # Cache static assets aggressively
    location ~* \.(js|css|png|jpg|svg|ico|woff2)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

- [ ] Create `frontend/nginx.conf`
- [ ] Test build locally: `docker build -t nginx-frontend ./frontend`

### 5.5 Push images to ghcr.io

```bash
# Login
echo $GHCR_TOKEN | docker login ghcr.io -u <github-username> --password-stdin

# Tag and push all 3
docker tag operational-node ghcr.io/<github-username>/investpilot-operational-node:latest
docker tag decisional-node ghcr.io/<github-username>/investpilot-decisional-node:latest
docker tag nginx-frontend ghcr.io/<github-username>/investpilot-nginx-frontend:latest

docker push ghcr.io/<github-username>/investpilot-operational-node:latest
docker push ghcr.io/<github-username>/investpilot-decisional-node:latest
docker push ghcr.io/<github-username>/investpilot-nginx-frontend:latest
```

- [ ] Push all 3 images to ghcr.io
- [ ] Make packages public in GitHub (repo → Packages → each image → visibility: public) OR configure imagePullSecrets in K8s

---

## Stage 6 — Kubernetes Manifests

Create a `k8s/` directory in the repo root for all manifests.

### 6.1 `k8s/operational-deployment.yaml`

2 replicas so rolling updates don't cause downtime. Secrets injected as env vars.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: operational-node
  namespace: investpilot
spec:
  replicas: 2
  selector:
    matchLabels:
      app: operational-node
  template:
    metadata:
      labels:
        app: operational-node
    spec:
      containers:
        - name: operational-node
          image: ghcr.io/<github-username>/investpilot-operational-node:latest
          ports:
            - containerPort: 8081
          envFrom:
            - secretRef:
                name: go-secrets
---
apiVersion: v1
kind: Service
metadata:
  name: operational-service
  namespace: investpilot
spec:
  selector:
    app: operational-node
  ports:
    - port: 8081
      targetPort: 8081
```

- [ ] Create `k8s/operational-deployment.yaml`

### 6.2 `k8s/decisional-deployment.yaml`

No Service needed — Decisional node only makes outbound connections (to RabbitMQ and Supabase), nothing connects inbound to it.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: decisional-node
  namespace: investpilot
spec:
  replicas: 1
  selector:
    matchLabels:
      app: decisional-node
  template:
    metadata:
      labels:
        app: decisional-node
    spec:
      containers:
        - name: decisional-node
          image: ghcr.io/<github-username>/investpilot-decisional-node:latest
          envFrom:
            - secretRef:
                name: python-secrets
```

- [ ] Create `k8s/decisional-deployment.yaml`

### 6.3 `k8s/nginx-deployment.yaml`

ConfigMap holds the nginx.conf so it can be updated without rebuilding the image.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: nginx-config
  namespace: investpilot
data:
  default.conf: |
    server {
        listen 80;
        server_name _;
        root /usr/share/nginx/html;
        index index.html;
        location / {
            try_files $uri $uri/ /index.html;
        }
        location ~* \.(js|css|png|jpg|svg|ico|woff2)$ {
            expires 1y;
            add_header Cache-Control "public, immutable";
        }
    }
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-frontend
  namespace: investpilot
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx-frontend
  template:
    metadata:
      labels:
        app: nginx-frontend
    spec:
      containers:
        - name: nginx-frontend
          image: ghcr.io/<github-username>/investpilot-nginx-frontend:latest
          ports:
            - containerPort: 80
          volumeMounts:
            - name: nginx-config
              mountPath: /etc/nginx/conf.d/default.conf
              subPath: default.conf
      volumes:
        - name: nginx-config
          configMap:
            name: nginx-config
---
apiVersion: v1
kind: Service
metadata:
  name: nginx-service
  namespace: investpilot
spec:
  selector:
    app: nginx-frontend
  ports:
    - port: 80
      targetPort: 80
```

- [ ] Create `k8s/nginx-deployment.yaml`

### 6.4 `k8s/ingress.yaml`

The Ingress object tells Traefik how to route traffic. `/api` is listed first — Traefik matches more specific paths first, so API calls go to Go and everything else goes to the Nginx frontend.

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: investpilot-ingress
  namespace: investpilot
  annotations:
    traefik.ingress.kubernetes.io/router.entrypoints: websecure
spec:
  ingressClassName: traefik
  tls:
    - hosts:
        - yourdomain.com
      secretName: cloudflare-origin-tls
  rules:
    - host: yourdomain.com
      http:
        paths:
          - path: /api
            pathType: Prefix
            backend:
              service:
                name: operational-service
                port:
                  number: 8081
          - path: /
            pathType: Prefix
            backend:
              service:
                name: nginx-service
                port:
                  number: 80
```

- `ingressClassName: traefik` — explicitly binds this Ingress to the Traefik controller
- `router.entrypoints: websecure` — Traefik only serves this route on the HTTPS (443) entrypoint

- [ ] Create `k8s/ingress.yaml` (replace `yourdomain.com`)

### 6.5 Apply everything

```bash
kubectl apply -f k8s/ -n investpilot
```

- [ ] Apply all manifests
- [ ] Check all pods running (may take ~1 min for image pulls):
  ```bash
  kubectl get pods -n investpilot
  # Expected:
  # operational-node-xxx       Running
  # operational-node-yyy       Running   (2 replicas)
  # decisional-node-xxx        Running
  # nginx-frontend-xx          Running
  ```
- [ ] Check services: `kubectl get svc -n investpilot`
- [ ] Check ingress: `kubectl get ingress -n investpilot`
- [ ] If a pod fails: `kubectl logs <pod-name> -n investpilot` and `kubectl describe pod <pod-name> -n investpilot`

---

## Stage 7 — DNS & Cloudflare Routing

### 7.1 Point DNS to your cluster

- [ ] In Cloudflare DNS → Add record:
  - Type: `A`
  - Name: `@` (root domain) or `yourdomain.com`
  - IPv4: vps-1 public IP (nginx-ingress listens here on 80/443)
  - Proxy: **Enabled** (orange cloud) — traffic goes through Cloudflare
- [ ] Optionally add `www` CNAME → `yourdomain.com` (also proxied)

### 7.2 Verify

- [ ] Wait ~2 min for DNS to propagate
- [ ] `curl -I https://yourdomain.com` — expect `200 OK` and valid TLS (Cloudflare cert in browser)
- [ ] `curl https://yourdomain.com/api/v1/health` — expect Operational node health response (your health endpoint)
- [ ] Open browser → `https://yourdomain.com` → React app loads
- [ ] Open browser DevTools → Network tab → confirm `/api/*` requests return data from Operational node

---

## Stage 8 — CI/CD Pipeline

### 8.1 Firewall prerequisite

GitHub Actions hosted runners have dynamic IPs. Port 6443 must be open to all so runners can reach the K3s API server.

- [ ] Confirm port `6443` is open to `0.0.0.0/0` on vps-1 (set in Stage 2)

### 8.2 Prepare kubeconfig secret

The runner needs credentials to talk to the cluster. Export and base64-encode your kubeconfig:

```bash
# On your local machine (with working kubectl)
cat ~/.kube/config | base64 -w 0
# Copy the output
```

- [ ] Add `KUBECONFIG_B64` to GitHub Actions secrets (repo → Settings → Secrets → Actions)
- [ ] Add `GHCR_TOKEN` to GitHub Actions secrets

### 8.3 Create workflow — `.github/workflows/deploy.yml`

```yaml
name: Build and Deploy

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Login to ghcr.io
        run: echo "${{ secrets.GHCR_TOKEN }}" | docker login ghcr.io -u ${{ github.actor }} --password-stdin

      - name: Build and push Operational node image
        run: |
          docker build -t ghcr.io/${{ github.repository_owner }}/investpilot-operational-node:${{ github.sha }} ./operational-node
          docker push ghcr.io/${{ github.repository_owner }}/investpilot-operational-node:${{ github.sha }}

      - name: Build and push Decisional node image
        run: |
          docker build -t ghcr.io/${{ github.repository_owner }}/investpilot-decisional-node:${{ github.sha }} ./decisional-node
          docker push ghcr.io/${{ github.repository_owner }}/investpilot-decisional-node:${{ github.sha }}

      - name: Build and push Nginx frontend image
        run: |
          docker build -t ghcr.io/${{ github.repository_owner }}/investpilot-nginx-frontend:${{ github.sha }} ./frontend
          docker push ghcr.io/${{ github.repository_owner }}/investpilot-nginx-frontend:${{ github.sha }}

      - name: Set up kubectl
        run: |
          mkdir -p ~/.kube
          echo "${{ secrets.KUBECONFIG_B64 }}" | base64 -d > ~/.kube/config

      - name: Update image tags in cluster
        run: |
          kubectl set image deployment/operational-node operational-node=ghcr.io/${{ github.repository_owner }}/investpilot-operational-node:${{ github.sha }} -n investpilot
          kubectl set image deployment/decisional-node decisional-node=ghcr.io/${{ github.repository_owner }}/investpilot-decisional-node:${{ github.sha }} -n investpilot
          kubectl set image deployment/nginx-frontend nginx-frontend=ghcr.io/${{ github.repository_owner }}/investpilot-nginx-frontend:${{ github.sha }} -n investpilot

      - name: Wait for rollout
        run: |
          kubectl rollout status deployment/operational-node -n investpilot --timeout=120s
          kubectl rollout status deployment/nginx-frontend -n investpilot --timeout=120s
```

Note: uses commit SHA as image tag (not `latest`) — each deploy is traceable and rollback is possible with `kubectl rollout undo`.

- [ ] Create `.github/workflows/deploy.yml`
- [ ] Push to `main` → verify Actions tab shows green pipeline
- [ ] Verify new pods running with correct image SHA: `kubectl describe pod <operational-node-xxx> -n investpilot | grep Image`
- [ ] Verify zero-downtime: Operational node has 2 replicas → K8s rolls one at a time → no downtime window

---

## Stage 9 — Validation & Smoke Tests

Run these after full deployment to verify all system paths work end-to-end.

- [ ] **Auth flow**: Register new user → verify email arrives → log in → JWT issued (Operational node)
- [ ] **Onboarding**: Complete risk profile questionnaire → model portfolio assigned
- [ ] **Portfolio generation**: Operational node publishes `CMD_GENERATE` → CloudAMQP receives it → Decisional node consumes → writes holdings to Supabase → Operational node returns portfolio data to frontend
- [ ] **Forecast**: Request forecast → Operational node creates pending `ForecastResult` row → publishes `CMD_FORECAST` → Decisional node computes → writes result → frontend polling returns completed result
- [ ] **Rebalancing**: Trigger manual rebalance (or wait for cron) → Operational node publishes `CMD_REBALANCE_USER` → Decisional node computes weight deltas → Operational node writes new holdings
- [ ] **Stripe payment**: Complete a deposit flow → Stripe webhook received by Operational node → wallet updated
- [ ] **DB writes visible**: Supabase dashboard → Table Editor → check `users`, `holdings`, `forecast_results` tables have data
- [ ] **RabbitMQ traffic**: CloudAMQP dashboard → show message rates during above tests
- [ ] **TLS valid**: Browser padlock shows Cloudflare certificate, no mixed-content warnings
- [ ] **Cloudflare proxy active**: `curl -I https://yourdomain.com` response headers include `cf-ray` header

---

## Stage 10 — Hardening (Optional — good for thesis)

These are production-quality improvements that demonstrate operational maturity.

### Resource limits — prevent one pod from starving others

Add to each Deployment's container spec:
```yaml
resources:
  requests:
    memory: "128Mi"
    cpu: "100m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```
- [ ] Add resource requests/limits to all 3 Deployments

### Health probes — K8s restarts unhealthy pods automatically

Add to Operational node Deployment (requires a `/health` endpoint):
```yaml
livenessProbe:
  httpGet:
    path: /api/v1/health
    port: 8081
  initialDelaySeconds: 15
  periodSeconds: 20
readinessProbe:
  httpGet:
    path: /api/v1/health
    port: 8081
  initialDelaySeconds: 5
  periodSeconds: 10
```
- [ ] Add liveness + readiness probes to Operational node Deployment
- [ ] Verify probe status: `kubectl describe pod <operational-node-xxx> -n investpilot`

### Horizontal Pod Autoscaler — scale Decisional node consumers under load

If many users trigger forecasts/rebalancing simultaneously, one decisional node pod may bottleneck. HPA adds replicas automatically when CPU > 70%.
```bash
kubectl autoscale deployment decisional-node \
  --cpu-percent=70 \
  --min=1 --max=5 \
  -n investpilot
```
- [ ] Configure HPA on Decisional node deployment

### Cloudflare firewall rules

- [ ] Rate limit `/api/v1/auth/*` — max 10 requests/min per IP (prevents brute force)
- [ ] Block non-GET requests to `/` and static asset paths (JS/CSS should never receive POST)

### Kubernetes NetworkPolicy — isolate Decisional node pod

Decisional node should only make outbound connections (to RabbitMQ and DB). It should never receive inbound HTTP. A NetworkPolicy enforces this at the network level:
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: decisional-isolation
  namespace: investpilot
spec:
  podSelector:
    matchLabels:
      app: decisional-node
  policyTypes:
    - Ingress
  ingress: []  # deny all inbound
```
- [ ] Apply NetworkPolicy for Decisional node pod
- [ ] Verify Operational node pods can still reach Supabase + CloudAMQP (outbound not affected)

### Supabase backups

- [ ] Verify automated backups enabled: Supabase dashboard → Settings → Backups
  - Free tier: daily backups, 7-day retention (enabled by default)

---

## Cost Summary

| Item | Provider | Cost/mo |
|------|----------|---------|
| vps-1 (k3s-master, 2vCPU/4GB) | DigitalOcean | $24 |
| vps-2 (k3s-worker-1, 2vCPU/4GB) | DigitalOcean | $24 |
| vps-3 (k3s-worker-2, 2vCPU/2GB) | DigitalOcean | $18 |
| PostgreSQL | Supabase free tier | $0 |
| RabbitMQ | CloudAMQP free tier | $0 |
| SSL + DNS + CDN + DDoS | Cloudflare free tier | $0 |
| Container registry | ghcr.io (GitHub) | $0 |
| **Total** | | **~$66/mo** |
