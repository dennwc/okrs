package okrs

type TreeNode struct {
	ID    string     `json:"id,omitempty"`
	Title string     `json:"title,omitempty"`
	Desc  string     `json:"desc,omitempty"`
	Sub   []TreeNode `json:"sub,omitempty"`
}
