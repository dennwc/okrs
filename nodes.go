package okrs

import "sort"

func NewTree() *Tree {
	return &Tree{
		root: &Node{},
	}
}

type Tree struct {
	root *Node
}

func (tr *Tree) Root() *Node {
	if tr == nil {
		return nil
	}
	return tr.root
}

func (tr *Tree) NewNode(nd Node) *Node {
	return &nd
}

func (tr *Tree) merge(n *Node, n2 Node) {
	if n.Title == "" {
		n.Title = n2.Title
	}
	if n.Desc == "" {
		n.Desc = n2.Desc
	}
	if n.Link.URL != "" {
		n.Link = n2.Link
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

type Link struct {
	Title string `json:"title,omitempty" yaml:"title,omitempty"`
	URL   string `json:"url,omitempty" yaml:"url,omitempty"`
}

type Node struct {
	ID       string    `json:"id,omitempty" yaml:"id,omitempty"`
	Title    string    `json:"title,omitempty" yaml:"title,omitempty"`
	Desc     string    `json:"desc,omitempty" yaml:"desc,omitempty"`
	Link     Link      `json:"url,omitempty" yaml:"url,omitempty"`
	Priority *int      `json:"priority,omitempty" yaml:"priority,omitempty"`
	Progress *Progress `json:"progress,omitempty" yaml:"progress,omitempty"`
	Sub      []*Node   `json:"sub,omitempty" yaml:"sub,omitempty"`
	Links    []Link    `json:"links,omitempty" yaml:"links,omitempty"`

	parent *Link
}

func (n *Node) isProxyNode() bool {
	return n.parent == nil && n.ID == "" && n.Title == "" && n.Desc == "" &&
		n.Link == (Link{}) && n.Priority == nil && n.Progress == nil && len(n.Links) == 0
}

func (n *Node) Sort() {
	sort.Slice(n.Sub, func(i, j int) bool {
		a, b := n.Sub[i], n.Sub[j]
		if a.Priority == nil || b.Priority == nil {
			if a.Priority != nil || b.Priority != nil {
				return b.Priority == nil
			}
		} else if p1, p2 := *a.Priority, *b.Priority; p1 != p2 {
			return p1 < p2
		}
		return a.Title < b.Title
	})
}

func (n *Node) GetProgress() Progress {
	if n.Progress != nil && *n.Progress != (Progress{}) {
		return *n.Progress
	}
	total := len(n.Sub)
	done := 0
	for _, sub := range n.Sub {
		if sub == n {
			panic("self-reference")
		}
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
		if n2 == nil {
			panic("nil node")
		} else if n2 == n {
			panic("self-reference")
		}
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
