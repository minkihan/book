package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// searchCmd represents the search command.
//
// Korean: search 커맨드를 나타낸다.
var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Full-text search plans",
	Long: `Search plan titles, readmes, and notes using FTS5 trigram index.
Supports Korean substring matching.

Korean: FTS5 trigram 인덱스를 사용하여 플랜 제목, readme, 노트를 검색한다.
한국어 부분 문자열 매칭을 지원한다.`,
	Args: cobra.ExactArgs(1),
	RunE: runSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := args[0]

	s, err := openStoreWithAutoIndex()
	if err != nil {
		return err
	}
	defer s.Close()

	results, err := s.SearchPlans(query)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		fmt.Printf("'%s' 검색 결과 없음\n", query)
		return nil
	}

	fmt.Printf("'%s' 검색 결과: %d건\n", query, len(results))
	fmt.Println(strings.Repeat("-", 70))

	for _, r := range results {
		fmt.Printf("P%03d  %s\n", r.Number, r.Title)
		if r.Snippet != "" {
			// clean up snippet
			snippet := strings.ReplaceAll(r.Snippet, "\n", " ")
			if len(snippet) > 120 {
				snippet = snippet[:117] + "..."
			}
			fmt.Printf("      %s\n", snippet)
		}
	}

	return nil
}
