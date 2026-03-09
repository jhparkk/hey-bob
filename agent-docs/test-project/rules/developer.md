# rules/developer.md — Deb @ test-project

## Deb's Coding Rules in this Project

### Mandatory Environment Configuration
```bash
export PATH=$PATH:/usr/local/go/bin
cd /agent-devs/test-project/
```

### Code Structure
```
...
```

### Coding Conventions

### Standard Build and Restart Procedure
```bash
export PATH=$PATH:/usr/local/go/bin
cd /agent-devs/test-project/
kill $(cat server.pid) 2>/dev/null || pkill -f "./test-project" || true
sleep 1
go build -o test-project .
nohup ./test-project >> server.log 2>&1 &
echo $! > server.pid
sleep 2
curl -s http://localhost:8080/health
```

### Testing Criteria
- Build success is mandatory
- `/health` 200 response is mandatory
- curl test on modified features is mandatory
- Must pass the above 3 before reporting completion

### UI Rules
- Maintain dark theme (Background: `#1a1a2e`)
- Chart: Fixed to TradingView Lightweight Charts **v3.8.0** (Do not change version)
- Uptrend: `#26a69a`, Downtrend: `#ef5350`
