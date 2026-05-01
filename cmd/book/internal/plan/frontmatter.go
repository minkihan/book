package plan

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

var fmDelimiter = []byte("---\n")

// Frontmatter holds plan metadata stored in YAML frontmatter.
//
// Korean: YAML frontmatter에 저장된 플랜 메타데이터를 보관한다.
type Frontmatter struct {
	Tags      []string `yaml:"tags,omitempty,flow"`
	Status    string   `yaml:"status"`
	Project   string   `yaml:"project,omitempty"`
	Priority  string   `yaml:"priority,omitempty"`
	Created   string   `yaml:"created,omitempty"`
	Completed string   `yaml:"completed,omitempty"`
}

// ParseResult holds the result of parsing a file with optional frontmatter.
//
// Korean: 선택적 frontmatter가 포함된 파일 파싱 결과를 보관한다.
type ParseResult struct {
	Meta    *Frontmatter
	Body    []byte // original body bytes (everything after frontmatter)
	HasMeta bool   // true if frontmatter was found
	Raw     []byte // original full file content
}

// ParseFrontmatter reads a file and separates frontmatter from body.
// The body is preserved as-is in raw bytes.
//
// Korean: 파일을 읽어 frontmatter와 body를 분리한다.
// body는 원본 바이트 그대로 보존된다.
func ParseFrontmatter(path string) (*ParseResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("파일 읽기 실패: %w", err)
	}
	return ParseFrontmatterBytes(data)
}

// ParseFrontmatterBytes parses frontmatter from raw bytes.
//
// Korean: 원본 바이트에서 frontmatter를 파싱한다.
func ParseFrontmatterBytes(data []byte) (*ParseResult, error) {
	result := &ParseResult{Raw: data}

	// check for opening delimiter
	if !bytes.HasPrefix(data, fmDelimiter) {
		result.Body = data
		return result, nil
	}

	// find closing delimiter
	rest := data[len(fmDelimiter):]
	idx := bytes.Index(rest, fmDelimiter)
	if idx < 0 {
		result.Body = data
		return result, nil
	}

	yamlBytes := rest[:idx]
	bodyStart := len(fmDelimiter) + idx + len(fmDelimiter)

	var meta Frontmatter
	if err := yaml.Unmarshal(yamlBytes, &meta); err != nil {
		return nil, fmt.Errorf("frontmatter YAML 파싱 실패: %w", err)
	}

	result.Meta = &meta
	result.HasMeta = true
	result.Body = data[bodyStart:]
	return result, nil
}

// WriteFrontmatter replaces or inserts frontmatter in a file.
// The body bytes are preserved exactly as-is.
//
// Korean: 파일에서 frontmatter를 교체하거나 삽입한다.
// body 바이트는 원본 그대로 보존된다.
func WriteFrontmatter(path string, meta *Frontmatter) error {
	result, err := ParseFrontmatter(path)
	if err != nil {
		return err
	}
	return writeFrontmatterToFile(path, meta, result.Body)
}

// InsertFrontmatter adds frontmatter to a file that doesn't have one.
// If frontmatter already exists, it is replaced.
//
// Korean: frontmatter가 없는 파일에 frontmatter를 추가한다.
// 이미 있으면 교체한다.
func InsertFrontmatter(path string, meta *Frontmatter) error {
	return WriteFrontmatter(path, meta)
}

// SerializeFrontmatter converts a Frontmatter struct to YAML bytes.
//
// Korean: Frontmatter 구조체를 YAML 바이트로 변환한다.
func SerializeFrontmatter(meta *Frontmatter) ([]byte, error) {
	out, err := yaml.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("frontmatter YAML 직렬화 실패: %w", err)
	}
	return out, nil
}

// writeFrontmatterToFile writes frontmatter + body to a file.
//
// Korean: frontmatter + body를 파일에 쓴다.
func writeFrontmatterToFile(path string, meta *Frontmatter, body []byte) error {
	yamlBytes, err := SerializeFrontmatter(meta)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	buf.Write(fmDelimiter)
	buf.Write(yamlBytes)
	buf.Write(fmDelimiter)

	// ensure exactly one blank line between frontmatter and body
	trimmedBody := bytes.TrimLeft(body, "\n")
	if len(trimmedBody) > 0 {
		buf.WriteByte('\n')
	}
	buf.Write(trimmedBody)

	return os.WriteFile(path, buf.Bytes(), 0644)
}

// ExtractTitle reads the first "# " heading from a readme body.
//
// Korean: readme body에서 첫 번째 "# " 헤딩을 추출한다.
func ExtractTitle(body []byte) string {
	for _, line := range bytes.Split(body, []byte("\n")) {
		trimmed := bytes.TrimSpace(line)
		if bytes.HasPrefix(trimmed, []byte("# ")) {
			return string(bytes.TrimSpace(trimmed[2:]))
		}
	}
	return ""
}
