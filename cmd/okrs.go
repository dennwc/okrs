package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/dennwc/okrs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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

func registerTreeWriterFlags(flags *pflag.FlagSet) {
	flags.StringP("out", "o", "json", "output format to use")
}

func writeTree(name string, cmd *cobra.Command, tree okrs.TreeNode) error {
	format, _ := cmd.Flags().GetString("out")
	wr := okrs.TreeWriter(format)
	if wr == nil {
		return fmt.Errorf("unknown format %q", format)
	}
	var w io.Writer = os.Stdout
	if name != "" && name != "-" {
		if wr.Ext != "" {
			name += "." + wr.Ext
		}
		f, err := os.Create(name)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}
	return wr.Write(w, tree)
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
			name := ""
			if len(args) == 1 && args[0] != "-" {
				name = args[0]
				f, err := os.Open(name)
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
			return writeTree(name, cmd, tree)
		},
	}
	registerTreeWriterFlags(MDParseTree.Flags())
	MDCmd.AddCommand(MDParseTree)
}
