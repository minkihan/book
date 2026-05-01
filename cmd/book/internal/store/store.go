package store

import (
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"

	"book/internal/plan"
)

// backfillDocumented 정규식: history 파일에서 플랜 번호를 추출한다.
var (
	reHeading = regexp.MustCompile(`(?m)^#{2,}\s+P(\d{2,})\b`)
	reBold    = regexp.MustCompile(`\*\*P(\d{2,})[:：]`)
)

// Store wraps a SQLite database connection for plan indexing.
//
// Korean: 플랜 인덱싱을 위한 SQLite 데이터베이스 연결을 래핑한다.
type Store struct {
	db       *sql.DB
	bookRoot string
}

// Open opens or creates the SQLite database at bookRoot/.book.db.
//
// Korean: bookRoot/.book.db에서 SQLite 데이터베이스를 열거나 생성한다.
func Open(bookRoot string) (*Store, error) {
	dbPath := filepath.Join(bookRoot, ".book.db")
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("DB 열기 실패: %w", err)
	}

	s := &Store{db: db, bookRoot: bookRoot}
	if err := s.ensureSchema(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Close closes the database connection.
//
// Korean: 데이터베이스 연결을 닫는다.
func (s *Store) Close() error {
	return s.db.Close()
}

// ensureSchema creates tables if they don't exist and checks schema version.
//
// Korean: 테이블이 없으면 생성하고 스키마 버전을 확인한다.
func (s *Store) ensureSchema() error {
	for _, stmt := range DDL {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("스키마 생성 실패: %w", err)
		}
	}

	// check/set schema version
	var ver string
	err := s.db.QueryRow("SELECT value FROM meta WHERE key = 'schema_version'").Scan(&ver)
	if err == sql.ErrNoRows {
		_, err = s.db.Exec("INSERT INTO meta (key, value) VALUES ('schema_version', ?)",
			strconv.Itoa(SchemaVersion))
		return err
	}
	if err != nil {
		return err
	}

	dbVer, _ := strconv.Atoi(ver)
	if dbVer < SchemaVersion {
		if err := s.migrateSchema(dbVer); err != nil {
			return fmt.Errorf("스키마 마이그레이션 실패: %w", err)
		}
	}
	return nil
}

// migrateSchema performs incremental schema migrations from oldVer to SchemaVersion.
//
// Korean: oldVer에서 SchemaVersion까지 점진적 스키마 마이그레이션을 수행한다.
func (s *Store) migrateSchema(oldVer int) error {
	needsRebuild := false

	if oldVer < 6 {
		// v1~v5 → v6: single number PK — DROP+CREATE (SQLite는 PK 변경 불가)
		if _, err := s.db.Exec("DROP TABLE IF EXISTS plans_fts"); err != nil {
			return fmt.Errorf("plans_fts DROP 실패: %w", err)
		}
		if _, err := s.db.Exec("DROP INDEX IF EXISTS idx_plan_tags_tag"); err != nil {
			return fmt.Errorf("idx DROP 실패: %w", err)
		}
		if _, err := s.db.Exec("DROP TABLE IF EXISTS plan_tags"); err != nil {
			return fmt.Errorf("plan_tags DROP 실패: %w", err)
		}
		if _, err := s.db.Exec("DROP TABLE IF EXISTS plans"); err != nil {
			return fmt.Errorf("plans DROP 실패: %w", err)
		}
		for _, stmt := range DDL[1:] { // skip meta table
			if _, err := s.db.Exec(stmt); err != nil {
				return fmt.Errorf("v6 스키마 생성 실패: %w", err)
			}
		}
		needsRebuild = true
	}

	// update schema version only after successful migration
	_, err := s.db.Exec("UPDATE meta SET value = ? WHERE key = 'schema_version'",
		strconv.Itoa(SchemaVersion))
	if err != nil {
		return err
	}

	if needsRebuild {
		fmt.Fprintf(os.Stderr, "스키마 마이그레이션 완료 (v%d → v%d), 자동 RebuildAll 실행\n", oldVer, SchemaVersion)
		return s.RebuildAll()
	}
	fmt.Fprintf(os.Stderr, "스키마 마이그레이션 완료 (v%d → v%d)\n", oldVer, SchemaVersion)
	return nil
}

