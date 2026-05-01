package plan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseChecklist(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantTotal int
		wantDone  int
	}{
		{
			"mixed items",
			"# Checklist\n- [ ] Item 1\n- [x] Item 2\n- [ ] Item 3\n- [X] Item 4\n",
			4, 2,
		},
		{
			"all done",
			"- [x] A\n- [x] B\n",
			2, 2,
		},
		{
			"none done",
			"- [ ] A\n- [ ] B\n- [ ] C\n",
			3, 0,
		},
		{
			"no items",
			"# Just a header\nSome text\n",
			0, 0,
		},
		{
			"items without trailing space",
			"- [ ]Item without space\n- [x]Done without space\n",
			2, 1,
		},
		{
			"with surrounding text",
			"# Header\n\n## Section\n- [ ] Real item\n- Not a checklist\n- [x] Done item\n",
			2, 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "checklist.md")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			stats, err := ParseChecklist(path)
			if err != nil {
				t.Fatalf("ParseChecklist failed: %v", err)
			}
			if stats.Total != tt.wantTotal {
				t.Errorf("Total: got %d, want %d", stats.Total, tt.wantTotal)
			}
			if stats.Completed != tt.wantDone {
				t.Errorf("Completed: got %d, want %d", stats.Completed, tt.wantDone)
			}
		})
	}
}

func TestParseChecklist_NotExist(t *testing.T) {
	stats, err := ParseChecklist("/nonexistent/path/checklist.md")
	if err != nil {
		t.Fatalf("expected nil error for nonexistent file, got: %v", err)
	}
	if stats.Total != 0 || stats.Completed != 0 {
		t.Errorf("expected 0/0 for nonexistent file, got %d/%d", stats.Total, stats.Completed)
	}
}
