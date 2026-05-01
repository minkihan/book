package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"book/internal"
	"book/internal/plan"
)

const argMaxSafe = 900_000 // macOS ARG_MAX ~1MB, keep margin

// claudeCmd represents the claude command.
//
// Korean: claude 커맨드를 나타낸다.
var claudeCmd = &cobra.Command{
	Use:   "claude [plan-number]",
	Short: "Open Claude with plan context",
	Long: `Start a Claude session with the plan's readme, notes, and checklist as context.
Uses syscall.Exec to replace the current process.

Korean: 플랜의 readme, notes, checklist를 컨텍스트로 Claude 세션을 시작한다.
syscall.Exec으로 현재 프로세스를 교체한다.`,
	Args: cobra.ExactArgs(1),
	RunE: runClaude,
}

func init() {
	rootCmd.AddCommand(claudeCmd)
}

func runClaude(cmd *cobra.Command, args []string) error {
	num, err := parseNumberArg(args[0])
	if err != nil {
		return err
	}

	root, err := internal.BookRoot()
	if err != nil {
		return err
	}

	p, err := plan.FindPlan(root, num)
	if err != nil {
		return err
	}

	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude CLI를 찾을 수 없음: %w", err)
	}

	// build prompt with plan context
	var prompt strings.Builder
	prompt.WriteString(fmt.Sprintf("P%03d 플랜 컨텍스트:\n\n", num))

	// readme
	if data, err := os.ReadFile(p.ReadmePath()); err == nil {
		prompt.WriteString("=== readme.md ===\n")
		prompt.Write(data)
		prompt.WriteString("\n\n")
	}

	// checklist
	if data, err := os.ReadFile(p.ChecklistPath()); err == nil {
		prompt.WriteString("=== checklist.md ===\n")
		prompt.Write(data)
		prompt.WriteString("\n\n")
	}

	// notes
	if data, err := os.ReadFile(p.NotesPath()); err == nil {
		prompt.WriteString("=== notes.md ===\n")
		prompt.Write(data)
		prompt.WriteString("\n\n")
	}

	promptStr := prompt.String()

	// ARG_MAX check
	if len(promptStr) > argMaxSafe {
		fmt.Fprintf(os.Stderr, "경고: 프롬프트 크기 %d bytes > %d limit. readme.md만 포함합니다.\n",
			len(promptStr), argMaxSafe)
		var short strings.Builder
		short.WriteString(fmt.Sprintf("P%03d 플랜 컨텍스트:\n\n", num))
		if data, err := os.ReadFile(p.ReadmePath()); err == nil {
			short.WriteString("=== readme.md ===\n")
			short.Write(data)
		}
		promptStr = short.String()
	}

	fmt.Printf("Claude 세션 시작: P%03d %s\n", num, p.Slug)

	return syscall.Exec(claudePath, []string{"claude", "-p", promptStr}, os.Environ())
}