// columnMissing checks if a column is missing from a table using PRAGMA table_info.
//
// Korean: PRAGMA table_info를 사용하여 테이블에 컬럼이 없는지 확인한다.
func (s *Store) columnMissing(table, column string) (bool, error) {
	rows, err := s.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull int
		var dfltValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return false, nil // column exists
		}
	}
	return true, rows.Err() // column missing
}

// RebuildAll drops all data and re-indexes all plans.
//
// Korean: 모든 데이터를 삭제하고 전체 플랜을 재인덱싱한다.
func (s *Store) RebuildAll() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM plans_fts"); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM plan_tags"); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM plans"); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	plans, err := plan.ListPlanDirs(s.bookRoot)
	if err != nil {
		return err
	}

	for i := range plans {
		if err := s.indexPlan(&plans[i]); err != nil {
			fmt.Fprintf(os.Stderr, "  WARN  P%03d: %v\n", plans[i].Number, err)
		}
	}

	// backfillDocumented: history 파일 기반 재계산 (documented 컬럼 존재 시에만)
	if err := s.backfillDocumented(); err != nil {
		fmt.Fprintf(os.Stderr, "WARN: backfillDocumented 실패: %v\n", err)
	}
	return nil
}

// cachedPlan holds mtime data loaded in batch from the database.
//
// Korean: 데이터베이스에서 배치 로드한 mtime 데이터를 보관한다.
type cachedPlan struct {
	number      int
	status      string
	dirPath     string
	readmeMtime int64
	notesMtime  int64
	checkMtime  int64
	hotfixCount int
	hotfixMtime int64
	dirMtime    int64
}

// UpdateChanged re-indexes only plans whose files have changed since last index.
// Uses batch mtime query and directory mtime guard for completed plans.
//
// Korean: 마지막 인덱싱 이후 파일이 변경된 플랜만 재인덱싱한다.
// 배치 mtime 쿼리와 completed 플랜의 디렉토리 mtime 가드를 사용한다.
func (s *Store) UpdateChanged() error {
	plans, err := plan.ListPlanDirs(s.bookRoot)
	if err != nil {
		return err
	}

	// batch load all cached mtime data in a single query
	cached, err := s.loadCachedPlans()
	if err != nil {
		return err
	}

	for i := range plans {
		p := &plans[i]
		c, exists := cached[p.Number]

		// new plan — not in DB yet
		if !exists {
			if err := s.indexPlan(p); err != nil {
				fmt.Fprintf(os.Stderr, "  WARN  P%03d: %v\n", p.Number, err)
			}
			continue
		}

		// completed plan — directory mtime guard with dir_path check
		if c.status == "completed" {
			if p.DirPath != c.dirPath {
				// path changed (e.g. archive move) — force re-index
				if err := s.indexPlan(p); err != nil {
					fmt.Fprintf(os.Stderr, "  WARN  P%03d: %v\n", p.Number, err)
				}
				continue
			}
			currentDirMtime := fileMtimeNano(p.DirPath)
			if currentDirMtime == c.dirMtime && currentDirMtime != 0 {
				continue // DB dir_mtime matches current — skip
			}
			// directory mtime changed — fall through to file check
		}

		if s.isPlanFilesChanged(p, &c) {
			if err := s.indexPlan(p); err != nil {
				fmt.Fprintf(os.Stderr, "  WARN  P%03d: %v\n", p.Number, err)
			}
		}
	}
	return nil
}

// loadCachedPlans loads all plan mtime data from the database in a single query.
//
// Korean: 단일 쿼리로 데이터베이스의 모든 플랜 mtime 데이터를 로드한다.
func (s *Store) loadCachedPlans() (map[int]cachedPlan, error) {
	rows, err := s.db.Query(
		`SELECT number, status, dir_path, readme_mtime, notes_mtime, check_mtime,
			hotfix_count, hotfix_mtime, dir_mtime FROM plans`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int]cachedPlan)
	for rows.Next() {
		var c cachedPlan
		if err := rows.Scan(&c.number, &c.status, &c.dirPath,
			&c.readmeMtime, &c.notesMtime, &c.checkMtime,
			&c.hotfixCount, &c.hotfixMtime, &c.dirMtime); err != nil {
			return nil, err
		}
		result[c.number] = c
	}
	return result, rows.Err()
}

