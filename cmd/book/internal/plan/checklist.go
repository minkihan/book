package plan

import (
	"bufio"
	"os"
	"strings"
)

// ChecklistStats holds the count of total and completed checklist items.
//
// Korean: 체크리스트 전체 항목 수와 완료 항목 수를 보관한다.
type ChecklistStats struct {
	Total     int
	Completed int
}

// ParseChecklist reads a checklist.md file and counts items.
// Checklist items are lines matching "- [ ]" (unchecked) or "- [x]" (checked).
//
// Korean: checklist.md 파일을 읽어 항목 수를 센다.
// "- [ ]"(미완료) 또는 "- [x]"(완료) 형식의 라인을 카운트한다.
func ParseChecklist(path string) (*ChecklistStats, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ChecklistStats{}, nil
		}
		return nil, err
	}
	defer f.Close()

	stats := &ChecklistStats{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "- [ ]") || strings.HasPrefix(line, "- [x]") || strings.HasPrefix(line, "- [X]") {
			stats.Total++
			if strings.HasPrefix(line, "- [x]") || strings.HasPrefix(line, "- [X]") {
				stats.Completed++
			}
		}
	}
	return stats, scanner.Err()
}
