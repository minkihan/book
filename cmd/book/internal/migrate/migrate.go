// Package migrate provides migration of existing plans to frontmatter format.
//
// Korean: 기존 플랜을 frontmatter 형식으로 마이그레이션하는 기능을 제공한다.
package migrate

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"book/internal/plan"
)

// normalizeMap maps various project name forms to their canonical names.
//
// Korean: 다양한 프로젝트명 형태를 정규화된 이름으로 매핑한다.
var normalizeMap = map[string]string{
	"sw-go":        "swgo",
	"sw go":        "swgo",
	"lambda-vault": "lv",
	"stra-web":     "sw",
	"ggf-infra":    "ggf",
	"yt-dl":        "yd",
	"my-mcp":       "mm",
}

// parenRe removes parenthetical annotations from project names.
var parenRe = regexp.MustCompile(`\s*\([^)]*\)`)

// projectLineRe matches the "관련 프로젝트:" line in various formats.
var projectLineRe = regexp.MustCompile(`^-\s*(?:\*\*)?관련\s*프로젝트(?:\*\*)?:\s*(.+)$`)

// TodoEntry holds parsed data from TODO.md for a single plan.
//
// Korean: TODO.md에서 파싱한 단일 플랜 데이터를 보관한다.
type TodoEntry struct {
	Number    int
	Status    string // active, completed, backlog
	Completed string // YYYY-MM-DD date string
}

// completedLineRe matches completed lines: "- **P041** Cloudflare Cdn Migration [2026-02-24]..."
var completedLineRe = regexp.MustCompile(`^-\s+\*\*P(\d+)\*\*\s+.+?\s+\[?(\d{4}-\d{2}-\d{2})\]?`)

// activeLineRe matches active lines: "- [P057 Sw Go Share Event]..."
var activeLineRe = regexp.MustCompile(`^\s*-\s+\[P(\d+)\s+`)

// ParseTodoMD parses TODO.md and returns a map of plan number to TodoEntry.
//
// Korean: TODO.md를 파싱하여 플랜 번호→TodoEntry 맵을 반환한다.
func ParseTodoMD(path string) (map[int]*TodoEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("TODO.md 열기 실패: %w", err)
	}
	defer f.Close()

	entries := make(map[int]*TodoEntry)
	scanner := bufio.NewScanner(f)
	section := ""

	for scanner.Scan() {
		line := scanner.Text()

		// track section
		if strings.HasPrefix(line, "## 진행 중") {
			section = "active"
			continue
		}
		if strings.HasPrefix(line, "## 백로그") {
			section = "backlog"
			continue
		}
		if strings.HasPrefix(line, "## 완료") {
			section = "completed"
			continue
		}
		if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "---") {
			if section != "completed" {
				section = ""
			}
			continue
		}

		switch section {
		case "active":
			if m := activeLineRe.FindStringSubmatch(line); m != nil {
				num := mustAtoi(m[1])
				entries[num] = &TodoEntry{Number: num, Status: "active"}
			}
		case "completed":
			if m := completedLineRe.FindStringSubmatch(line); m != nil {
				num := mustAtoi(m[1])
				entries[num] = &TodoEntry{Number: num, Status: "completed", Completed: m[2]}
			}
		}
	}

	return entries, scanner.Err()
}

// ExtractProject extracts and normalizes the project name from readme.md body.
//
// Korean: readme.md body에서 프로젝트명을 추출하고 정규화한다.
func ExtractProject(body []byte) string {
	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	for scanner.Scan() {
		line := scanner.Text()
		m := projectLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		raw := strings.TrimSpace(m[1])
		return normalizeProject(raw)
	}
	return ""
}

// normalizeProject applies normalization rules to a raw project string.
//
// Korean: 원시 프로젝트 문자열에 정규화 규칙을 적용한다.
func normalizeProject(raw string) string {
	// remove bold markers
	raw = strings.ReplaceAll(raw, "**", "")

	// skip non-project entries (check BEFORE removing parenthetical annotations)
	if raw == "" || strings.Contains(raw, "프로젝트 무관") || strings.Contains(raw, "모든 프로젝트") {
		return ""
	}

	// remove parenthetical annotations: "ggf(GPU)" → "ggf"
	raw = parenRe.ReplaceAllString(raw, "")

	// take first comma-separated value
	parts := strings.SplitN(raw, ",", 2)
	raw = strings.TrimSpace(parts[0])

	// apply normalization map
	lower := strings.ToLower(raw)
	if mapped, ok := normalizeMap[lower]; ok {
		return mapped
	}

	// keep known names as-is
	return lower
}

// MigratePlan generates frontmatter for a single plan.
// It returns nil if the plan already has frontmatter.
//
// Korean: 단일 플랜의 frontmatter를 생성한다.
// 이미 frontmatter가 있으면 nil을 반환한다.
func MigratePlan(p *plan.Plan, todoEntries map[int]*TodoEntry) (*plan.Frontmatter, error) {
	result, err := plan.ParseFrontmatter(p.ReadmePath())
	if err != nil {
		return nil, err
	}

	// skip if already has frontmatter
	if result.HasMeta {
		return nil, nil
	}

	meta := &plan.Frontmatter{
		Status:   "backlog",
		Priority: "normal",
	}

	// extract project from body
	meta.Project = ExtractProject(result.Body)

	// extract created date from git log
	meta.Created = extractCreatedDate(p.ReadmePath())

	// apply TODO.md data
	if entry, ok := todoEntries[p.Number]; ok {
		meta.Status = entry.Status
		if entry.Completed != "" {
			meta.Completed = entry.Completed
		}
	}

	return meta, nil
}

// extractCreatedDate gets the earliest commit date for a file from git log.
//
// Korean: git log에서 파일의 최초 커밋 날짜를 가져온다.
func extractCreatedDate(path string) string {
	out, err := exec.Command("git", "log", "--follow", "--format=%aI", "--", path).Output()
	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		return time.Now().Format("2006-01-02")
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	dateStr := lines[len(lines)-1]
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return time.Now().Format("2006-01-02")
	}
	return t.Format("2006-01-02")
}

func mustAtoi(s string) int {
	var n int
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}