// isPlanFilesChanged checks if any of the plan's files have changed since last index.
//
// Korean: 플랜의 파일이 마지막 인덱싱 이후 변경되었는지 확인한다.
func (s *Store) isPlanFilesChanged(p *plan.Plan, c *cachedPlan) bool {
	if fileMtimeNano(p.ReadmePath()) != c.readmeMtime {
		return true
	}
	if fileMtimeNano(p.NotesPath()) != c.notesMtime {
		return true
	}
	if fileMtimeNano(p.ChecklistPath()) != c.checkMtime {
		return true
	}

	// hotfix change detection: count + max(mtime)
	hotfixes := p.HotfixFiles()
	if len(hotfixes) != c.hotfixCount {
		return true
	}
	var maxMtime int64
	for _, hf := range hotfixes {
		if mt := fileMtimeNano(hf.Readme); mt > maxMtime {
			maxMtime = mt
		}
		if hf.Checklist != "" {
			if mt := fileMtimeNano(hf.Checklist); mt > maxMtime {
				maxMtime = mt
			}
		}
	}
	if maxMtime != c.hotfixMtime {
		return true
	}

	return false
}

// indexPlan indexes a single plan into the database.
//
// Korean: 단일 플랜을 데이터베이스에 인덱싱한다.
func (s *Store) indexPlan(p *plan.Plan) error {
	// parse frontmatter
	result, err := plan.ParseFrontmatter(p.ReadmePath())
	if err != nil {
		return err
	}

	title := plan.ExtractTitle(result.Body)

	meta := result.Meta
	if meta == nil {
		meta = &plan.Frontmatter{Status: "backlog", Priority: "normal"}
	}

	// parse checklist
	checkStats, _ := plan.ParseChecklist(p.ChecklistPath())

	// read notes content
	notesContent := ""
	if data, err := os.ReadFile(p.NotesPath()); err == nil {
		notesContent = string(data)
	}

	// read readme content (body only, without frontmatter)
	readmeContent := string(result.Body)

	// concat hotfix content into readmeContent for FTS
	hotfixes := p.HotfixFiles()
	var hotfixCount int
	var hotfixMaxMtime int64
	for _, hf := range hotfixes {
		if data, err := os.ReadFile(hf.Readme); err == nil {
			readmeContent += fmt.Sprintf("\n\n--- HOTFIX: %s ---\n\n%s",
				filepath.Base(hf.Readme), string(data))
		}
		if mt := fileMtimeNano(hf.Readme); mt > hotfixMaxMtime {
			hotfixMaxMtime = mt
		}
		if hf.Checklist != "" {
			if mt := fileMtimeNano(hf.Checklist); mt > hotfixMaxMtime {
				hotfixMaxMtime = mt
			}
		}
		hotfixCount++
	}

	readmeMtime := fileMtimeNano(p.ReadmePath())
	notesMtime := fileMtimeNano(p.NotesPath())
	checkMtime := fileMtimeNano(p.ChecklistPath())
	dirMtime := fileMtimeNano(p.DirPath)

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// upsert plan
	_, err = tx.Exec(`
		INSERT INTO plans (number, slug, title, status, project, priority, created, completed,
			dir_path, is_archived, check_total, check_done,
			readme_mtime, notes_mtime, check_mtime, hotfix_count, hotfix_mtime, dir_mtime)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(number) DO UPDATE SET
			slug=excluded.slug, title=excluded.title, status=excluded.status,
			project=excluded.project, priority=excluded.priority,
			created=excluded.created, completed=excluded.completed,
			dir_path=excluded.dir_path, is_archived=excluded.is_archived,
			check_total=excluded.check_total, check_done=excluded.check_done,
			readme_mtime=excluded.readme_mtime, notes_mtime=excluded.notes_mtime,
			check_mtime=excluded.check_mtime,
			hotfix_count=excluded.hotfix_count, hotfix_mtime=excluded.hotfix_mtime,
			dir_mtime=excluded.dir_mtime`,
		p.Number, p.Slug, title, meta.Status, meta.Project, meta.Priority,
		meta.Created, meta.Completed, p.DirPath, boolToInt(p.IsArchived),
		checkStats.Total, checkStats.Completed,
		readmeMtime, notesMtime, checkMtime, hotfixCount, hotfixMaxMtime, dirMtime,
	)
	if err != nil {
		return err
	}

	// update tags
	if _, err := tx.Exec("DELETE FROM plan_tags WHERE plan_number = ?", p.Number); err != nil {
		return err
	}
	for _, tag := range meta.Tags {
		if _, err := tx.Exec("INSERT INTO plan_tags (plan_number, tag) VALUES (?, ?)", p.Number, tag); err != nil {
			return err
		}
	}

	// update FTS
	if _, err := tx.Exec("DELETE FROM plans_fts WHERE plan_number = ?", p.Number); err != nil {
		return err
	}
	_, err = tx.Exec(
		"INSERT INTO plans_fts (plan_number, title, readme_content, notes_content) VALUES (?, ?, ?, ?)",
		p.Number, title, readmeContent, notesContent,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// fileMtimeNano returns the file's modification time in nanoseconds.
// Returns 0 if the file doesn't exist or can't be stat'd.
//
// Korean: 파일의 수정 시간을 나노초 단위로 반환한다.
// 파일이 없거나 stat 실패 시 0을 반환한다.
func fileMtimeNano(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.ModTime().UnixNano()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// backfillDocumented scans all history files and marks completed plans as documented
// if they appear in any history .md file.
//
// Korean: 전체 history 파일을 스캔하여 history에 기록된 completed 플랜을 documented=1로 마킹한다.
func (s *Store) backfillDocumented() error {
	// documented 컬럼 존재 확인 (v1→v2 RebuildAll에서 미존재 시 안전)
	missing, err := s.columnMissing("plans", "documented")
	if err != nil {
		return err
	}
	if missing {
		return nil // no-op
	}

	historyPath := filepath.Join(s.bookRoot, "history")
	if _, err := os.Stat(historyPath); os.IsNotExist(err) {
		return nil // history 디렉토리 없으면 no-op
	}

	// 1. history/ 전체 스캔 → documentedSet 구축
	documentedSet := make(map[int]struct{})
	err = filepath.WalkDir(historyPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// _로 시작하는 디렉토리 제외 (_templates 등)
		if d.IsDir() && strings.HasPrefix(d.Name(), "_") {
			return filepath.SkipDir
		}
		// .md 파일만 처리
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err // 엄격 모드
		}
		content := string(data)

		for _, match := range reHeading.FindAllStringSubmatch(content, -1) {
			if num, err := strconv.Atoi(match[1]); err == nil {
				documentedSet[num] = struct{}{}
			}
		}
		for _, match := range reBold.FindAllStringSubmatch(content, -1) {
			if num, err := strconv.Atoi(match[1]); err == nil {
				documentedSet[num] = struct{}{}
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("history 스캔 실패: %w", err)
	}

	if len(documentedSet) == 0 {
		return nil
	}

	// 2. completedSet 조회
	rows, err := s.db.Query("SELECT number FROM plans WHERE status = 'completed'")
	if err != nil {
		return err
	}
	defer rows.Close()

	var toUpdate []int
	for rows.Next() {
		var num int
		if err := rows.Scan(&num); err != nil {
			return err
		}
		if _, ok := documentedSet[num]; ok {
			toUpdate = append(toUpdate, num)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if len(toUpdate) == 0 {
		return nil
	}

	// 3. 트랜잭션 내 일괄 UPDATE (400개 단위 chunking)
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	const chunkSize = 400
	for i := 0; i < len(toUpdate); i += chunkSize {
		end := i + chunkSize
		if end > len(toUpdate) {
			end = len(toUpdate)
		}
		chunk := toUpdate[i:end]

		placeholders := make([]string, len(chunk))
		args := make([]any, len(chunk))
		for j, num := range chunk {
			placeholders[j] = "?"
			args[j] = num
		}
		query := "UPDATE plans SET documented = 1 WHERE number IN (" +
			strings.Join(placeholders, ",") + ") AND documented = 0"
		if _, err := tx.Exec(query, args...); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// SetDocumented marks a plan as documented (documented=1).
// Returns nil if already documented (not an error).
//
// Korean: 플랜을 문서화 완료(documented=1)로 마킹한다.
// 이미 문서화된 경우 nil을 반환한다 (에러 아님).
func (s *Store) SetDocumented(number int) (alreadyDone bool, err error) {
	query := "UPDATE plans SET documented = 1 WHERE number = ? AND documented = 0"
	res, err := s.db.Exec(query, number)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	if affected == 0 {
		var exists bool
		err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM plans WHERE number = ?)", number).Scan(&exists)
		if err != nil {
			return false, err
		}
		if !exists {
			return false, fmt.Errorf("플랜 P%03d를 찾을 수 없음", number)
		}
		return true, nil // 이미 문서화 완료
	}
	return false, nil
}
