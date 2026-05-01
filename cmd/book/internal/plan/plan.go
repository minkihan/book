// Package plan provides plan data structures and filesystem operations.
//
// Korean: 플랜 데이터 구조체와 파일시스템 연산을 제공한다.
package plan

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"
)

var planDirRe = regexp.MustCompile(`^p(\d{3,})-(.+)$`)

// Plan represents a single book plan with metadata.
//
// Korean: 메타데이터를 포함한 단일 book 플랜을 나타낸다.
type Plan struct {
	Number     int
	Slug       string
	Title      string
	DirPath    string
	IsArchived bool
	Year       int
	Meta       *Frontmatter
}

// ReadmePath returns the path to the plan's readme.md.
//
// Korean: 플랜의 readme.md 경로를 반환한다.
func (p *Plan) ReadmePath() string {
	return filepath.Join(p.DirPath, "readme.md")
}

// NotesPath returns the path to the plan's notes.md.
//
// Korean: 플랜의 notes.md 경로를 반환한다.
func (p *Plan) NotesPath() string {
	return filepath.Join(p.DirPath, "notes.md")
}

// ChecklistPath returns the path to the plan's checklist.md.
//
// Korean: 플랜의 checklist.md 경로를 반환한다.
func (p *Plan) ChecklistPath() string {
	return filepath.Join(p.DirPath, "checklist.md")
}

// hotfixFileRe matches readme-hf{N}.md or readme-hf-{N}.md patterns.
var hotfixFileRe = regexp.MustCompile(`^readme-hf-?(\d+)\.md$`)

// HotfixFile represents a single hotfix file discovered in a plan directory.
//
// Korean: 플랜 디렉토리에서 발견된 단일 핫픽스 파일을 나타낸다.
type HotfixFile struct {
	Number    int    // HF number (1, 2, 10, ...)
	Readme    string // readme-hf{N}.md 절대 경로
	Checklist string // checklist-hf{N}.md 절대 경로 (없으면 빈 문자열)
}

// HotfixFiles scans the plan directory for hotfix files and returns them sorted by number.
//
// Korean: 플랜 디렉토리에서 핫픽스 파일을 탐색하고 번호 순으로 정렬하여 반환한다.
func (p *Plan) HotfixFiles() []HotfixFile {
	entries, err := os.ReadDir(p.DirPath)
	if err != nil {
		return nil
	}

	seen := map[int]string{} // number → first filename (for duplicate detection)
	var hotfixes []HotfixFile

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		m := hotfixFileRe.FindStringSubmatch(entry.Name())
		if m == nil {
			continue
		}
		num, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}

		// duplicate number detection
		if prev, ok := seen[num]; ok {
			fmt.Fprintf(os.Stderr, "WARN: HF-%d 중복 파일: %s, %s (첫 번째 사용)\n", num, prev, entry.Name())
			continue
		}
		seen[num] = entry.Name()

		hf := HotfixFile{
			Number: num,
			Readme: filepath.Join(p.DirPath, entry.Name()),
		}

		// check for matching checklist file
		checkName := fmt.Sprintf("checklist-hf%d.md", num)
		if _, err := os.Stat(filepath.Join(p.DirPath, checkName)); err == nil {
			hf.Checklist = filepath.Join(p.DirPath, checkName)
		}
		// also check hyphenated variant
		checkNameHyphen := fmt.Sprintf("checklist-hf-%d.md", num)
		if hf.Checklist == "" {
			if _, err := os.Stat(filepath.Join(p.DirPath, checkNameHyphen)); err == nil {
				hf.Checklist = filepath.Join(p.DirPath, checkNameHyphen)
			}
		}

		hotfixes = append(hotfixes, hf)
	}

	sort.Slice(hotfixes, func(i, j int) bool {
		return hotfixes[i].Number < hotfixes[j].Number
	})

	return hotfixes
}

// ListPlanDirs scans the book root for plan directories.
// It searches plans/ and archive/plans/ subdirectories.
//
// Korean: book root에서 플랜 디렉토리를 탐색한다.
// plans/ 및 archive/plans/ 하위 디렉토리를 검색한다.
func ListPlanDirs(bookRoot string) ([]Plan, error) {
	var plans []Plan

	// active plans (year determined from created field)
	active, err := scanPlanDir(filepath.Join(bookRoot, "plans"), false, 0)
	if err != nil {
		return nil, fmt.Errorf("plans/ 탐색 실패: %w", err)
	}
	plans = append(plans, active...)

	// archived plans (archive/plans/{year}/p{NNN}-*)
	archiveBase := filepath.Join(bookRoot, "archive", "plans")
	if entries, err := os.ReadDir(archiveBase); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			year, err := strconv.Atoi(entry.Name())
			if err != nil {
				continue // skip non-year directories
			}
			yearDir := filepath.Join(archiveBase, entry.Name())
			archived, err := scanPlanDir(yearDir, true, year)
			if err != nil {
				continue
			}
			plans = append(plans, archived...)
		}
	}

	sort.Slice(plans, func(i, j int) bool {
		return plans[i].Number < plans[j].Number
	})

	return plans, nil
}

// FindPlan finds a single plan by number from the book root.
//
// Korean: book root에서 번호로 단일 플랜을 찾는다.
func FindPlan(bookRoot string, number int) (*Plan, error) {
	plans, err := ListPlanDirs(bookRoot)
	if err != nil {
		return nil, err
	}

	for i := range plans {
		if plans[i].Number == number {
			return &plans[i], nil
		}
	}
	return nil, fmt.Errorf("플랜 P%03d를 찾을 수 없음", number)
}

// scanPlanDir scans a single directory for plan subdirectories.
//
// Korean: 단일 디렉토리에서 플랜 하위 디렉토리를 탐색한다.
func scanPlanDir(dir string, archived bool, year int) ([]Plan, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var plans []Plan
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		m := planDirRe.FindStringSubmatch(entry.Name())
		if m == nil {
			continue
		}
		num, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		dirPath := filepath.Join(dir, entry.Name())

		// only include directories that have readme.md
		readmePath := filepath.Join(dirPath, "readme.md")
		if _, err := os.Stat(readmePath); err != nil {
			continue
		}

		planYear := year
		if planYear == 0 {
			planYear = extractYearFromCreated(readmePath)
		}

		p := Plan{
			Number:     num,
			Slug:       m[2],
			DirPath:    dirPath,
			IsArchived: archived,
			Year:       planYear,
		}
		plans = append(plans, p)
	}
	return plans, nil
}

// extractYearFromCreated extracts the year from the created field in frontmatter.
// Falls back to current year if unavailable.
//
// Korean: frontmatter의 created 필드에서 연도를 추출한다.
// 사용 불가 시 현재 연도로 대체한다.
func extractYearFromCreated(readmePath string) int {
	result, err := ParseFrontmatter(readmePath)
	if err != nil || result.Meta == nil || result.Meta.Created == "" {
		return time.Now().Year()
	}
	if len(result.Meta.Created) >= 4 {
		if y, err := strconv.Atoi(result.Meta.Created[:4]); err == nil {
			return y
		}
	}
	return time.Now().Year()
}
