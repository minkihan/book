package plan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseNotes(t *testing.T) {
	content := `# P041 Notes

---

### 2026-02-24 21:30

Cloudflare Workers 배포 완료, Phase 1 검증 통과

---

### 2026-02-24 15:00

R2 버킷 생성 완료

---
`

	notes, err := ParseNotes([]byte(content))
	if err != nil {
		t.Fatalf("ParseNotes failed: %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notes))
	}
	if notes[0].Timestamp != "2026-02-24 21:30" {
		t.Errorf("note[0].Timestamp: got %q", notes[0].Timestamp)
	}
	if notes[0].Content != "Cloudflare Workers 배포 완료, Phase 1 검증 통과" {
		t.Errorf("note[0].Content: got %q", notes[0].Content)
	}
	if notes[1].Timestamp != "2026-02-24 15:00" {
		t.Errorf("note[1].Timestamp: got %q", notes[1].Timestamp)
	}
}

func TestParseNotes_Empty(t *testing.T) {
	notes, err := ParseNotes([]byte("# P001 Notes\n\n---\n"))
	if err != nil {
		t.Fatalf("ParseNotes failed: %v", err)
	}
	if len(notes) != 0 {
		t.Errorf("expected 0 notes, got %d", len(notes))
	}
}

func TestAppendNote_NewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.md")

	if err := AppendNote(path, 41, "테스트 노트"); err != nil {
		t.Fatalf("AppendNote failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.HasPrefix(content, "# P041 Notes") {
		t.Error("missing header")
	}
	if !strings.Contains(content, "테스트 노트") {
		t.Error("missing note text")
	}

	// verify parseability
	notes, err := ParseNotes(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1 note, got %d", len(notes))
	}
	if notes[0].Content != "테스트 노트" {
		t.Errorf("Content: got %q, want %q", notes[0].Content, "테스트 노트")
	}
}

func TestAppendNote_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.md")

	// create initial note
	if err := AppendNote(path, 1, "첫 번째 노트"); err != nil {
		t.Fatal(err)
	}

	// append second note
	if err := AppendNote(path, 1, "두 번째 노트"); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	notes, err := ParseNotes(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notes))
	}

	// newest should be first
	if notes[0].Content != "두 번째 노트" {
		t.Errorf("note[0].Content: got %q, want %q", notes[0].Content, "두 번째 노트")
	}
	if notes[1].Content != "첫 번째 노트" {
		t.Errorf("note[1].Content: got %q, want %q", notes[1].Content, "첫 번째 노트")
	}
}
