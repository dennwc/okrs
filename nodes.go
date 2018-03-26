package okrs

type TreeNode struct {
	ID       string      `json:"id,omitempty" yaml:"id,omitempty"`
	Title    string      `json:"title,omitempty" yaml:"title,omitempty"`
	Desc     string      `json:"desc,omitempty" yaml:"desc,omitempty"`
	URL      string      `json:"url,omitempty" yaml:"url,omitempty"`
	Priority *int        `json:"priority,omitempty" yaml:"priority,omitempty"`
	Progress *Progress   `json:"progress,omitempty" yaml:"progress,omitempty"`
	Sub      []*TreeNode `json:"sub,omitempty" yaml:"sub,omitempty"`

	parent string
}

type Progress struct {
	Done  int `json:"done,omitempty" yaml:"done,omitempty"`
	Total int `json:"total,omitempty" yaml:"total,omitempty"`
}
