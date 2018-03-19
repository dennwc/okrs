package okrs

import (
	"encoding/json"
	"io"
)

func init() {
	RegisterTreeWriter(TreeWriterDesc{
		Name: "mindmup", Ext: "mup",
		Write: func(w io.Writer, t TreeNode) error {
			enc := json.NewEncoder(w)
			enc.SetIndent("", "\t")
			return enc.Encode(asMindMup(t))
		},
	})
}

func asMindMup(t TreeNode) interface{} {
	type Node struct {
		ID    interface{}  `json:"id"`
		Title string       `json:"title"`
		Sub   map[int]Node `json:"ideas,omitempty"`
	}
	var last int
	var conv func(t TreeNode) Node
	conv = func(t TreeNode) Node {
		last++
		id := last
		n := Node{ID: id, Title: t.Title, Sub: make(map[int]Node)}
		for i, s := range t.Sub {
			n.Sub[i+1] = conv(s)
		}
		return n
	}

	root := conv(t)
	root.ID = "root"
	return struct {
		Node
		Vers int `json:"formatVersion"`
	}{
		Node: root,
		Vers: 3,
	}
}
