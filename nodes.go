package okrs

type TreeNode struct {
	ID    string     `json:"id,omitempty" yaml:"id,omitempty"`
	Title string     `json:"title,omitempty" yaml:"title,omitempty"`
	Desc  string     `json:"desc,omitempty" yaml:"desc,omitempty"`
	Sub   []TreeNode `json:"sub,omitempty" yaml:"sub,omitempty"`
}
