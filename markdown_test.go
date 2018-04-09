package okrs

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func done() *Progress {
	return &Progress{Done: 1, Total: 1}
}

func pri(v int) *int {
	return &v
}

var casesMDTree = []struct {
	name string
	md   string
	exp  *Node
}{
	{
		name: "basic",
		md: `# Root
Some description

## Level 2:

Description 2

* sub 1
  * sub 1.1
  * sub 1.2
* sub 2
  * sub 2.1
  * sub 2.2

### Level 3:

Description 3

* sub 3.1
* sub 3.2

## Level 2.2

##### Level 5

## Level 2.3

- [ ] sub 2.3.1
- [x] sub 2.3.2
	- [x] sub 2.3.2.1
`,
		exp: &Node{
			Title: "Root",
			Desc:  "Some description",
			Sub: []*Node{
				{
					Title: "Level 2",
					Desc:  "Description 2",
					Sub: []*Node{
						{
							Title: "sub 1",
							Sub: []*Node{
								{Title: "sub 1.1"},
								{Title: "sub 1.2"},
							},
						},
						{
							Title: "sub 2",
							Sub: []*Node{
								{Title: "sub 2.1"},
								{Title: "sub 2.2"},
							},
						},
						{
							Title: "Level 3",
							Desc:  "Description 3",
							Sub: []*Node{
								{Title: "sub 3.1"},
								{Title: "sub 3.2"},
							},
						},
					},
				},
				{
					Title: "Level 2.2",
					Sub: []*Node{
						{Sub: []*Node{
							{Sub: []*Node{
								{Title: "Level 5"},
							}},
						}},
					},
				},
				{
					Title: "Level 2.3",
					Sub: []*Node{
						{Title: "sub 2.3.1"},
						{Title: "sub 2.3.2", Progress: done(), Sub: []*Node{
							{Title: "sub 2.3.2.1", Progress: done()},
						}},
					},
				},
			},
		},
	},
	{
		name: "okr1",
		md: `**Parent objective:** #12 
**Progress:** 0%

- [ ] [P0] Higher-level APIs (functions, classes, etc). #21
- [x] [P1] Higher-level categories for nodes. #23
- [X] [P2] Structural pointers.
`,
		exp: &Node{
			parent: &Link{"#12", "#12"},
			Sub: []*Node{
				{Title: `Higher-level APIs (functions, classes, etc).`, Link: Link{"#21", "#21"}, Priority: pri(0)},
				{Title: `Higher-level categories for nodes.`, Link: Link{"#23", "#23"}, Priority: pri(1), Progress: done()},
				{Title: `Structural pointers.`, Priority: pri(2), Progress: done()},
			},
		},
	},
	{
		name: "okr2",
		md: `**Parent objective:** [#12](http://github.com)
**Progress:** 2/3
`,
		exp: &Node{
			Progress: &Progress{Done: 2, Total: 3},
			parent:   &Link{"#12", "http://github.com"},
		},
	},
	{
		name: "okr3",
		md:   "**Parent objective:**  #5\r\n**Progress:** 0%\r\n\r\n- [ ] [P1] Optimize stuff #27",
		exp: &Node{
			parent: &Link{"#5", "#5"},
			Sub: []*Node{
				{Title: "Optimize stuff", Link: Link{"#27", "#27"}, Priority: pri(1)},
			},
		},
	},
}

func TestMDTree(t *testing.T) {
	for _, c := range casesMDTree {
		t.Run(c.name, func(t *testing.T) {
			tr := NewTree()
			err := ParseMDTree(strings.NewReader(c.md), tr)
			require.NoError(t, err)
			if !assert.ObjectsAreEqual(c.exp, tr.root) {
				ast, err := parseMD(strings.NewReader(c.md))
				require.NoError(t, err)
				printMD(os.Stderr, ast, "")
			}
			require.Equal(t, c.exp, tr.root)
		})
	}
}
