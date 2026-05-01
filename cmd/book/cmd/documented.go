package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"book/internal"
	"book/internal/store"
)

// documentedCmd represents the documented command.
//
// Korean: documented 커맨드를 나타낸다.
var documentedCmd = &cobra.Command{
	Use:   "documented [plan-number]",
	Short: "Mark a plan as documented",
	Long: `Mark a plan as documented (documented=1).
This is called by the /문서화 skill after committing history.

Korean: 플랜을 문서화 완료(documented=1)로 마킹한다.
/문서화 스킬에서 history 커밋 후 호출한다.`,
	Args: cobra.ExactArgs(1),
	RunE: runDocumented,
}

func init() {
	rootCmd.AddCommand(documentedCmd)
}

func runDocumented(cmd *cobra.Command, args []string) error {
	num, err := parseNumberArg(args[0])
	if err != nil {
		return err
	}

	root, err := internal.BookRoot()
	if err != nil {
		return err
	}

	s, err := store.Open(root)
	if err != nil {
		return err
	}
	defer s.Close()

	alreadyDone, err := s.SetDocumented(num)
	if err != nil {
		return err
	}

	if alreadyDone {
		fmt.Printf("P%03d: 이미 문서화 완료\n", num)
	} else {
		fmt.Printf("P%03d: 문서화 완료로 마킹\n", num)
	}
	return nil
}
