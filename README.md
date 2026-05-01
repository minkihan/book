# Book — AI Workflow Plan Manager

> 플랜 단위로 작업을 관리하고, SQLite FTS5 인덱스로 검색하는 CLI. Claude Code 같은 AI 코워크 환경에서 컨텍스트 자동 주입용.

## 왜 만들었나

- 장기 프로젝트에서 누적된 플랜·작업 이력을 GPT/Claude 컨텍스트에 빠르게 떠먹여주려면, 디스크에 흩어진 markdown을 매번 다시 grep할 게 아니라 **인덱싱된 단일 진입점**이 필요했다.
- `book search` 한 줄로 FTS5 풀텍스트 검색 → 히트 5개를 곧바로 LLM 프롬프트에 붙여넣는 흐름.
- 5축 태그(type / area / tech / scope / series)로 도메인을 좁혀 "관련 작업만" 추출.

## 설치

```bash
cd cmd/book
go build -o ../../book .
./book --help
```

또는 Makefile:

```bash
make build
```

## 빠른 사용

```bash
book list                         # active+backlog 플랜 목록
book list --status active         # 진행 중만
book list --tag area:auth --all   # 영역별 필터

book show 42                      # 플랜 상세 (frontmatter + 본문 + 최근 노트)
book search "캐싱"                # FTS5 풀텍스트 검색 (한국어 3글자+, 영어 OK)

book note 42 "memcached 후보 검토"  # 노트 추가
book notes 42 --last 5             # 최근 노트 조회

book tag 42 add type:feature       # 태그 부여
book tags --prefix area:           # 사용 중인 태그 어휘

book status 42 completed           # 상태 전환 (completed → 날짜 자동)
book index                         # 변경된 플랜만 인덱스 갱신
book index --full                  # 전체 재인덱싱
```

## 디렉토리 구조 (예시)

```
your-book/
├── plans/
│   └── p042-{slug}/
│       ├── readme.md      # frontmatter + 본문
│       └── checklist.md   # (선택) 구현 체크리스트
└── .book.db               # SQLite 인덱스 (gitignore)
```

`plans/p{NNN}-{slug}/readme.md`의 frontmatter 예시:

```yaml
---
tags: ['type:feature', 'area:auth', 'tech:go', 'scope:minor']
status: active
priority: normal
created: "2026-01-15"
---
```

## 5축 태그 체계

| 축 | 접두사 | 예 |
|---|---|---|
| 작업 유형 | `type:` | feature, bugfix, perf, refactor, infra, migration, tool, hotfix |
| 비즈니스 영역 | `area:` | api, worker, billing, auth, cdn, db, devtool 등 |
| 기술 스택 | `tech:` | go, rust, python, mongodb, redis, s3 등 |
| 영향 범위 | `scope:` | patch / minor / major / cross |
| 시리즈 | `series:` | go-migration, billing 등 (선택) |

## FTS5 인덱스

- 첫 실행 시 `.book.db`를 SQLite로 생성하고 `plans/p*/readme.md`를 모두 인덱싱.
- `book index` — mtime 변경된 파일만 갱신 (증분).
- `book index --full` — 전체 재인덱싱.
- 검색 토큰화는 한국어 3글자 이상부터 매칭 (FTS5 trigram 유사 동작).

## 개발

- Go 1.26+
- `make build` / `make test`
- 단일 정적 바이너리 (CGO_ENABLED=0). macOS / Linux 모두.

## 라이선스

MIT.
