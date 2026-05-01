package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"book/internal"
	"book/internal/plan"
	"book/internal/store"
)

var validStatuses = map[string]bool{
	"active":    true,
	"completed": true,
	"backlog":   true,
}

// statusCmd represents the status command.
//
// Korean: status 커맨드를 나타낸다.
var statusCmd = &cobra.Command{
	Use:   "status [plan-number] [status]",
	Short: "Change a plan's status",
	Long: `Change a plan's status in frontmatter and sync to SQLite.
Valid statuses: active, completed, backlog.
Setting to "completed" automatically sets the completed date.

Korean: 플랜의 frontmatter 상태를 변경하고 SQLite에 동기화한다.
유효한 상태: active, completed, backlog.
"completed"로 설정하면 완료 날짜가 자동 설정된다.`,
	Args: cobra.ExactArgs(2),
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	num, err := parseNumberArg(args[0])
	if err != nil {
		return err
	}

	newStatus := args[1]
	if !validStatuses[newStatus] {
		return fmt.Errorf("유효하지 않은 상태: %s (active/completed/backlog)", newStatus)
	}

	root, err := internal.BookRoot()
	if err != nil {
		return err
	}

	p, err := plan.FindPlan(root, num)
	if err != nil {
		return err
	}

	result, err := plan.ParseFrontmatter(p.ReadmePath())
	if err != nil {
		return err
	}

	meta := result.Meta
	if meta == nil {
		return fmt.Errorf("P%03d: frontmatter가 없습니다. 먼저 book migrate를 실행하세요", num)
	}

	oldStatus := meta.Status
	meta.Status = newStatus

	// auto-set completed date
	if newStatus == "completed" && meta.Completed == "" {
		meta.Completed = time.Now().Format("2006-01-02")
	}
	if newStatus != "completed" {
		meta.Completed = ""
	}

	if err := plan.WriteFrontmatter(p.ReadmePath(), meta); err != nil {
		return err
	}

	// sync to SQLite
	s, err := store.Open(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARN: SQLite 동기화 실패: %v\n", err)
	} else {
		if syncErr := s.IndexPlan(num); syncErr != nil {
			fmt.Fprintf(os.Stderr, "WARN: 인덱스 업데이트 실패: %v\n", syncErr)
		}
		s.Close()
	}

	fmt.Printf("P%03d: %s → %s\n", num, oldStatus, newStatus)
	return nil
}
