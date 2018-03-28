package okrs

func NewTree() *Tree {
	return &Tree{
		root:  &Node{},
		nodes: make(map[string]*Node),
		urls:  make(map[string]*Node),
	}
}

type Tree struct {
	root  *Node
	nodes map[string]*Node
	urls  map[string]*Node
	unk   []*Node
}

func (tr *Tree) NewNode(nd Node) *Node {
	if n := tr.nodes[nd.ID]; n != nil {
		tr.merge(n, nd)
		return n
	} else if n := tr.nodes[nd.URL]; n != nil {
		tr.merge(n, nd)
		return n
	}
	n := &nd
	if n.ID != "" {
		tr.nodes[n.ID] = n
		if n.URL != "" {
			tr.urls[n.URL] = n
		}
	}
	for i, un := range tr.unk {
		if (n.Title != "" && n.Title == un.Title) || (n.URL != "" && n.URL == un.URL) {
			tr.merge(n, *un)
			if n.ID != "" {
				tr.unk = append(tr.unk[:i], tr.unk[i+1:]...)
			} else {
				tr.unk[i] = n
			}
			return n
		}
	}
	if n.ID == "" {
		tr.unk = append(tr.unk, n)
	}
	return n
}

func (tr *Tree) merge(n *Node, n2 Node) {
	if n.Title == "" {
		n.Title = n2.Title
	}
	if n.Desc == "" {
		n.Desc = n2.Desc
	}
	if n.URL == "" {
		n.URL = n2.URL
		tr.urls[n.URL] = n
	}
	if n.Priority == nil {
		n.Priority = n2.Priority
	}
	if n.Progress == nil {
		n.Progress = n2.Progress
	}
	if len(n.Sub) == 0 {
		n.Sub = n2.Sub
	}
}

type Node struct {
	ID       string    `json:"id,omitempty" yaml:"id,omitempty"`
	Title    string    `json:"title,omitempty" yaml:"title,omitempty"`
	Desc     string    `json:"desc,omitempty" yaml:"desc,omitempty"`
	URL      string    `json:"url,omitempty" yaml:"url,omitempty"`
	Priority *int      `json:"priority,omitempty" yaml:"priority,omitempty"`
	Progress *Progress `json:"progress,omitempty" yaml:"progress,omitempty"`
	Sub      []*Node   `json:"sub,omitempty" yaml:"sub,omitempty"`

	parent string
}

func (n *Node) GetProgress() Progress {
	if n.Progress != nil && *n.Progress != (Progress{}) {
		return *n.Progress
	}
	total := len(n.Sub)
	done := 0
	for _, sub := range n.Sub {
		p := sub.GetProgress()
		if p.IsDone() {
			done++
		}
	}
	return Progress{Done: done, Total: total}
}

func (n *Node) AddChild(arr ...*Node) {
	if len(arr) == 0 {
		return
	}
	m := make(map[*Node]struct{}, len(n.Sub))
	for _, n2 := range n.Sub {
		m[n2] = struct{}{}
	}
	for _, n2 := range arr {
		if _, ok := m[n2]; ok {
			continue
		}
		n.Sub = append(n.Sub, n2)
		m[n2] = struct{}{}
	}
}

type Progress struct {
	Done  int `json:"done,omitempty" yaml:"done,omitempty"`
	Total int `json:"total,omitempty" yaml:"total,omitempty"`
}

func (p Progress) IsDone() bool {
	return p.Done == p.Total
}
