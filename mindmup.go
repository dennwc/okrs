package okrs

import (
	"encoding/json"
	"io"
)

func init() {
	RegisterTreeWriter(TreeWriterDesc{
		Name: "mindmup", Ext: "mup",
		Write: func(w io.Writer, t *Node) error {
			enc := json.NewEncoder(w)
			enc.SetIndent("", "\t")
			return enc.Encode(asMindMup(t))
		},
	})
}

func asMindMup(t *Node) interface{} {
	type MupNode struct {
		ID    interface{}     `json:"id"`
		Title string          `json:"title"`
		Sub   map[int]MupNode `json:"ideas,omitempty"`
	}
	var last int
	var conv func(t *Node) MupNode
	conv = func(t *Node) MupNode {
		last++
		id := last
		n := MupNode{ID: id, Title: t.Title, Sub: make(map[int]MupNode)}
		for i, s := range t.Sub {
			n.Sub[i+1] = conv(s)
		}
		return n
	}

	root := conv(t)
	root.ID = "root"
	return struct {
		MupNode
		Vers int `json:"formatVersion"`
	}{
		MupNode: root,
		Vers:    3,
	}
}
