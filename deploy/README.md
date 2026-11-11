# Deploying to a VPS

On every push to `main`, GitHub Actions (`.github/workflows/deploy.yml`) builds a static Linux binary,
copies it to the VPS and restarts the service. The VPS holds **only** the binary and `config.json`
(DB migrations are embedded in the binary).

```
/opt/bloodlink/
  bloodlink-server        # current binary
  bloodlink-server.prev   # previous binary (auto-rollback)
  config.json             # production config, created by hand, never in git
  assets/ errors.log access.log   # created by the app
```

## One-time VPS setup (Ubuntu/Debian)

```bash
# 1. Users: "bloodlink" runs the app, "deploy" is used by CI over SSH
sudo useradd --system --no-create-home --shell /usr/sbin/nologin bloodlink
sudo adduser --disabled-password --gecos "" deploy
sudo usermod -aG bloodlink,systemd-journal deploy

# 2. App directory
sudo mkdir -p /opt/bloodlink
sudo chown deploy:bloodlink /opt/bloodlink
sudo chmod 775 /opt/bloodlink

# 3. Production config (copy your real values in)
sudo -u deploy nano /opt/bloodlink/config.json
sudo chmod 640 /opt/bloodlink/config.json
sudo chown deploy:bloodlink /opt/bloodlink/config.json

# 4. systemd service (copy deploy/bloodlink.service from this repo)
sudo cp bloodlink.service /etc/systemd/system/bloodlink.service
sudo systemctl daemon-reload
sudo systemctl enable bloodlink

# 5. Let "deploy" restart ONLY this service without a password
echo 'deploy ALL=(root) NOPASSWD: /usr/bin/systemctl restart bloodlink, /bin/systemctl restart bloodlink' \
  | sudo tee /etc/sudoers.d/bloodlink-deploy
sudo chmod 440 /etc/sudoers.d/bloodlink-deploy
sudo visudo -c

# 6. SSH key for CI (run on your PC, NOT the VPS)
ssh-keygen -t ed25519 -C "github-actions-bloodlink" -f bloodlink_deploy -N ""
#    put bloodlink_deploy.pub into /home/deploy/.ssh/authorized_keys on the VPS
#    put bloodlink_deploy (private key) into the GitHub secret VPS_SSH_KEY
```

`curl` must be installed on the VPS (the health check uses it).

## GitHub secrets

Repo → Settings → Environments → create **production** → add secrets:

| Secret        | Value                                  |
|---------------|----------------------------------------|
| `VPS_HOST`    | VPS IP or hostname                     |
| `VPS_USER`    | `deploy`                               |
| `VPS_SSH_KEY` | contents of the private key `bloodlink_deploy` |
| `VPS_PORT`    | SSH port (optional, default `22`)      |

The health check reads `serverPort` from `/opt/bloodlink/config.json` and calls `GET /ping`.
If it fails for 30 seconds, the previous binary is restored automatically and the job fails.

## Recommended: Nginx + HTTPS

Keep the Go port private and expose it through Nginx:

```nginx
server {
    server_name api.example.com;
    location / {
        proxy_pass http://127.0.0.1:8765;          # = serverPort
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;    # websockets
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

```bash
sudo certbot --nginx -d api.example.com
sudo ufw allow OpenSSH && sudo ufw allow 'Nginx Full' && sudo ufw enable
```

## Day-to-day

```bash
sudo systemctl status bloodlink
journalctl -u bloodlink -f
tail -f /opt/bloodlink/errors.log
```

Manual rollback: `cd /opt/bloodlink && mv bloodlink-server.prev bloodlink-server && sudo systemctl restart bloodlink`

## Notes
- VPS on ARM (e.g. Ampere/Graviton)? Change `GOARCH: amd64` → `arm64` in the workflow.
- `go test` is not run in CI: `pkg/notifications/sms_test.go` needs a real `config.json`.
