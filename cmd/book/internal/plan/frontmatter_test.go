package plan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFrontmatterBytes_WithMeta(t *testing.T) {
	input := []byte("---\ntags: [worker, credits]\nstatus: active\nproject: swgo\npriority: high\n---\n# P074: Title\n")

	result, err := ParseFrontmatterBytes(input)
	if err != nil {
		t.Fatalf("ParseFrontmatterBytes failed: %v", err)
	}
	if !result.HasMeta {
		t.Fatal("expected HasMeta=true")
	}
	if result.Meta.Status != "active" {
		t.Errorf("Status: got %q, want %q", result.Meta.Status, "active")
	}
	if result.Meta.Project != "swgo" {
		t.Errorf("Project: got %q, want %q", result.Meta.Project, "swgo")
	}
	if result.Meta.Priority != "high" {
		t.Errorf("Priority: got %q, want %q", result.Meta.Priority, "high")
	}
	if len(result.Meta.Tags) != 2 {
		t.Errorf("Tags: got %d, want 2", len(result.Meta.Tags))
	}
	if string(result.Body) != "# P074: Title\n" {
		t.Errorf("Body: got %q, want %q", string(result.Body), "# P074: Title\n")
	}
}

func TestParseFrontmatterBytes_WithoutMeta(t *testing.T) {
	input := []byte("# P074: Title\n\n## 개요\n")

	result, err := ParseFrontmatterBytes(input)
	if err != nil {
		t.Fatalf("ParseFrontmatterBytes failed: %v", err)
	}
	if result.HasMeta {
		t.Fatal("expected HasMeta=false")
	}
	if result.Meta != nil {
		t.Fatal("expected Meta=nil")
	}
	if string(result.Body) != string(input) {
		t.Errorf("Body should equal input")
	}
}

func TestParseFrontmatterBytes_IncompleteDelimiter(t *testing.T) {
	input := []byte("---\ntags: []\n# No closing delimiter\n")

	result, err := ParseFrontmatterBytes(input)
	if err != nil {
		t.Fatalf("ParseFrontmatterBytes failed: %v", err)
	}
	if result.HasMeta {
		t.Fatal("expected HasMeta=false for incomplete delimiter")
	}
}

func TestSerializeFrontmatter(t *testing.T) {
	meta := &Frontmatter{
		Tags:     []string{"worker", "credits"},
		Status:   "active",
		Project:  "swgo",
		Priority: "high",
	}

	out, err := SerializeFrontmatter(meta)
	if err != nil {
		t.Fatalf("SerializeFrontmatter failed: %v", err)
	}
	s := string(out)
	if len(s) == 0 {
		t.Fatal("empty output")
	}
}

func TestWriteFrontmatter_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "readme.md")

	// create file without frontmatter
	original := []byte("# P001: Test\n\n## 개요\n")
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}

	// write frontmatter
	meta := &Frontmatter{Status: "active", Priority: "normal"}
	if err := WriteFrontmatter(path, meta); err != nil {
		t.Fatalf("WriteFrontmatter failed: %v", err)
	}

	// read back and verify
	result, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("ParseFrontmatter failed: %v", err)
	}
	if !result.HasMeta {
		t.Fatal("expected HasMeta=true after write")
	}
	if result.Meta.Status != "active" {
		t.Errorf("Status: got %q, want %q", result.Meta.Status, "active")
	}
	// body includes the blank line between frontmatter and content after round-trip
	bodyStr := string(result.Body)
	if !strings.Contains(bodyStr, "# P001: Test") || !strings.Contains(bodyStr, "## 개요") {
		t.Errorf("Body content not preserved: got %q", bodyStr)
	}
}

func TestWriteFrontmatter_ReplaceExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "readme.md")

	// create file with existing frontmatter
	initial := []byte("---\nstatus: backlog\n---\n# P001: Test\n")
	if err := os.WriteFile(path, initial, 0644); err != nil {
		t.Fatal(err)
	}

	// replace frontmatter
	meta := &Frontmatter{Status: "completed", Completed: "2026-02-24"}
	if err := WriteFrontmatter(path, meta); err != nil {
		t.Fatalf("WriteFrontmatter failed: %v", err)
	}

	// verify
	result, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatal(err)
	}
	if result.Meta.Status != "completed" {
		t.Errorf("Status: got %q, want %q", result.Meta.Status, "completed")
	}
	if result.Meta.Completed != "2026-02-24" {
		t.Errorf("Completed: got %q, want %q", result.Meta.Completed, "2026-02-24")
	}
}

func TestWriteFrontmatter_NoDoubleBlankLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "readme.md")

	// body starting with double newline
	initial := []byte("---\nstatus: backlog\n---\n\n# P001: Test\n")
	if err := os.WriteFile(path, initial, 0644); err != nil {
		t.Fatal(err)
	}

	meta := &Frontmatter{Status: "active"}
	if err := WriteFrontmatter(path, meta); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	// should not have triple newline (which would mean double blank line)
	for i := 0; i < len(content)-2; i++ {
		if content[i] == '\n' && content[i+1] == '\n' && content[i+2] == '\n' {
			t.Errorf("found triple newline (double blank line) at position %d", i)
			break
		}
	}
}

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"normal", "# P074: 환불된 잡 크레딧 재차감\n", "P074: 환불된 잡 크레딧 재차감"},
		{"with leading newline", "\n# P074: Title\n", "P074: Title"},
		{"no heading", "## 개요\n- 목적\n", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractTitle([]byte(tt.body))
			if got != tt.want {
				t.Errorf("ExtractTitle(%q) = %q, want %q", tt.body, got, tt.want)
			}
		})
	}
}
