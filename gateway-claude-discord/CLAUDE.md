# gateway-claude-discord

##프로젝트 개요
1. 사용자에게 Claude를 원격으로 수행 할 수 있도록 discord 채널을 통해 중계
2. Claude가 실행 되는 위치는 상위 디렉토리인 /home/jhpark/hey-bob 으로 하여 해당 디렉토리의 CLAUDE.md 파일을 읽어 동작 할 수 있도록 지원한다.
3. /home/jhpark/hey-bob/CLAUDE.md 파일에는 /home/jhpark/hey-bob/workspace에 정의된 md 파일들에 대한 설명 및 ai agent가 동작 하기 위한 가이드가 수록 되어 있음.


##프로젝트 스펙
1. go 언어로 개발
2. db가 필요하다면 sqlite3 이용
3. api server 가 필요하다면 gin 이용
4. domain 모델 디렉토리 구조
```
- router  : echo, gin 등의 web framework router 생성
- db      : database connector 정의
<domain_name>
    - handler.go : service handler struct 정의 및 생성
   	- route.go : frontend와의 통신을 위한 URI 경로 및 메소드 정의
    - model.go : database access object, ORM, data 작업
    - <domain_name>.go : handle function 정의
- main.go : router 생성 및 web server 실행
```

##기능 요구사항
1. 사용자는 discord 채널을 통해 메시지를 전달
2. gateway 세션 생성 : claude 명령어를 수행하여 input output 메세지를 중계하는 세션
3. 세션이 유지중이라면 새로 세션을 생성 할 필요는 없으나, claude가 끊어 졌다면 다시 세션을 재생성 해야함
4. 세션 생성시의 실행되는 claude는 상위디렉토리  ../ (/home/jhpark/hey-bob)에서 수행 한다.
5. claude에서 interactive 한 선택 기능은 사용자에게 discord를 통해 전달하여 결정 할 수 있도록 한다.


##구현 개요

### 패키지 구조
- **main.go** : 진입점. `.env` 로드, DB·세션·Discord 핸들러 초기화, OS 시그널 대기
- **db/** : SQLite 커넥터. `sessions` 테이블 생성 및 마이그레이션
- **session/** : Claude 세션 관리 도메인
  - `model.go` : `sessions` 테이블 CRUD (channel_id ↔ claude_session_id)
  - `session.go` : `claude` CLI를 `--print --output-format json` 으로 실행, 응답 JSON 파싱
  - `handler.go` : 채널별 직렬 처리(sync.Map + Mutex), 세션 만료 시 자동 재시도
- **discord/** : Discord 봇 도메인
  - `handler.go` : discordgo 세션 생성, MESSAGE_CONTENT 인텐트 등록
  - `discord.go` : 메시지 이벤트 처리, 타이핑 인디케이터, 2000자 청크 분할 전송

### 주요 동작 흐름
1. Discord 멘션(@Bot) 또는 DM 수신
2. SQLite에서 해당 채널의 `claude_session_id` 조회
3. `/home/jhpark/hey-bob` 디렉토리에서 `claude --print` 실행 (stdin으로 메시지 전달)
4. 응답 JSON의 `session_id`를 SQLite에 저장 (대화 컨텍스트 유지)
5. 세션이 끊어진 경우 session_id 없이 재시도 → 새 세션 자동 생성
6. Claude 응답을 Discord 채널로 전송

### 특수 명령어
- `!reset` : 현재 채널의 Claude 세션 초기화 (새 대화 시작)



## 빌드/테스트 및 배포 절차

### 최초 설치 (서비스 등록)
```bash
# 1. .env 파일에 DISCORD_TOKEN 입력
cp .env.example .env
vi .env

# 2. 최초 빌드
go build -o gateway-claude-discord .

# 3. systemd 서비스 등록 및 시작
sudo cp gateway-claude-discord.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable gateway-claude-discord   # WSL 시작 시 자동 실행
sudo systemctl start gateway-claude-discord

# 4. 상태 확인
sudo systemctl status gateway-claude-discord
```

### 패치 배포 (코드 수정 후)
```bash
./build.sh
```
`build.sh`가 빌드 → 서비스 재시작을 자동으로 수행한다.

### 로그 확인
```bash
journalctl -u gateway-claude-discord -f
```

### 동작 방식
- WSL 부팅 시 systemd가 `gateway-claude-discord` 바이너리를 자동 실행
- `.env` 파일을 `EnvironmentFile`로 읽어 `DISCORD_TOKEN` 주입
- 크래시 발생 시 5초 후 자동 재시작 (`Restart=on-failure`)
- `./build.sh` 한 번으로 빌드 + 재시작이 완료되어 신규 버전 즉시 반영
