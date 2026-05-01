package plan

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

var noteHeaderRe = regexp.MustCompile(`^###\s+(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2})`)

// Note represents a single timestamped note entry.
//
// Korean: 타임스탬프가 포함된 단일 노트 항목을 나타낸다.
type Note struct {
	Timestamp string
	Content   string
}

// ParseNotes parses notes.md content into individual notes.
// Notes are ordered newest-first (as they appear in the file).
//
// Korean: notes.md 내용을 개별 노트로 파싱한다.
// 노트는 최신순(파일에 나타나는 순서)으로 정렬된다.
func ParseNotes(data []byte) ([]Note, error) {
	lines := strings.Split(string(data), "\n")
	var notes []Note
	var current *Note

	for _, line := range lines {
		if m := noteHeaderRe.FindStringSubmatch(line); m != nil {
			if current != nil {
				current.Content = strings.TrimSpace(current.Content)
				notes = append(notes, *current)
			}
			current = &Note{Timestamp: m[1]}
			continue
		}
		if current != nil {
			trimmed := strings.TrimSpace(line)
			if trimmed == "---" {
				continue // skip visual separators
			}
			if current.Content == "" {
				current.Content = trimmed
			} else if trimmed != "" {
				current.Content += "\n" + trimmed
			}
		}
	}
	if current != nil {
		current.Content = strings.TrimSpace(current.Content)
		notes = append(notes, *current)
	}

	return notes, nil
}

// AppendNote adds a new note entry to a notes.md file.
// Creates the file if it doesn't exist.
//
// Korean: notes.md 파일에 새 노트 항목을 추가한다.
// 파일이 없으면 생성한다.
func AppendNote(path string, planNumber int, text string) error {
	now := time.Now().Format("2006-01-02 15:04")

	entry := fmt.Sprintf("### %s\n\n%s\n\n---\n", now, text)

	// read existing content
	existing, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("notes.md 읽기 실패: %w", err)
		}
		// create new file with header
		content := fmt.Sprintf("# P%03d Notes\n\n---\n\n%s", planNumber, entry)
		return os.WriteFile(path, []byte(content), 0644)
	}

	// insert new note after the first "---" separator (after title)
	parts := strings.SplitN(string(existing), "\n---\n", 2)
	if len(parts) == 2 {
		content := parts[0] + "\n---\n\n" + entry + parts[1]
		return os.WriteFile(path, []byte(content), 0644)
	}

	// fallback: append at end
	content := string(existing) + "\n" + entry
	return os.WriteFile(path, []byte(content), 0644)
}
