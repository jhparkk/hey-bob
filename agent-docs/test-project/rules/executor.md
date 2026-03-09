# rules/executor.md — Q @ test-project

## Q's Deployment Environment in this Project

### Environment Info
- OS: Linux (WSL2)
- Go: `/usr/local/go/bin` (Requires export PATH)
- Project: `/agent-devs/test-project/`
- Port: **8080**
- Execution Method: nohup background

### Standard Deployment Procedure
```bash
export PATH=$PATH:/usr/local/go/bin
cd /agent-devs/test-project/

# 1. Build
go build -o test-project .

# 2. Restart with systemd service (auto-start configured)
systemctl --user restart test-project
sleep 2

# 3. Health check
curl -s http://localhost:8080/health
```

### Server Management (systemd)
```bash
systemctl --user start test-project    # Start
systemctl --user stop test-project     # Stop
systemctl --user restart test-project  # Restart
systemctl --user status test-project   # Check status
```
- Auto-starts on WSL2 boot (loginctl linger configured)
- Auto-restart after 5 seconds on abnormal exit (Restart=on-failure)

### Stop Server
```bash
kill $(cat ./agent-devs/test-project/server.pid)
```

### Rollback
- Direct source code modification is prohibited (Delegate to Deb)
- No previous binary → Stop the server and request Deb to fix it
- Register BLOCKED immediately upon build failure

### Health Check Criteria
```bash
curl -s http://localhost:8080/health
# Normal if it returns {"status":"ok","success":true}

# Additional check for simulator endpoints
curl -s "http://localhost:8080/api/v1/simulation?coin=BTC"
curl -s "http://localhost:8080/api/v1/simulation?coin=ETH"
```

### Key Log Paths
- Server Log: `./agent-devs/test-project/server.log`
- PID: `./agent-devs/test-project/server.pid`
