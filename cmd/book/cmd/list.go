package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"book/internal/store"
)

var (
	filterTag        string
	filterStatus     string
	filterProject    string
	listAll          bool
	listUndocumented bool
)

// listCmd represents the list command.
//
// Korean: list 커맨드를 나타낸다.
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List plans with optional filters",
	Long: `List plans from the index. By default shows active and backlog plans.
Use --all to include archived and completed plans.

Korean: 인덱스에서 플랜을 목록 조회한다. 기본적으로 active/backlog만 표시.
--all로 아카이브 및 완료된 플랜도 포함한다.`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringVar(&filterTag, "tag", "", "filter by tag")
	listCmd.Flags().StringVar(&filterStatus, "status", "", "filter by status (active/completed/backlog)")
	listCmd.Flags().StringVar(&filterProject, "project", "", "filter by project")
	listCmd.Flags().BoolVar(&listAll, "all", false, "include archived and completed plans")
	listCmd.Flags().BoolVar(&listUndocumented, "undocumented", false, "show undocumented completed plans")
}

func runList(cmd *cobra.Command, args []string) error {
	// --undocumented와 --status 상호 배타 검증
	if listUndocumented && filterStatus != "" {
		return fmt.Errorf("--undocumented와 --status는 함께 사용할 수 없습니다")
	}

	s, err := openStoreWithAutoIndex()
	if err != nil {
		return err
	}
	defer s.Close()

	plans, err := s.ListPlans(store.ListFilter{
		Status:       filterStatus,
		Tag:          filterTag,
		Project:      filterProject,
		IncludeAll:   listAll,
		Undocumented: listUndocumented,
	})
	if err != nil {
		return err
	}

	if len(plans) == 0 {
		fmt.Println("조건에 맞는 플랜이 없습니다.")
		return nil
	}

	// table header
	fmt.Printf("%-9s %-9s %-8s %-40s %s\n", "번호", "상태", "프로젝트", "제목", "태그")
	fmt.Println(strings.Repeat("-", 93))

	for _, p := range plans {
		tags := ""
		if len(p.Tags) > 0 {
			tags = "[" + strings.Join(p.Tags, ", ") + "]"
		}

		title := p.Title
		if len(title) > 38 {
			title = title[:35] + "..."
		}

		check := ""
		if p.CheckTotal > 0 {
			check = fmt.Sprintf(" (%d/%d)", p.CheckDone, p.CheckTotal)
		}

		hfInfo := ""
		if p.HotfixCount > 0 {
			hfInfo = fmt.Sprintf(" [HF:%d]", p.HotfixCount)
		}

		docInfo := ""
		if !listUndocumented && p.Status == "completed" && !p.Documented {
			docInfo = " [미문서화]"
		}

		label := fmt.Sprintf("P%03d", p.Number)

		fmt.Printf("%-9s %-9s %-8s %-40s %s%s%s%s\n",
			label, p.Status, p.Project, title, tags, check, hfInfo, docInfo)
	}

	fmt.Printf("\n총 %d개 플랜\n", len(plans))
	return nil
}
