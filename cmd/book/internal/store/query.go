package store

import (
	"database/sql"
	"fmt"
	"strings"

	"book/internal/plan"
)

// PlanRow represents a plan record from the database.
//
// Korean: 데이터베이스의 플랜 레코드를 나타낸다.
type PlanRow struct {
	Number      int
	Slug        string
	Title       string
	Status      string
	Project     string
	Priority    string
	Created     string
	Completed   string
	DirPath     string
	IsArchived  bool
	CheckTotal  int
	CheckDone   int
	HotfixCount int
	Documented  bool
	Tags        []string
}

// ListFilter holds filter criteria for listing plans.
//
// Korean: 플랜 목록 필터 조건을 보관한다.
type ListFilter struct {
	Status       string
	Tag          string
	Project      string
	IncludeAll   bool // include archived plans
	Undocumented bool // completed + documented=0
}

// ListPlans returns plans matching the given filter.
//
// Korean: 주어진 필터에 맞는 플랜을 반환한다.
func (s *Store) ListPlans(f ListFilter) ([]PlanRow, error) {
	where := []string{}
	args := []any{}

	if !f.IncludeAll {
		where = append(where, "is_archived = 0")
	}
	if f.Status != "" {
		where = append(where, "status = ?")
		args = append(args, f.Status)
	}
	if f.Project != "" {
		where = append(where, "project = ?")
		args = append(args, f.Project)
	}
	if f.Tag != "" {
		where = append(where, "number IN (SELECT plan_number FROM plan_tags WHERE tag = ?)")
		args = append(args, f.Tag)
	}
	if f.Undocumented {
		where = append(where, "status = 'completed' AND documented = 0")
	}

	// default: show active + backlog (not completed) unless status/undocumented is specified
	if f.Status == "" && !f.IncludeAll && !f.Undocumented {
		where = append(where, "status IN ('active', 'backlog')")
	}

	query := "SELECT number, slug, title, status, project, priority, created, completed, dir_path, is_archived, check_total, check_done, hotfix_count, documented FROM plans"
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY number DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("플랜 조회 실패: %w", err)
	}
	defer rows.Close()

	var plans []PlanRow
	for rows.Next() {
		var p PlanRow
		var archived, documented int
		var created, completed sql.NullString
		if err := rows.Scan(&p.Number, &p.Slug, &p.Title, &p.Status, &p.Project,
			&p.Priority, &created, &completed, &p.DirPath, &archived,
			&p.CheckTotal, &p.CheckDone, &p.HotfixCount, &documented); err != nil {
			return nil, err
		}
		p.IsArchived = archived == 1
		p.Documented = documented == 1
		p.Created = created.String
		p.Completed = completed.String
		plans = append(plans, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// batch load tags (N+1 → 1 query)
	keys := make([]int, len(plans))
	for i := range plans {
		keys[i] = plans[i].Number
	}
	tagMap, _ := s.loadTagsBatch(keys)
	for i := range plans {
		plans[i].Tags = tagMap[plans[i].Number]
	}

	return plans, nil
}

// GetPlan returns a single plan by number.
//
// Korean: 번호로 단일 플랜을 반환한다.
func (s *Store) GetPlan(number int) (*PlanRow, error) {
	var p PlanRow
	var archived, documented int
	var created, completed sql.NullString

	query := `SELECT number, slug, title, status, project, priority, created, completed,
		dir_path, is_archived, check_total, check_done, hotfix_count, documented
	FROM plans WHERE number = ?`

	err := s.db.QueryRow(query, number).Scan(&p.Number, &p.Slug, &p.Title,
		&p.Status, &p.Project, &p.Priority,
		&created, &completed, &p.DirPath, &archived, &p.CheckTotal, &p.CheckDone,
		&p.HotfixCount, &documented)
	if err != nil {
		return nil, fmt.Errorf("플랜 P%03d 조회 실패: %w", number, err)
	}
	p.IsArchived = archived == 1
	p.Documented = documented == 1
	p.Created = created.String
	p.Completed = completed.String
	p.Tags, _ = s.loadTags(number)
	return &p, nil
}

// SearchResult holds a search hit with a snippet.
//
// Korean: 스니펫을 포함한 검색 결과를 보관한다.
type SearchResult struct {
	Number  int
	Title   string
	Snippet string
}

// SearchPlans performs full-text search using FTS5.
//
// Korean: FTS5를 사용하여 전문 검색을 수행한다.
func (s *Store) SearchPlans(query string) ([]SearchResult, error) {
	// escape double quotes for safe FTS5 MATCH
	safeQuery := `"` + strings.ReplaceAll(query, `"`, `""`) + `"`
	rows, err := s.db.Query(
		`SELECT f.plan_number, f.title, snippet(plans_fts, 2, '>>>', '<<<', '...', 32)
		FROM plans_fts f
		JOIN plans p ON p.number = f.plan_number
		WHERE plans_fts MATCH ?
		ORDER BY rank`, safeQuery,
	)
	if err != nil {
		return nil, fmt.Errorf("검색 실패: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.Number, &r.Title, &r.Snippet); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// loadTagsBatch returns tags for multiple plan numbers in a single query.
//
// Korean: 여러 플랜 번호의 태그를 단일 쿼리로 반환한다.
func (s *Store) loadTagsBatch(keys []int) (map[int][]string, error) {
	if len(keys) == 0 {
		return make(map[int][]string), nil
	}
	placeholders := make([]string, len(keys))
	args := make([]any, len(keys))
	for i, k := range keys {
		placeholders[i] = "?"
		args[i] = k
	}
	rows, err := s.db.Query(
		"SELECT plan_number, tag FROM plan_tags WHERE plan_number IN ("+
			strings.Join(placeholders, ",")+") ORDER BY plan_number, tag",
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int][]string)
	for rows.Next() {
		var num int
		var tag string
		if err := rows.Scan(&num, &tag); err != nil {
			return nil, err
		}
		result[num] = append(result[num], tag)
	}
	return result, rows.Err()
}

// loadTags returns all tags for a given plan number.
//
// Korean: 주어진 플랜 번호의 모든 태그를 반환한다.
func (s *Store) loadTags(number int) ([]string, error) {
	rows, err := s.db.Query(
		"SELECT tag FROM plan_tags WHERE plan_number = ? ORDER BY tag", number)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

// TagCount holds a tag and its usage count.
//
// Korean: 태그와 사용 횟수를 보관한다.
type TagCount struct {
	Tag   string
	Count int
}

// TagCounts returns all tags with usage counts, optionally filtered by prefix.
//
// Korean: 모든 태그와 사용 횟수를 반환한다. 접두사 필터를 선택적으로 적용한다.
func (s *Store) TagCounts(prefix string) ([]TagCount, error) {
	query := "SELECT tag, COUNT(*) as cnt FROM plan_tags"
	var args []any
	if prefix != "" {
		query += " WHERE tag LIKE ?"
		args = append(args, prefix+"%")
	}
	query += " GROUP BY tag ORDER BY tag"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("태그 조회 실패: %w", err)
	}
	defer rows.Close()

	var result []TagCount
	for rows.Next() {
		var tc TagCount
		if err := rows.Scan(&tc.Tag, &tc.Count); err != nil {
			return nil, err
		}
		result = append(result, tc)
	}
	return result, rows.Err()
}

// TagsForPlan returns all tags for a specific plan number.
//
// Korean: 특정 플랜 번호의 모든 태그를 반환한다.
func (s *Store) TagsForPlan(number int) ([]string, error) {
	return s.loadTags(number)
}

// IndexPlan re-indexes a single plan by number.
//
// Korean: 번호로 단일 플랜을 재인덱싱한다.
func (s *Store) IndexPlan(number int) error {
	p, err := plan.FindPlan(s.bookRoot, number)
	if err != nil {
		return err
	}
	return s.indexPlan(p)
}
