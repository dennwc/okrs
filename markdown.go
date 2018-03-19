package okrs

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"gopkg.in/russross/blackfriday.v2"
)

func ParseMDTree(r io.Reader) (TreeNode, error) {
	ast, err := parseMD(r)
	if err != nil {
		return TreeNode{}, err
	}
	return mdDoc2Tree(ast), nil
}

func parseMD(r io.Reader) (*blackfriday.Node, error) {
	parser := blackfriday.New()
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return parser.Parse(data), nil
}

var mdTypeNames = map[blackfriday.NodeType]string{
	blackfriday.Document:       "Document",
	blackfriday.BlockQuote:     "BlockQuote",
	blackfriday.List:           "List",
	blackfriday.Item:           "Item",
	blackfriday.Paragraph:      "Paragraph",
	blackfriday.Heading:        "Heading",
	blackfriday.HorizontalRule: "HorizontalRule",
	blackfriday.Emph:           "Emph",
	blackfriday.Strong:         "Strong",
	blackfriday.Del:            "Del",
	blackfriday.Link:           "Link",
	blackfriday.Image:          "Image",
	blackfriday.Text:           "Text",
	blackfriday.HTMLBlock:      "HTMLBlock",
	blackfriday.CodeBlock:      "CodeBlock",
	blackfriday.Softbreak:      "Softbreak",
	blackfriday.Hardbreak:      "Hardbreak",
	blackfriday.Code:           "Code",
	blackfriday.HTMLSpan:       "HTMLSpan",
	blackfriday.Table:          "Table",
	blackfriday.TableCell:      "TableCell",
	blackfriday.TableHead:      "TableHead",
	blackfriday.TableBody:      "TableBody",
	blackfriday.TableRow:       "TableRow",
}

func printMD(w io.Writer, n *blackfriday.Node, tabs string) {
	if n == nil {
		return
	}
	typ := mdTypeNames[n.Type]
	if typ == "" {
		typ = fmt.Sprint("node", n.Type)
	}
	const tab = "  "
	fmt.Fprintf(w, tabs+"%s {\n", typ)
	defer fmt.Fprint(w, tabs+"}\n")
	switch n.Type {
	case blackfriday.Text:
		fmt.Fprintf(w, tabs+tab+"%q\n", string(n.Literal))
	}
	for n := n.FirstChild; n != nil; n = n.Next {
		printMD(w, n, tabs+tab)
	}
}

func mdDoc2Tree(doc *blackfriday.Node) TreeNode {
	var root TreeNode
	cur := func() (*TreeNode, int) {
		n, lvl := &root, 0
		for len(n.Sub) != 0 {
			n = &n.Sub[len(n.Sub)-1]
			lvl++
		}
		return n, lvl
	}
	curAt := func(dst int) *TreeNode {
		n, lvl := &root, 0
		for lvl < dst {
			if len(n.Sub) == 0 {
				n.Sub = append(n.Sub, TreeNode{})
			}
			n = &n.Sub[len(n.Sub)-1]
			lvl++
		}
		return n
	}
	for n := doc.FirstChild; n != nil; n = n.Next {
		switch n.Type {
		case blackfriday.Heading:
			par := curAt(n.HeadingData.Level - 1)
			var nd TreeNode
			if txt := n.FirstChild; txt != nil && txt.Type == blackfriday.Text {
				nd.Title = strings.TrimRight(string(txt.Literal), ":")
			}
			par.Sub = append(par.Sub, nd)
		case blackfriday.Paragraph:
			desc := ""
			if txt := n.FirstChild; txt != nil && txt.Type == blackfriday.Text {
				desc = strings.TrimSpace(string(txt.Literal))
			}
			c, _ := cur()
			if c.Desc == "" {
				c.Desc = desc
			} else {
				c.Desc += "\n" + desc
			}
		case blackfriday.List:
			c, _ := cur()
			c.Sub = append(c.Sub, mdList2Tree(n)...)
		}
	}
	for len(root.Sub) == 1 && root.Title == "" {
		root = root.Sub[0]
	}
	return root
}

func mdList2Tree(list *blackfriday.Node) []TreeNode {
	var out []TreeNode
	for n := list.FirstChild; n != nil; n = n.Next {
		switch n.Type {
		case blackfriday.Item:
			out = append(out, mdItem2Tree(n))
		}
	}
	return out
}

func mdItem2Tree(root *blackfriday.Node) TreeNode {
	cur := TreeNode{}
	for n := root.FirstChild; n != nil; n = n.Next {
		switch n.Type {
		case blackfriday.Paragraph:
			if txt := n.FirstChild; txt != nil && txt.Type == blackfriday.Text {
				cur.Title = string(txt.Literal)
			}
		case blackfriday.List:
			cur.Sub = mdList2Tree(n)
		}
	}
	return cur
}
