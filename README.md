# Book — AI Workflow Plan Manager

> 플랜 단위로 작업을 관리하고, **SQLite FTS5 인덱스**로 검색하는 CLI.
> Claude Code 같은 AI 코워크 환경에서 **장기 프로젝트의 도메인 컨텍스트를 LLM 프롬프트에 자동 주입**하기 위해 만들었다.

---

## 목차

- [왜 만들었나](#왜-만들었나)
- [어떻게 만들어졌나](#어떻게-만들어졌나)
- [핵심 개념](#핵심-개념)
- [빠른 사용](#빠른-사용)
- [디렉토리 구조](#디렉토리-구조)
- [5축 태그 체계](#5축-태그-체계)
- [데이터 모델](#데이터-모델)
- [FTS5 검색 엔진](#fts5-검색-엔진)
- [LLM 컨텍스트 주입 패턴](#llm-컨텍스트-주입-패턴)
- [핫픽스 라이프사이클](#핫픽스-라이프사이클)
- [개발](#개발)

---

## 왜 만들었나

장기 프로젝트(SaaS 백엔드 5년, 200+ 플랜, 핫픽스 50+)에서 누적된 작업 이력·결정 노트·핫픽스를 GPT/Claude 컨텍스트에 빠르게 떠먹여주려면, 디스크에 흩어진 markdown을 매번 grep할 게 아니라 **인덱싱된 단일 진입점**이 필요했다.

이전엔:
- **태그 없음** → "auth 관련 핫픽스" 찾으려면 grep
- **상태 추적 분산** → TODO.md 수동 기록, 플랜 파일에 메타데이터 없음
- **노트 기능 없음** → 비공식 메모(포스트잇) 붙일 방법 없음
- **검색 한계** → grep으로만. 상태+태그 조합 필터링 불가
- **컨텍스트 주입** → 매번 디렉토리 트리 보면서 수동으로 LLM 프롬프트에 카피

`book` CLI 도입 후:
- `book search "캐싱"` → FTS5 풀텍스트, 히트 5개를 그대로 LLM 프롬프트에 복붙
- `book list --tag area:auth --status completed` → 1초 쿼리
- `book note 42 "memcached 후보 검토"` → 플랜에 포스트잇
- `book show 42` → frontmatter + 본문 + 최근 노트가 한 화면
- `book claude 42` → 해당 플랜 디렉토리에서 Claude Code 세션 시작 (컨텍스트 자동 주입)

---

## 어떻게 만들어졌나

2026-02-24, **단일 플랜 (P075)** 으로 출발했다. 74개 기존 플랜 frontmatter 일괄 마이그레이션 포함. 이후 핫픽스 5개 + 후속 리팩토링 1건으로 진화.

| 시점 | 마일스톤 | 무엇 |
|---|---|---|
| 2026-02-24 | **P075** | Go CLI 신규, 74+개 기존 플랜 frontmatter 마이그레이션, FTS5 trigram 인덱서 |
| HF-1 | 5축 태그 체계 도입 | type / area / tech / scope / series + CLAUDE.md 연동 |
| HF-2 | 증분 인덱싱 최적화 | mtime 기반, 변경된 파일만 재인덱싱 |
| HF-3 | 핫픽스 라이프사이클 | `readme-hf{N}.md` / `checklist-hf{N}.md` 인식, completed mtime 가드 |
| HF-4 | `book documented` 플래그 | 문서화 워크플로우 게이트 (`/문서화` 슬래시 커맨드 연동) |
| HF-5 | 연도 구분 + 전체 태그 정리 | 누적 플랜 정렬 |
| 2026-04-15 | **P150** | 글로벌 순차 번호 전환 — year 기반 composite PK 제거, 단일 PRIMARY KEY로 단순화 |

운영 통계 (2026-05 기준):
- 누적 플랜: **152개** (active+backlog+completed+archived)
- 핫픽스: **48개** (P{NNN}-HF{N} 패턴)
- 인덱스 사이즈: ~30MB (.book.db)
- 평균 쿼리: <50ms (FTS5 풀텍스트 + 5축 태그 JOIN)

---

## 핵심 개념

### 플랜 (Plan)

작업의 단위. 디렉토리 + frontmatter + 본문 + 체크리스트 + (선택) 핫픽스.

```
plans/p042-add-redis-cache/
├── readme.md              # frontmatter + 본문 (필수)
├── checklist.md           # 구현 체크리스트 (선택)
├── readme-hf1.md          # 핫픽스 1 (선택)
└── checklist-hf1.md       # 핫픽스 1 체크리스트
```

### 노트 (Note)

플랜에 붙이는 비공식 포스트잇. 플랜 본문은 결정/설계의 기록, 노트는 진행하면서 떠오르는 메모·의문·후속 아이디어 보관용. 플랜 외부 파일(`notes-{plan_number}.md`)에 저장돼 본문을 오염시키지 않음.

### 태그 (Tag)

5축 네임스페이스 태그. 단일 플랜이 여러 태그를 가짐 (`area:`는 1~2개, `tech:`는 0~2개 등 축별 권장 범위).

### 상태 (Status)

`backlog → active → completed` (세 단계). `completed` 진입 시 날짜 자동 기록. `archived` 플래그는 별도 (오래된 완료 플랜을 `archive/` 디렉토리로 이동).

### 문서화 플래그 (Documented)

`book documented {N}`로 마킹. 워크플로우 게이트 — `/문서화` 슬래시 커맨드 완료 시에만 호출. 미문서화 플랜을 `book list --undocumented`로 추적.

---

## 빠른 사용

### 빌드

```bash
cd cmd/book && go build -o ../../book .
```

또는 Makefile:

```bash
make build
```

### 일상 흐름

```bash
# 목록
book list                              # active+backlog (기본)
book list --status active              # 진행 중만
book list --tag area:auth --all        # 영역별 + 완료/아카이브 포함
book list --undocumented               # 문서화 안 된 플랜만
book list --tag series:go-migration    # 시리즈별

# 상세
book show 42                           # frontmatter + 본문 + 최근 노트 5개

# 검색 (FTS5)
book search "캐싱"                     # 한국어 3글자+, 영어 OK
book search "redis pub sub"            # 다중 토큰
book search "캐싱" --status completed  # 결합 필터

# 노트
book note 42 "memcached 후보 검토"     # 추가
book notes 42                          # 전체 조회
book notes 42 --last 5                 # 최근 5개

# 태그
book tag 42 add type:feature
book tag 42 remove area:legacy
book tags                              # 전체 어휘
book tags --prefix area:               # 축별
book tags --plan 42                    # 특정 플랜의 태그

# 상태
book status 42 active
book status 42 completed               # → completed: "2026-05-02" 자동 기록

# 인덱스
book index                             # mtime 변경된 플랜만 (증분)
book index --full                      # 전체 재인덱싱

# 문서화 플래그
book documented 42                     # 마킹
book list --undocumented               # 미문서화 플랜 조회

# Claude Code 세션 (해당 플랜 디렉토리에서 시작)
book claude 42
```

---

## 디렉토리 구조

```
your-book/
├── plans/
│   ├── p042-add-redis-cache/
│   │   ├── readme.md
│   │   └── checklist.md
│   └── p075-book-cli-tool/
│       ├── readme.md
│       ├── checklist.md
│       ├── readme-hf1.md
│       └── checklist-hf1.md
├── archive/                  # 완료/오래된 플랜 (선택)
│   └── plans/2026/...
└── .book.db                  # SQLite 인덱스 (gitignore)
```

플랜 frontmatter:

```yaml
---
tags: ['type:feature', 'area:auth', 'tech:go', 'scope:minor', 'series:auth-rewrite']
status: active
priority: normal              # high / normal / low
created: "2026-01-15"
completed: "2026-02-03"       # status=completed 진입 시 자동 추가
project: backend              # 자유 문자열, list 출력에 표시
---

# P042: Redis 캐싱 도입

## 개요
- 목적: ...
```

---

## 5축 태그 체계

```
{축}:{값}
```

| 축 | 접두사 | 권장 개수 | 예 |
|---|---|---|---|
| **작업 유형** | `type:` | 1 | feature, bugfix, perf, refactor, infra, migration, tool, hotfix |
| **비즈니스 영역** | `area:` | 1~2 | api, worker, billing, auth, cdn, db, devtool, media, logging |
| **기술 스택** | `tech:` | 0~2 | go, rust, python, mongodb, redis, s3, sqlite, cloudflare |
| **영향 범위** | `scope:` | 1 | patch (1~5 항목) / minor (6~20) / major (21+) / cross (다중 프로젝트) |
| **시리즈** | `series:` | 0~2 | go-migration, billing, auth-rewrite (선택) |

축 접두사는 `book tags --prefix area:` 같은 빠른 어휘 조회를 가능하게 하고, `series:`로 다중 플랜을 묶어 history를 끌어올린다.

---

## 데이터 모델

스키마 v6 기준. 핵심 테이블:

```sql
-- 플랜 메타
CREATE TABLE plans (
    number       INTEGER PRIMARY KEY,
    slug         TEXT NOT NULL,
    title        TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'backlog',
    project      TEXT NOT NULL DEFAULT '',
    priority     TEXT NOT NULL DEFAULT 'normal',
    created      TEXT,
    completed    TEXT,
    dir_path     TEXT NOT NULL,
    is_archived  INTEGER NOT NULL DEFAULT 0,
    check_total  INTEGER NOT NULL DEFAULT 0,
    check_done   INTEGER NOT NULL DEFAULT 0,
    readme_mtime INTEGER NOT NULL DEFAULT 0,
    notes_mtime  INTEGER NOT NULL DEFAULT 0,
    check_mtime  INTEGER NOT NULL DEFAULT 0,
    hotfix_count INTEGER NOT NULL DEFAULT 0,
    hotfix_mtime INTEGER NOT NULL DEFAULT 0,
    dir_mtime    INTEGER NOT NULL DEFAULT 0,
    documented   INTEGER NOT NULL DEFAULT 0
);

-- 다대다 태그
CREATE TABLE plan_tags (
    plan_number INTEGER NOT NULL,
    tag         TEXT NOT NULL,
    PRIMARY KEY (plan_number, tag)
);
CREATE INDEX idx_plan_tags_tag ON plan_tags(tag);

-- FTS5 풀텍스트 인덱스 (trigram 토크나이저)
CREATE VIRTUAL TABLE plans_fts USING fts5(
    plan_number UNINDEXED,
    title,
    readme_content,
    notes_content,
    tokenize='trigram'
);
```

각 *_mtime 컬럼이 증분 인덱싱의 핵심 — 파일시스템 mtime을 비교해 변경된 플랜만 재인덱싱한다 (P075-HF2).

---

## FTS5 검색 엔진

### Trigram 토크나이저 선택 이유

`tokenize='trigram'` — 한국어/혼합 문자열에 강하다. 일반 `unicode61` 토크나이저는 한국어 어절을 통째로 한 토큰으로 처리해 부분 매칭이 약함. trigram은 3글자 단위로 잘라 인덱싱하므로 부분 매칭 OK (한국어 3글자+ 권장).

### 쿼리 결합

```bash
book search "캐싱" --status completed --tag tech:redis
```

내부적으로:
1. `plans_fts MATCH 'redis cache'` — 풀텍스트 1차 후보
2. `JOIN plans ON plans_fts.plan_number = plans.number` — 메타 결합
3. `JOIN plan_tags ON ...` — 태그 필터
4. `WHERE plans.status = 'completed'` — 상태 필터

### 출력 포맷

```
'캐싱' 검색 결과: 3건
----------------------------------------------------------------------
P042  P042: Redis 캐싱 도입
      ...>>>캐싱<<< 도입 — TTL 30초 / LRU eviction...
P088  P088: API 응답 캐싱 레이어 추가
      ...HTTP >>>캐싱<<< 헤더 표준화...
```

매치 토큰을 `>>>...<<<`로 강조 (FTS5 `snippet()` 함수 활용). 그대로 LLM 프롬프트에 붙여넣어도 검색 컨텍스트가 살아남음.

---

## LLM 컨텍스트 주입 패턴

book CLI는 단독 도구가 아니라 **Claude Code 슬래시 커맨드 시스템과 결합해 동작**한다.

### 패턴 1: 새 플랜 검토 시

```
사용자: /review 042
  ↓
슬래시 커맨드: book show 42 → frontmatter + 본문 추출
  ↓ + 관련 시리즈 자동 첨부
  book list --tag series:auth-rewrite --all
  book list --tag area:auth --status completed --all (최근 5개)
  ↓
Claude 컨텍스트에 자동 주입
```

### 패턴 2: 유사 작업 탐색

```
사용자: "이전에 비슷한 캐싱 작업 했나?"
  ↓
어시스턴트: book search "캐싱" → 3건 즉시 추출
            book list --tag area:cache --all → 5건
            book show {각각} → 본문 일부
  ↓
"P042에서 Redis TTL 30초로 시도, P088에서 HTTP 캐싱 헤더 표준화..." 같은 컨텍스트 회상
```

### 패턴 3: 핫픽스 흐름

```
deploy → 운영 이슈 발견
  ↓
book note 42 "P42 deploy 후 5xx 증가, 캐시 무효화 누락 가능성"
  ↓
/debug → book show 42 + book notes 42 자동 추출 → 진단
  ↓
HF-1 작성 → book index → 다음 검색에 즉시 반영
```

---

## 핫픽스 라이프사이클

book은 핫픽스를 **별도 파일로 분리**해서 관리한다 (P075-HF3 결과).

```
plans/p042-add-redis-cache/
├── readme.md              # 본문
├── checklist.md           # 본문 체크리스트
├── readme-hf1.md          # 핫픽스 1
├── checklist-hf1.md       # 핫픽스 1 체크리스트
├── readme-hf2.md          # 핫픽스 2 (선택)
└── checklist-hf2.md
```

`readme.md`의 `## 핫픽스 이력` 섹션에 한 줄 링크만 남기고, 디테일은 별도 파일.

```markdown
## 핫픽스 이력

- **HF-1**: TTL 누락 수정 (2026-03-15) → [readme-hf1.md](readme-hf1.md)
- **HF-2 (미구현)**: 캐시 워밍 → [readme-hf2.md](readme-hf2.md)
```

`book show 42`는 본문 + 핫픽스 목록을 함께 출력. `book list`의 `[HF:5]` 라벨은 핫픽스 5건이 달린 플랜.

---

## 개발

- **Go 1.26+**
- **modernc.org/sqlite** — 순수 Go SQLite (CGO 없음). 정적 빌드, 크로스 컴파일 자유.
- **spf13/cobra** — CLI 프레임워크
- **gopkg.in/yaml.v3** — frontmatter 파싱

```bash
make build       # 단일 정적 바이너리
make test        # plan/, migrate/ 패키지 테스트
```

테스트 커버리지: `internal/plan/` (frontmatter, checklist, notes 파싱), `internal/migrate/` (스키마 마이그레이션).

---

## 라이선스

MIT.
