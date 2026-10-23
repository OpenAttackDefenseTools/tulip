# Sunflower on a fresh vulnbox (manual setup)

Tested on Fedora 44. Takes ~15 min plus image build time.

## 1. Install dependencies

```bash
sudo dnf install -y docker tcpdump git vim
sudo systemctl enable --now docker
sudo usermod -aG docker $USER   # re-login after this
```

Verify: `docker --version`, `docker compose version`.
(If `docker compose` is missing: `sudo dnf install -y docker-compose`.)

## 2. Get the code

```bash
git clone https://github.com/kBxAc/sunflower ~/sunflower
cd ~/sunflower
```

No git access from the box? Copy it from your machine instead:

```bash
# on your machine:
tar -czf sunflower.tar.gz -C ~/Desktop/projects sunflower
scp -P 2222 sunflower.tar.gz vuln@<vulnbox>:~/
# on the box:
tar -xzf ~/sunflower.tar.gz -C ~/
```

## 3. Configure

```bash
cd ~/sunflower
cp .env.example .env
```

Edit `.env` (minimum):

| Var | Value |
|---|---|
| `TRAFFIC_DIR_HOST` | `/traffic` — **must be absolute**. The default (`./services/test_pcap`) points at the bundled 2018 sample capture, so the dashboard fills with stale demo flows and your service never appears. |
| `TRAFFIC_DIR_DOCKER` | `/traffic` (leave as-is) |
| `BPF` | `tcp port 8080` (your service port — keeps noise out) |
| `TICK_START` | event start, e.g. `2026-10-22T00:00:00Z` |
| `TICK_LENGTH` | tick ms, e.g. `180000` |
| `FLAG_REGEX` | event flag format, e.g. `FAUST_[A-Za-z0-9+/=]+` |

```bash
sudo mkdir -p /traffic
```

## 4. SELinux (Fedora/RHEL only, skip on Debian)

Bind mounts are denied by default. Label the repo and traffic dir:

```bash
sudo chcon -R -t svirt_sandbox_file_t ~/sunflower /traffic
```

Also open the dashboard port:

```bash
sudo firewall-cmd --permanent --add-port=3000/tcp
sudo firewall-cmd --reload
```

## 5. Start the stack

```bash
cd ~/sunflower
docker compose up -d --build
```

First boot builds 5 images (go toolchain + pip + yarn) — give it 5–15 min.
Watch progress: `docker compose logs -f assembler timescale`.
Healthy when `docker compose ps` shows all `Up` and the assembler logs
`Copied N rows into table [flow]`.

Dashboard: **http://<vulnbox-ip>:3000**

## 6. Capture traffic

In a separate shell (tmux/screen recommended):

```bash
sudo chmod 777 /traffic
sudo tcpdump -i any -G 180 -w '/traffic/traffic_%Y-%m-%d_%H:%M:%S.pcap' 'tcp port 8080'
```

The assembler picks up each rotated file automatically. Confirm in the UI
(flows appear within ~1 min) or via logs:
`docker compose logs assembler | tail`.

## 7. Service names (important)

`services/api/configurations.py` has a **hardcoded service list**
(`helper` string). Add your box:

```python
10.x.x.x:8080 MyService 1
```

Anything unmatched lands in the `other` bucket. After editing:

```bash
docker compose up -d --build api
```

## 8. Useful commands

```bash
docker compose ps                 # status
docker compose logs -f assembler  # ingest activity
docker compose down               # stop everything (data stays in docker volumes)
docker compose down -v            # stop AND wipe the database
```

## 9. Verify everything

```bash
./scripts/vulnbox-check.sh
```

Checks docker, container status, `.env` paths, tcpdump, pcap freshness,
DB connectivity (assembler → timescale DNS), the dashboard on :3000 and
the service on :8080, plus box IPs. Exit 0 = healthy. Run it after setup
and whenever the dashboard looks dead.

## Troubleshooting (seen in practice)

**Timescale restart-loops, logs show `Permission denied` on
`/docker-entrypoint-initdb.d/*.sql`** — SELinux blocked the mount (step 4
skipped). Fix: `sudo chcon -R -t svirt_sandbox_file_t ~/sunflower`, then
`docker compose up -d --force-recreate timescale`. If it still loops, the
data volume caught a half-init: `docker compose down -v && docker compose up -d`
(safe only while the DB holds nothing real).

**Dashboard shows flows but not your service** — `.env` still points at the
sample capture. Set `TRAFFIC_DIR_HOST="/traffic"`, then
`docker compose down -v && docker compose up -d` so old sample rows don't
pollute the DB.

**Dashboard unreachable from your machine** — guest firewall/NAT, not the
stack. On the box `curl -m 5 http://127.0.0.1:3000/` must return 200
(the check script does this). If it works locally but not remotely, add a
port forward (e.g. VirtualBox NAT `127.0.0.1:13000 → guest:3000`) instead
of touching guest firewall rules.

## Alternative: sniff here, analyze elsewhere

If the vulnbox is too weak to run the stack, only do step 6 on the box
and rsync pcaps to the machine running Sunflower:

```bash
rsync -avz -e ssh ./traffic/ analyst@<sunflower-box>:/traffic
```
