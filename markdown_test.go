package okrs

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var casesMDTree = []struct {
	name string
	md   string
	exp  Node
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
`,
		exp: Node{
			Title: "Root",
			Desc:  "Some description",
			Sub: []Node{
				{
					Title: "Level 2",
					Desc:  "Description 2",
					Sub: []Node{
						{
							Title: "sub 1",
							Sub: []Node{
								{Title: "sub 1.1"},
								{Title: "sub 1.2"},
							},
						},
						{
							Title: "sub 2",
							Sub: []Node{
								{Title: "sub 2.1"},
								{Title: "sub 2.2"},
							},
						},
						{
							Title: "Level 3",
							Desc:  "Description 3",
							Sub: []Node{
								{Title: "sub 3.1"},
								{Title: "sub 3.2"},
							},
						},
					},
				},
				{
					Title: "Level 2.2",
					Sub: []Node{
						{Sub: []Node{
							{Sub: []Node{
								{Title: "Level 5"},
							}},
						}},
					},
				},
				{Title: "Level 2.3"},
			},
		},
	},
}

func TestMDTree(t *testing.T) {
	for _, c := range casesMDTree {
		t.Run(c.name, func(t *testing.T) {
			ast, err := parseMD(strings.NewReader(c.md))
			require.NoError(t, err)
			got := mdDoc2Tree(ast)
			require.Equal(t, c.exp, got)
		})
	}
}
