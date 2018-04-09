package okrs

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"gopkg.in/russross/blackfriday.v2"
)

func init() {
	RegisterTreeWriter(TreeWriterDesc{
		Name: "md", Ext: "md",
		Write: WriteMDTree,
	})
}

func ParseMDTree(r io.Reader, tr *Tree) error {
	ast, err := parseMD(r)
	if err != nil {
		return err
	}
	mdDoc2Tree(tr, ast)
	return nil
}

func parseMD(r io.Reader) (*blackfriday.Node, error) {
	parser := blackfriday.New()
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	data = bytes.Replace(data, []byte("\r\n"), []byte("\n"), -1)
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

var (
	rePriority = regexp.MustCompile(`\[P(\d+)\]\s*`)
	rePerc     = regexp.MustCompile(`([\d]+)%`)
	reParts    = regexp.MustCompile(`([\d]+)/([\d]+)`)
	reHashRef  = regexp.MustCompile(`#(\d+)`)
	reURL      = regexp.MustCompile(`\(?(?:\[[^]]+\]\()?(http(?:s)?://[^)\s]+)\)?\)?`)
)

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

func mdDoc2Tree(tr *Tree, doc *blackfriday.Node) {
	root := tr.root
	cur := func() (*Node, int) {
		n, lvl := root, 0
		for len(n.Sub) != 0 {
			n = n.Sub[len(n.Sub)-1]
			lvl++
		}
		return n, lvl
	}
	curAt := func(dst int) *Node {
		n, lvl := root, 0
		for lvl < dst {
			if len(n.Sub) == 0 {
				n.AddChild(tr.NewNode(Node{}))
			}
			n = n.Sub[len(n.Sub)-1]
			lvl++
		}
		return n
	}
	for n := doc.FirstChild; n != nil; n = n.Next {
		switch n.Type {
		case blackfriday.Heading:
			par := curAt(n.HeadingData.Level - 1)
			var nd Node
			if txt := n.FirstChild; txt != nil && txt.Type == blackfriday.Text {
				nd.Title = strings.TrimRight(string(txt.Literal), ":")
			}
			par.AddChild(tr.NewNode(nd))
		case blackfriday.Paragraph:
			c, _ := cur()
			mdParToDesc(c, n)
		case blackfriday.List:
			c, _ := cur()
			c.AddChild(mdList2Tree(tr, n)...)
		}
	}
	for len(tr.root.Sub) == 1 && tr.root.isProxyNode() {
		tr.root = tr.root.Sub[0]
	}
}

func mdParToDesc(nd *Node, par *blackfriday.Node) {
	desc := ""
	for n := par.FirstChild; n != nil; n = n.Next {
		switch n.Type {
		case blackfriday.Text:
			if desc == "" {
				desc = strings.TrimSpace(string(n.Literal))
			}
		case blackfriday.Strong:
			key := strings.TrimSuffix(string(n.FirstChild.Literal), ":")
			vnode := n.Next
			if vnode == nil || vnode.Type != blackfriday.Text {
				continue
			}
			n = vnode // skip text value
			val := strings.TrimSpace(string(vnode.Literal))
			switch key {
			case "Progress":
				if sub := rePerc.FindStringSubmatch(val); len(sub) > 0 {
					perc, err := strconv.ParseFloat(sub[1], 64)
					if err != nil {
						log.Println(fmt.Errorf("cannot parse percents: %v", err))
						continue
					}
					if v := int(perc); v != 0 {
						nd.Progress = &Progress{Done: v, Total: 100}
					}
				} else if sub = reParts.FindStringSubmatch(val); len(sub) > 0 {
					done, err := strconv.ParseInt(sub[1], 10, 64)
					if err != nil {
						log.Println(fmt.Errorf("cannot parse done parts: %v", err))
						continue
					}
					total, err := strconv.ParseInt(sub[2], 10, 64)
					if err != nil {
						log.Println(fmt.Errorf("cannot parse total parts: %v", err))
						continue
					}
					nd.Progress = &Progress{Done: int(done), Total: int(total)}
				}
			default:
				switch {
				case strings.HasPrefix(key, "Parent"):
					var u Link
					if val != "" {
						if sub := reHashRef.FindStringSubmatch(val); len(sub) != 0 {
							u.Title = "#" + sub[1]
						}
						if sub := reURL.FindStringSubmatch(val); len(sub) != 0 {
							u.URL = sub[1]
						}
					} else if lnk := vnode.Next; lnk != nil && lnk.Type == blackfriday.Link {
						n = lnk // skip link value
						u.URL = string(lnk.LinkData.Destination)
						if len(lnk.LinkData.Title) != 0 {
							u.Title = string(lnk.LinkData.Title)
						} else if txt := lnk.FirstChild; txt != nil && txt.Type == blackfriday.Text {
							u.Title = string(txt.Literal)
						}
					}
					if u.URL == "" {
						u.URL = u.Title
					}
					if u != (Link{}) {
						nd.parent = &u
					}
				}
			}
		}
	}
	if nd.Desc == "" {
		nd.Desc = desc
	} else {
		nd.Desc += "\n" + desc
	}
}

func mdList2Tree(tr *Tree, list *blackfriday.Node) []*Node {
	var out []*Node
	for n := list.FirstChild; n != nil; n = n.Next {
		switch n.Type {
		case blackfriday.Item:
			out = append(out, mdItem2Tree(tr, n))
		}
	}
	return out
}

func parseTitle(n *Node, s string) {
	if sub := rePriority.FindStringSubmatch(s); len(sub) > 0 {
		s = strings.Replace(s, sub[0], "", 1)
		pr, err := strconv.Atoi(sub[1])
		if err == nil {
			n.Priority = &pr
		}
	}
	var links []Link
	for _, sub := range reHashRef.FindAllStringSubmatch(s, -1) {
		s = strings.Replace(s, sub[0], "", 1)
		links = append(links, Link{
			Title: "#" + sub[1],
			URL:   "#" + sub[1],
		})
	}
	for _, sub := range reURL.FindAllStringSubmatch(s, -1) {
		s = strings.Replace(s, sub[0], "", 1)
		links = append(links, Link{
			URL: sub[1],
		})
	}
	if len(links) == 1 {
		n.Link = links[0]
	} else {
		n.Links = links
	}
	n.Title = strings.TrimSpace(s)
}

func mdItem2Tree(tr *Tree, root *blackfriday.Node) *Node {
	var cur Node
	for n := root.FirstChild; n != nil; n = n.Next {
		switch n.Type {
		case blackfriday.Paragraph:
			if txt := n.FirstChild; txt != nil && txt.Type == blackfriday.Text {
				s := string(txt.Literal)
				if len(s) > 4 && s[0] == '[' && s[2] == ']' && unicode.IsSpace(rune(s[3])) {
					done := !unicode.IsSpace(rune(s[1]))
					s = strings.TrimSpace(s[4:])
					if done {
						cur.Progress = &Progress{Done: 1, Total: 1}
					}
				}
				parseTitle(&cur, s)
			}
		case blackfriday.List:
			cur.Sub = mdList2Tree(tr, n)
		}
	}
	return tr.NewNode(cur)
}

func WriteMDTree(w io.Writer, tree *Node) error {
	return writeMDTree(w, tree, 1, -1)
}

func writeMDTree(w io.Writer, node *Node, lvl, blvl int) error {
	var last error
	write := func(format string, args ...interface{}) {
		_, err := fmt.Fprintf(w, format, args...)
		if err != nil {
			last = err
		}
	}
	var hasDesc func(*Node) bool
	hasDesc = func(node *Node) bool {
		if node.Desc != "" {
			return true
		}
		for _, c := range node.Sub {
			if hasDesc(c) {
				return true
			}
		}
		return false
	}
	if blvl > 0 {
		title := node.Title
		if node.Priority != nil {
			title = fmt.Sprintf("[P%d] %s", *node.Priority, title)
		}
		if u := node.Link; u.URL != "" {
			txt := "link"
			if u.Title != "" {
				txt = u.Title
			}
			title += fmt.Sprintf(" ([%s](%s))", txt, u.URL)
		}
		write("%s* %s\n", strings.Repeat("\t", blvl-1), title)
		for _, c := range node.Sub {
			if err := writeMDTree(w, c, lvl+1, blvl+1); err != nil {
				return err
			}
		}
		return last
	}
	if node.Title != "" {
		write("%s %s\n\n", strings.Repeat("#", lvl), node.Title)
	}
	if node.Link.URL != "" {
		write("[Source page](%s)\n\n", node.Link.URL)
	}
	if p := node.GetProgress(); p != (Progress{}) {
		if p.Total == 100 {
			write("**Progress:** %d%%\n\n", p.Done)
		} else {
			write("**Progress:** %d/%d\n\n", p.Done, p.Total)
		}
	}
	if node.Desc != "" {
		if blvl < 0 {
			blvl = 0
		}
		write("%s\n\n", node.Desc)
	}
	if last != nil {
		return last
	}
	if blvl == 0 {
		noDesc := true
		for _, c := range node.Sub {
			if hasDesc(c) {
				noDesc = false
				break
			}
		}
		if noDesc {
			for _, c := range node.Sub {
				if err := writeMDTree(w, c, lvl+1, blvl+1); err != nil {
					return err
				}
			}
			write("\n")
			return last
		}
	}
	for _, c := range node.Sub {
		bl := blvl
		if bl < 0 && len(node.Sub) > 1 {
			bl = 0
		}
		if err := writeMDTree(w, c, lvl+1, bl); err != nil {
			return err
		}
	}
	return last
}
