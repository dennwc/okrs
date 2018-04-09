package okrs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type TreeWriterDesc struct {
	Name  string
	Ext   string
	Write func(w io.Writer, t *Node) error
}

var treeWriters = make(map[string]TreeWriterDesc)

func TreeWriters() []TreeWriterDesc {
	var out []TreeWriterDesc
	for _, d := range treeWriters {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func TreeWriter(name string) *TreeWriterDesc {
	d, ok := treeWriters[name]
	if !ok {
		return nil
	}
	return &d
}

func RegisterTreeWriter(d TreeWriterDesc) {
	if _, ok := treeWriters[d.Name]; ok {
		panic(d.Name + " is already registered")
	}
	treeWriters[d.Name] = d
}

func DumpTree(name string, tr *Tree) error {
	var wr *TreeWriterDesc
	ext := filepath.Ext(name)
	for _, w := range TreeWriters() {
		if ext == "."+w.Ext {
			wr = &w
			break
		}
	}
	if wr == nil {
		return fmt.Errorf("unknown extension: %q", ext)
	}
	var w io.Writer = os.Stdout
	if name != "" && name != "-" {
		if wr.Ext != "" && !strings.HasSuffix(name, "."+wr.Ext) {
			name += "." + wr.Ext
		}
		f, err := os.Create(name)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}
	return wr.Write(w, tr.root)
}
