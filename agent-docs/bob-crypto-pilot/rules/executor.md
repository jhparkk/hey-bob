# rules/executor.md — Q @ bob-crypto-pilot

## 이 프로젝트에서 Q의 배포 환경

### 환경 정보
- OS: Linux (WSL2)
- Go: `/usr/local/go/bin` (export PATH 필요)
- 프로젝트: `/home/jhpark/devel/workspace/bob-crypto-pilot/`
- 포트: **8080**
- 실행 방식: nohup 백그라운드

### 표준 배포 절차
```bash
export PATH=$PATH:/usr/local/go/bin
cd /home/jhpark/devel/workspace/bob-crypto-pilot/

# 1. 빌드
go build -o bob-crypto-pilot .

# 2. systemd 서비스로 재시작 (자동 시작 설정됨)
systemctl --user restart bob-crypto-pilot
sleep 2

# 3. 헬스체크
curl -s http://localhost:8080/health
```

### 서버 관리 (systemd)
```bash
systemctl --user start bob-crypto-pilot    # 시작
systemctl --user stop bob-crypto-pilot     # 중지
systemctl --user restart bob-crypto-pilot  # 재시작
systemctl --user status bob-crypto-pilot   # 상태 확인
```
- WSL2 부팅 시 자동 시작 (loginctl linger 설정됨)
- 비정상 종료 시 5초 후 자동 재시작 (Restart=on-failure)

### 서버 중지
```bash
kill $(cat /home/jhpark/devel/workspace/bob-crypto-pilot/server.pid)
```

### 롤백
- 소스 코드 수정 금지 (Deb에게 위임)
- 이전 바이너리 없음 → 서버 중지 후 Deb에게 수정 요청
- 빌드 실패 시 즉시 BLOCKED 등록

### 헬스체크 기준
```bash
curl -s http://localhost:8080/health
# {"status":"ok","success":true} 반환 시 정상

# 시뮬레이터 엔드포인트 추가 확인
curl -s "http://localhost:8080/api/v1/simulation?coin=BTC"
curl -s "http://localhost:8080/api/v1/simulation?coin=ETH"
```

### 주요 로그 경로
- 서버 로그: `/home/jhpark/devel/workspace/bob-crypto-pilot/server.log`
- PID: `/home/jhpark/devel/workspace/bob-crypto-pilot/server.pid`
