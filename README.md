# Book — AI Workflow Plan Manager

> 플랜 단위로 작업을 관리하고, SQLite FTS5 인덱스로 검색하는 CLI.
> Claude Code 같은 AI 코워크 환경에서 **장기 프로젝트의 도메인 컨텍스트를 LLM 프롬프트에 자동 주입**하기 위해 만들었다.

## 왜 만들었나

장기 프로젝트(SaaS 백엔드 5년, 200+ 플랜)에서 누적된 작업 이력·핫픽스·결정 노트를 GPT/Claude 컨텍스트에 빠르게 떠먹여주려면, 디스크에 흩어진 markdown을 매번 grep할 게 아니라 **인덱싱된 단일 진입점**이 필요했다.

이전엔:
- 태그 없음 → "auth 관련 핫픽스" 찾으려면 grep
- 상태 추적 분산 → TODO.md 수동 기록
- 노트(포스트잇) 기능 없음 → 비공식 메모 붙일 방법 없음
- grep은 상태+태그 조합 필터링 불가

`book` CLI 도입 후:
- `book search "캐싱"` → FTS5 풀텍스트, 히트 5개를 그대로 LLM 프롬프트에 복붙
- `book list --tag area:auth --status completed` → 1초 쿼리
- `book note 42 "memcached 후보 검토"` → 플랜에 포스트잇
- `book show 42` → frontmatter + 본문 + 최근 노트가 한 화면에

## 어떻게 만들어졌나

2026-02-24, **단일 플랜 (P075)** 으로 출발했다. 74개 기존 플랜 frontmatter 일괄 마이그레이션 포함. 이후 핫픽스 5개 + 후속 리팩토링 1건으로 확장.

| 시점 | 마일스톤 | 무엇 |
|---|---|---|
| 2026-02-24 | **P075** | Go CLI 신규, 74+개 기존 플랜 frontmatter 마이그레이션, FTS5 인덱서 |
| HF-1 | 플랜 태그 체계 도입 | 5축 태그(type/area/tech/scope/series) + CLAUDE.md 연동 |
| HF-3 | 핫픽스 라이프사이클 | 핫픽스 파일 인식, completed mtime 가드 |
| HF-4 | `book documented` 플래그 | 문서화 워크플로우 게이트 |
| HF-5 | 연도 구분 + 전체 태그 정리 | 누적 플랜 정렬 |
| 2026-04-15 | **P150** | 글로벌 순차 번호 전환 — year 기반 composite PK 제거. 리팩토링 (major) |

## 빠른 사용

```bash
# 빌드
cd cmd/book && go build -o ../../book .

# 일상 흐름
book list                           # active+backlog 플랜
book list --status active           # 진행 중만
book list --tag area:auth --all     # 영역별 + 완료 포함
book show 42                        # 플랜 상세 (frontmatter + 본문 + 최근 노트)
book search "캐싱"                  # FTS5 풀텍스트 (한국어 3글자+, 영어 OK)

# 노트
book note 42 "memcached 후보 검토"
book notes 42 --last 5

# 태그
book tag 42 add type:feature
book tags --prefix area:

# 상태
book status 42 completed            # → 날짜 자동 기록
book index                          # 변경된 플랜만 인덱스 갱신 (mtime 기반)
book index --full                   # 전체 재인덱싱
```

## 디렉토리 구조 (사용자가 직접 채우는 부분)

```
your-book/
├── plans/
│   └── p042-{slug}/
│       ├── readme.md       # frontmatter + 본문
│       └── checklist.md    # (선택) 구현 체크리스트
└── .book.db                # SQLite 인덱스 (gitignore)
```

`plans/p{NNN}-{slug}/readme.md`의 frontmatter:

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

## FTS5 인덱서

- 첫 실행 시 `.book.db`(SQLite)를 생성하고 `plans/p*/readme.md`를 인덱싱
- `book index` — mtime 변경된 파일만 갱신 (증분, P075-HF2 최적화)
- `book index --full` — 전체 재인덱싱
- 한국어 3글자 이상부터 매칭 (FTS5 trigram 유사 동작)

## 개발

- Go 1.26+
- `make build` / `make test`
- 단일 정적 바이너리 (CGO_ENABLED=0). macOS / Linux 모두

## 라이선스

MIT.
