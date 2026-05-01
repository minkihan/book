package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"book/internal/plan"
)

// showCmd represents the show command.
//
// Korean: show 커맨드를 나타낸다.
var showCmd = &cobra.Command{
	Use:   "show [plan-number]",
	Short: "Show detailed plan information",
	Long: `Show detailed information about a plan including metadata, overview, and recent notes.

Korean: 플랜의 메타데이터, 개요, 최근 노트를 포함한 상세 정보를 표시한다.`,
	Args: cobra.ExactArgs(1),
	RunE: runShow,
}

func init() {
	rootCmd.AddCommand(showCmd)
}

func runShow(cmd *cobra.Command, args []string) error {
	num, err := parseNumberArg(args[0])
	if err != nil {
		return err
	}

	s, err := openStoreWithAutoIndex()
	if err != nil {
		return err
	}
	defer s.Close()

	p, err := s.GetPlan(num)
	if err != nil {
		return err
	}

	// header
	fmt.Printf("P%03d: %s\n", p.Number, p.Title)
	fmt.Println(strings.Repeat("=", 60))

	// metadata
	fmt.Printf("상태:     %s\n", p.Status)
	fmt.Printf("프로젝트: %s\n", p.Project)
	fmt.Printf("우선순위: %s\n", p.Priority)
	if p.Created != "" {
		fmt.Printf("생성일:   %s\n", p.Created)
	}
	if p.Completed != "" {
		fmt.Printf("완료일:   %s\n", p.Completed)
		if p.Documented {
			fmt.Println("문서화:   완료")
		} else {
			fmt.Println("문서화:   미완료")
		}
	}
	if len(p.Tags) > 0 {
		fmt.Printf("태그:     %s\n", strings.Join(p.Tags, ", "))
	}
	if p.CheckTotal > 0 {
		fmt.Printf("체크:     %d/%d 완료\n", p.CheckDone, p.CheckTotal)
	}
	if p.IsArchived {
		fmt.Println("아카이브: 예")
	}
	fmt.Printf("경로:     %s\n", p.DirPath)

	// overview (first few lines of readme body)
	fmt.Println()
	result, err := plan.ParseFrontmatter(p.DirPath + "/readme.md")
	if err == nil && len(result.Body) > 0 {
		lines := strings.Split(string(result.Body), "\n")
		fmt.Println("--- 개요 ---")
		maxLines := 15
		if len(lines) < maxLines {
			maxLines = len(lines)
		}
		for i := 0; i < maxLines; i++ {
			fmt.Println(lines[i])
		}
		if len(lines) > 15 {
			fmt.Println("  ...")
		}
	}

	// recent notes
	notesPath := p.DirPath + "/notes.md"
	if data, err := os.ReadFile(notesPath); err == nil {
		notes, _ := plan.ParseNotes(data)
		if len(notes) > 0 {
			fmt.Println()
			fmt.Println("--- 최근 노트 ---")
			max := 3
			if len(notes) < max {
				max = len(notes)
			}
			for i := 0; i < max; i++ {
				fmt.Printf("[%s] %s\n", notes[i].Timestamp, notes[i].Content)
			}
		}
	}

	// hotfix files
	planObj := &plan.Plan{DirPath: p.DirPath}
	hotfixes := planObj.HotfixFiles()
	if len(hotfixes) > 0 {
		fmt.Println()
		fmt.Println("--- 핫픽스 ---")
		for _, hf := range hotfixes {
			title := extractHotfixTitle(hf.Readme)
			suffix := ""
			if hf.Checklist != "" {
				suffix = " [+checklist]"
			}
			fmt.Printf("  HF-%d: %s%s\n", hf.Number, title, suffix)
		}
	}

	return nil
}

// extractHotfixTitle reads the first # heading from a hotfix readme file.
//
// Korean: 핫픽스 readme 파일에서 첫 번째 # 제목을 추출한다.
func extractHotfixTitle(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return "(읽기 실패)"
	}
	defer f.Close()

	// read first 2KB only (title is near the top)
	buf := make([]byte, 2048)
	n, _ := f.Read(buf)
	for _, line := range strings.Split(string(buf[:n]), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(trimmed[2:])
		}
	}
	return "(제목 없음)"
}
