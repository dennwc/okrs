package okrs

import (
	"encoding/json"
	"io"
)

func init() {
	RegisterTreeWriter(TreeWriterDesc{
		Name: "json", Ext: "json",
		Write: func(w io.Writer, t TreeNode) error {
			enc := json.NewEncoder(w)
			enc.SetIndent("", "\t")
			return enc.Encode(t)
		},
	})
}
