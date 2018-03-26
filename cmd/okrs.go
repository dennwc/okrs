package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

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

func writeTree(name string, cmd *cobra.Command, tree *okrs.TreeNode) error {
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
		Use:   "tree [FILE]",
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

	GHCmd := &cobra.Command{
		Use:   "github",
		Short: "Github-related tools",
	}
	GHCmd.PersistentFlags().String("auth", "", "github auth token")
	GHCmd.PersistentFlags().String("org", "", "github org")
	Root.AddCommand(GHCmd)

	GHProjTree := &cobra.Command{
		Use:   "proj [PROJECTNAME]",
		Short: "load OKR tree from Github project",
		RunE: func(cmd *cobra.Command, args []string) error {
			gh := &okrs.Github{}
			gh.Token, _ = cmd.Flags().GetString("auth")
			org, _ := cmd.Flags().GetString("org")
			if org == "" {
				return errors.New("organization should be specified")
			}
			o := okrs.GHOrg{Name: org}
			for _, arg := range args {
				o.Projects = append(o.Projects, okrs.GHProject{Name: arg})
			}
			gh.Orgs = append(gh.Orgs, o)
			tree, err := gh.LoadTree(context.TODO())
			if err != nil {
				return err
			}
			name := org
			if len(args) == 1 {
				name += "_" + strings.Replace(args[0], " ", "_", -1)
			}
			return writeTree(name, cmd, tree)
		},
	}
	registerTreeWriterFlags(GHProjTree.Flags())
	GHCmd.AddCommand(GHProjTree)

	GHRepoTree := &cobra.Command{
		Use:   "repo [NAME]",
		Short: "load OKR tree from Github issues of a repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("expected one argument")
			}
			gh := &okrs.Github{}
			gh.Token, _ = cmd.Flags().GetString("auth")
			org, _ := cmd.Flags().GetString("org")
			rname := args[0]
			if org == "" && rname != "" {
				if i := strings.Index(rname, "/"); i > 0 {
					org, rname = rname[:i], rname[i+1:]
				}
			}
			if org == "" {
				return errors.New("organization should be specified")
			} else if rname == "" {
				return errors.New("repository should be specified")
			}
			rname = strings.TrimPrefix(rname, org+"/")
			o := okrs.GHOrg{Name: org}
			repo := okrs.GHRepo{Name: rname}
			o.Repos = append(o.Repos, repo)
			gh.Orgs = append(gh.Orgs, o)
			tree, err := gh.LoadTree(context.TODO())
			if err != nil {
				return err
			}
			name := org
			if len(args) == 1 {
				name += "_" + strings.Replace(args[0], " ", "_", -1)
			}
			return writeTree(name, cmd, tree)
		},
	}
	registerTreeWriterFlags(GHRepoTree.Flags())
	GHCmd.AddCommand(GHRepoTree)
}
