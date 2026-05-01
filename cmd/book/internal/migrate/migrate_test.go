package migrate

import "testing"

func TestNormalizeProject(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"sw-go", "sw-go", "swgo"},
		{"SW Go", "SW Go", "swgo"},
		{"sw go", "sw go", "swgo"},
		{"lambda-vault", "lambda-vault", "lv"},
		{"stra-web", "stra-web", "sw"},
		{"ggf-infra", "ggf-infra", "ggf"},
		{"bold markers", "**swgo**", "swgo"},
		{"parenthetical", "ggf(GPU)", "ggf"},
		{"comma separated", "sw-go, digest", "swgo"},
		{"프로젝트 무관", "프로젝트 무관", ""},
		{"with parens 프로젝트 무관", "스킬 시스템 (프로젝트 무관)", ""},
		{"모든 프로젝트", "모든 프로젝트", ""},
		{"empty", "", ""},
		{"simple lowercase", "digest", "digest"},
		{"mixed case", "Digest", "digest"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeProject(tt.raw)
			if got != tt.want {
				t.Errorf("normalizeProject(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestExtractProject(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			"standard format",
			"## 개요\n- 관련 프로젝트: sw-go\n",
			"swgo",
		},
		{
			"bold format",
			"## 개요\n- **관련 프로젝트**: **ggf(GPU)**\n",
			"ggf",
		},
		{
			"프로젝트 무관",
			"## 개요\n- 관련 프로젝트: 프로젝트 무관\n",
			"",
		},
		{
			"no project line",
			"## 개요\n- 목적: 테스트\n",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractProject([]byte(tt.body))
			if got != tt.want {
				t.Errorf("ExtractProject() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMustAtoi(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"001", 1},
		{"074", 74},
		{"100", 100},
		{"0", 0},
	}

	for _, tt := range tests {
		got := mustAtoi(tt.input)
		if got != tt.want {
			t.Errorf("mustAtoi(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
