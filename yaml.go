package okrs

import (
	"io"

	"gopkg.in/yaml.v2"
)

func init() {
	RegisterTreeWriter(TreeWriterDesc{
		Name: "yaml", Ext: "yml",
		Write: func(w io.Writer, t *TreeNode) error {
			enc := yaml.NewEncoder(w)
			return enc.Encode(t)
		},
	})
}
