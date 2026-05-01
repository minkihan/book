// Package cmd provides the CLI commands for the book tool.
//
// Korean: book 도구의 CLI 커맨드를 제공한다.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"book/internal"
)

var bookRoot string

// rootCmd represents the base command when called without any subcommands.
//
// Korean: 서브커맨드 없이 호출될 때의 기본 커맨드를 나타낸다.
var rootCmd = &cobra.Command{
	Use:   "book",
	Short: "Book plan manager CLI",
	Long: `book is a CLI tool for managing book project plans.
It provides tagging, notes, SQLite indexing, and full-text search.

Korean: book은 프로젝트 플랜을 관리하는 CLI 도구이다.
태깅, 노트, SQLite 인덱싱, 전문 검색 기능을 제공한다.`,
}

// Execute adds all child commands to the root command and sets flags.
// This is called by main.main(). It only needs to happen once.
//
// Korean: 모든 자식 커맨드를 루트 커맨드에 추가하고 플래그를 설정한다.
// main.main()에서 호출되며, 한 번만 실행하면 된다.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&bookRoot, "book-root", "", "book project root directory")
}

// initConfig resolves the BookRoot path from flag, env, or auto-detection.
//
// Korean: 플래그, 환경변수, 자동 탐색 순으로 BookRoot 경로를 결정한다.
func initConfig() {
	if bookRoot != "" {
		internal.SetBookRoot(bookRoot)
		return
	}
	if env := os.Getenv("BOOK_ROOT"); env != "" {
		internal.SetBookRoot(env)
		return
	}
	// auto-detection handled by internal.BookRoot()
}
