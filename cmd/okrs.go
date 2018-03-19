package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"

	"github.com/dennwc/okrs"
	"github.com/spf13/cobra"
)

var (
	Root = &cobra.Command{
		Use:   "okrs",
		Short: "tool for building OKR trees",
	}
)

func main() {
	if err := Root.Execute(); err != nil {
		log.Fatal(err)
	}
}

func writeTree(w io.Writer, _ *cobra.Command, tree okrs.TreeNode) error {
	// TODO: support other output formats
	enc := json.NewEncoder(w)
	enc.SetIndent("", "\t")
	return enc.Encode(tree)
}

func init() {
	MDCmd := &cobra.Command{
		Use:   "md",
		Short: "markdown-related tools",
	}
	Root.AddCommand(MDCmd)

	MDParseTree := &cobra.Command{
		Use:   "tree",
		Short: "parse markdown file ast OKR tree",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return errors.New("expected one argument")
			}
			var r io.Reader = os.Stdin
			if len(args) == 1 && args[0] != "-" {
				f, err := os.Open(args[0])
				if err != nil {
					return err
				}
				defer f.Close()
				r = f
			}
			tree, err := okrs.ParseMDTree(r)
			if err != nil {
				return err
			}
			return writeTree(os.Stdout, cmd, tree)
		},
	}
	MDCmd.AddCommand(MDParseTree)
}
