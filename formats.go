package okrs

import (
	"io"
	"sort"
)

type TreeWriterDesc struct {
	Name  string
	Ext   string
	Write func(w io.Writer, t *TreeNode) error
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
