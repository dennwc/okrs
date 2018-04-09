package okrs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type Github struct {
	Token string  `json:"token,omitempty" yaml:"token,omitempty"`
	Cache string  `json:"cache,omitempty" yaml:"cache,omitempty"`
	Orgs  []GHOrg `json:"orgs,omitempty" yaml:"orgs,omitempty"`

	cli  *github.Client
	orgs map[string]*ghOrg
}

type ghOrg struct {
	g     *Github
	name  string
	repos map[string]*ghRepo
}

func (org *ghOrg) loadRepo(ctx context.Context, repo string) error {
	r, ok := org.repos[repo]
	if !ok {
		r = &ghRepo{org: org, name: repo}
		if org.repos == nil {
			org.repos = make(map[string]*ghRepo)
		}
		org.repos[repo] = r
	}
	return r.load(ctx)
}

type ghRepo struct {
	org    *ghOrg
	name   string
	issues map[int]*ghIssue
}

func (r *ghRepo) load(ctx context.Context) error {
	if err := r.loadIssues(ctx); err != nil {
		return err
	}
	for _, is := range r.issues {
		if err := is.parse(ctx); err != nil {
			return err
		}
	}
	return nil
}
func (r *ghRepo) loadIssues(ctx context.Context) error {
	if len(r.issues) != 0 {
		return nil
	}
	issues, err := r.org.g.listIssues(ctx, r.org.name, r.name)
	if err != nil {
		return err
	}
	r.issues = make(map[int]*ghIssue)
	for _, is := range issues {
		r.issues[is.GetNumber()] = &ghIssue{
			repo: r, issue: is,
		}
	}
	return nil
}

type ghIssue struct {
	repo  *ghRepo
	issue *github.Issue
	local *Tree
}

func (is *ghIssue) parse(ctx context.Context) error {
	tr := NewTree()
	err := ParseMDTree(strings.NewReader(is.issue.GetBody()), tr)
	if err != nil {
		return err
	}
	is.local = tr

	//DumpTree(fmt.Sprintf("%s-%s-%d.dot", is.repo.org.name, is.repo.name, is.issue.GetNumber()), tr)

	return nil
}

type GHOrg struct {
	Name string `json:"name" yaml:"name"`
	//Projects []GHProject `json:"projects,omitempty" yaml:"projects,omitempty"`
	Repos []GHRepo `json:"repos,omitempty" yaml:"repos,omitempty"`
}

type GHRepo struct {
	Name string `json:"name" yaml:"name"`
}

//type GHProject struct {
//	ID   int64  `json:"id" yaml:"id"`
//	Name string `json:"name" yaml:"name"`
//}

func (g *Github) client(ctx context.Context) *github.Client {
	if g.cli != nil {
		return g.cli
	}
	var cli *http.Client
	if g.Token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: g.Token},
		)
		cli = oauth2.NewClient(ctx, ts)
	}
	g.cli = github.NewClient(cli)
	return g.cli
}

func (g *Github) LoadTree(ctx context.Context, tr *Tree) error {
	for _, org := range g.Orgs {
		if err := g.loadOrgTree(ctx, tr, org); err != nil {
			return err
		}
	}
	return nil
}

func (g *Github) loadOrgTree(ctx context.Context, tr *Tree, org GHOrg) error {
	if g.orgs == nil {
		g.orgs = make(map[string]*ghOrg)
	}
	o, ok := g.orgs[org.Name]
	if !ok {
		o = &ghOrg{g: g, name: org.Name}
		g.orgs[org.Name] = o
	}
	for _, repo := range org.Repos {
		if err := o.loadRepo(ctx, repo.Name); err != nil {
			return err
		}
	}
	nd := tr.root
	if err := g.asTree(tr, nd); err != nil {
		return err
	}
	if len(nd.Sub) == 1 {
		tr.root = nd.Sub[0]
	}
	return nil
}
func (g *Github) loadProjTree(ctx context.Context, tr *Tree, org string, proj int64) (*Node, error) {
	cols, err := g.listProjColumns(ctx, proj)
	if err != nil {
		return nil, err
	}
	root := tr.NewNode(Node{})
	for _, col := range cols {
		cards, err := g.listProjCards(ctx, col.GetID())
		if err != nil {
			return root, err
		}
		for _, c := range cards {
			url := c.GetContentURL()
			if url == "" && c.Note != nil {
				root.AddChild(tr.NewNode(Node{Desc: c.GetNote()}))
				continue
			}
			nd, err := g.loadByURL(ctx, tr, url)
			if err != nil {
				return root, err
			}
			if nd != nil {
				root.AddChild(nd)
			}
		}
	}
	return root, nil
}

func (g *Github) asTree(tr *Tree, root *Node) error {
	for _, org := range g.orgs {
		u := fmt.Sprintf("https://github.com/%s", org.name)
		nd := tr.NewNode(Node{
			Title: org.name,
			Link: Link{
				Title: org.name,
				URL:   u,
			},
		})
		if err := org.asTree(tr, nd); err != nil {
			return err
		}
		if len(nd.Sub) == 1 {
			root.AddChild(nd.Sub[0])
		} else {
			root.AddChild(nd)
		}
	}
	root.Sort()
	return nil
}

func (org *ghOrg) asTree(tr *Tree, root *Node) error {
	for _, repo := range org.repos {
		u := fmt.Sprintf("https://github.com/%s/%s", org.name, repo.name)
		nd := tr.NewNode(Node{
			Title: repo.name,
			Link: Link{
				Title: repo.name,
				URL:   u,
			},
		})
		if err := repo.asTree(tr, nd); err != nil {
			return err
		}
		root.AddChild(nd)
	}
	root.Sort()
	return nil
}

func (r *ghRepo) asTree(tr *Tree, root *Node) error {
	byNum := make(map[string]*Node)
	byID := make(map[string]*Node) // TODO: should be global?
	byURL := make(map[string]*Node)
	byTitle := make(map[string]*Node)
	nodes := make(map[*ghIssue]*Node)

	// build indexes
	for _, is := range r.issues {
		id := is.issue.GetURL()
		url := is.issue.GetHTMLURL()
		ref := fmt.Sprintf("#%d", is.issue.GetNumber())
		title := is.issue.GetTitle()

		nd := tr.NewNode(Node{
			ID: id, Title: title,
			Link: Link{
				Title: ref, URL: url,
			},
		})

		byID[id] = nd
		byURL[url] = nd
		byNum[ref] = nd
		byTitle[title] = nd
		nodes[is] = nd

		// get parent info from local tree
		local := is.local.Root()
		if local == nil {
			continue
		}
		if nd.parent == nil {
			nd.parent = local.parent
		}
	}

	byLink := func(p Link) *Node {
		if is, ok := byNum[p.Title]; ok {
			return is
		} else if is, ok := byID[p.URL]; ok {
			return is
		} else if is, ok := byURL[p.URL]; ok {
			return is
		}
		return nil
	}

	// resolve all parent links
	parents := make(map[*Node]*Node)
	for _, is := range r.issues {
		nd := nodes[is]
		p := nd.parent
		if p == nil {
			continue
		}
		par := byLink(*p)
		if par == nil {
			return fmt.Errorf("cannot find parent: %+v", p)
		}
		par.AddChild(nd)
		parents[nd] = par
	}

	// resolve local trees
	var resolve func(root *Node, local *Node) error
	resolve = func(root *Node, local *Node) error {
		if root.Priority == nil {
			root.Priority = local.Priority
		}
		if root.Progress == nil {
			root.Progress = local.Progress
		}
		if root.Desc == "" {
			root.Desc = local.Desc
		}
		root.Links = append(root.Links, local.Links...)

		for _, s := range local.Sub {
			l := s.Link

			sn := byLink(l)
			create := sn == nil
			if create {
				// no link for this issue - create a new node
				// TODO: try to match by title?
				// FIXME: this issue might be from different repo
				snc := *s
				snc.Sub = nil
				sn = &snc

				if l.URL != "" {
					byURL[l.URL] = sn
					byID[l.URL] = sn // TODO: should try to decode URL and convert to id
				}
				if l.Title != "" {
					if reHashRef.MatchString(l.Title) {
						byNum[l.Title] = sn
					}
					byTitle[l.Title] = sn
				}
			}
			l = sn.Link

			if l.Title == "" && l.URL != "" {
				if u, err := url.Parse(l.URL); err == nil && u.Host == "github.com" {
					sub := strings.Split(strings.Trim(u.Path, "/"), "/")
					if len(sub) == 4 && sub[2] == "issues" || sub[2] == "pull" {
						org, repo, num := sub[0], sub[1], sub[3]
						if org == r.org.name && repo == r.name {
							l.Title = "#" + num
						} else {
							l.Title = fmt.Sprintf("%s/%s#%s", org, repo, num)
						}
					} else {
						log.Println("missing title for the link:", sub)
					}
				}
				sn.Link = l
			}

			if err := resolve(sn, s); err != nil {
				return err
			}
			if sn.parent != nil {
				l1, l2 := *sn.parent, root.Link
				if l1.Title == l2.Title && reHashRef.MatchString(l1.Title) {
					l1 = l2
					sn.parent = &l1
				}
				if l1.URL != l2.URL {
					return fmt.Errorf("incorrect parent of %v: %v (from parent link in %v) vs %v (from local subtree of %v)",
						sn.Link.Title, l1.Title, sn.Link.Title, l2.Title, l2.Title)
				}
			}
			par := root.Link
			sn.parent = &par

			if op, ok := parents[sn]; !ok {
				parents[sn] = root
				root.AddChild(sn)
			} else {
				// TODO: use order in the local tree
				if op != root {
					return fmt.Errorf("unexpected root for %v: exp %v, got %v",
						sn.Link, root.Link, op.Link)
				}
			}
			// TODO: merge
			root.Sort()
		}
		return nil
	}
	for _, is := range r.issues {
		local := is.local.Root()
		if local == nil {
			continue
		}
		nd := nodes[is]
		if err := resolve(nd, local); err != nil {
			return err
		}
	}

	// add all root nodes to parent node
	for _, is := range r.issues {
		nd := nodes[is]
		if nd.parent == nil {
			root.AddChild(nd)
		}
	}
	return nil
}

func (g *Github) loadByURL(ctx context.Context, tr *Tree, url string) (*Node, error) {
	if strings.Contains(url, "/issues/") {
		return g.loadIssueTreeByURL(ctx, tr, url)
	}
	log.Println("unknown url format:", url)
	return nil, nil
}
func (g *Github) loadIssueTreeByURL(ctx context.Context, tr *Tree, url string) (*Node, error) {
	i := strings.Index(url, "/repos/")
	i += 1 + len("repos/")
	url = strings.Trim(url[i:], "/")
	sub := strings.Split(url, "/")
	var org, repo, nums string
	switch len(sub) {
	case 4:
		org, repo, nums = sub[0], sub[1], sub[3]
	case 5: // github.com/<org>/<repo>/issues/<num>
		if sub[0] != "github.com" || sub[3] != "issues" {
			log.Printf("unexpected url: %s", url)
			return tr.NewNode(Node{Link: Link{URL: url}}), nil
		}
		org, repo, nums = sub[1], sub[2], sub[4]
	default:
		log.Printf("unexpected url: %s", url)
		return tr.NewNode(Node{Link: Link{URL: url}}), nil
	}
	num, err := strconv.Atoi(nums)
	if err != nil {
		return nil, fmt.Errorf("cannot parse issues number %s: %v", nums, err)
	}
	_ = num
	_, _ = org, repo
	panic("not implemented") // FIXME
	//return g.loadIssueTreeByNum(ctx, tr, org, repo, num)
}
func (g *Github) parseLinks(ctx context.Context, org, repo, str string) (string, []string, error) {
	var links []string
	for _, s := range reHashRef.FindAllStringSubmatch(str, -1) {
		str = strings.Replace(str, s[0], "", 1)
		ref, _ := strconv.Atoi(s[1])
		link, err := g.resolveHashRef(ctx, org, repo, ref)
		if err != nil {
			return str, links, err
		}
		links = append(links, link)
	}
	for _, s := range reURL.FindAllStringSubmatch(str, -1) {
		str = strings.Replace(str, s[0], "", 1)
		links = append(links, s[1])
	}
	return str, links, nil
}

func (g *Github) cacheDir() string {
	return g.Cache
}
func (g *Github) cachePath(key string) string {
	return filepath.Join(g.cacheDir(), "gh_"+key+".json")
}
func (g *Github) fromCache(key string, out interface{}) bool {
	if g.cacheDir() == "" {
		return false
	}
	f, err := os.Open(g.cachePath(key))
	if err != nil {
		if !os.IsNotExist(err) {
			log.Println(err)
		}
		return false
	}
	defer f.Close()
	err = json.NewDecoder(f).Decode(out)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}
func (g *Github) writeCache(key string, data interface{}) {
	if g.cacheDir() == "" {
		return
	}
	_ = os.MkdirAll(g.cacheDir(), 0755)
	f, err := os.Create(g.cachePath(key))
	if err != nil {
		return
	}
	defer f.Close()
	json.NewEncoder(f).Encode(data)
}
func (g *Github) getIssue(ctx context.Context, org, repo string, num int) (*github.Issue, error) {
	key := fmt.Sprintf("%s_%s_issue_%v", org, repo, num)
	var iss *github.Issue
	if g.fromCache(key, &iss) {
		return iss, nil
	}
	gh := g.client(ctx)
	iss, _, err := gh.Issues.Get(ctx, org, repo, num)
	if err == nil {
		g.writeCache(key, iss)
	}
	return iss, err
}
func (g *Github) listProjects(ctx context.Context, org string) ([]*github.Project, error) {
	key := fmt.Sprintf("%s_projects", org)
	var out []*github.Project
	if g.fromCache(key, &out) {
		return out, nil
	}
	gh := g.client(ctx)
	for page := 1; ; page++ {
		buf, _, err := gh.Organizations.ListProjects(ctx, org, &github.ProjectListOptions{
			State: "open",
			ListOptions: github.ListOptions{
				Page: page, PerPage: 100,
			},
		})
		out = append(out, buf...)
		if err != nil {
			return out, err
		} else if len(buf) == 0 {
			break
		}
	}
	g.writeCache(key, out)
	return out, nil
}
func (g *Github) listProjColumns(ctx context.Context, proj int64) ([]*github.ProjectColumn, error) {
	key := fmt.Sprintf("project_%d_col", proj)
	var out []*github.ProjectColumn
	if g.fromCache(key, &out) {
		return out, nil
	}
	gh := g.client(ctx)
	for page := 1; ; page++ {
		buf, _, err := gh.Projects.ListProjectColumns(ctx, proj, &github.ListOptions{
			Page: page, PerPage: 100,
		})
		out = append(out, buf...)
		if err != nil {
			return out, err
		} else if len(buf) == 0 {
			break
		}
	}
	g.writeCache(key, out)
	return out, nil
}
func (g *Github) listProjCards(ctx context.Context, col int64) ([]*github.ProjectCard, error) {
	key := fmt.Sprintf("project_cards_%v", col)
	var out []*github.ProjectCard
	if g.fromCache(key, &out) {
		return out, nil
	}
	gh := g.client(ctx)
	for page := 1; ; page++ {
		buf, _, err := gh.Projects.ListProjectCards(ctx, col, &github.ListOptions{
			Page: page, PerPage: 100,
		})
		out = append(out, buf...)
		if err != nil {
			return out, err
		} else if len(buf) == 0 {
			break
		}
	}
	g.writeCache(key, out)
	return out, nil
}
func (g *Github) listIssues(ctx context.Context, org, repo string) ([]*github.Issue, error) {
	key := fmt.Sprintf("%s_%s_issues", org, repo)
	var out []*github.Issue
	if g.fromCache(key, &out) {
		return out, nil
	}
	gh := g.client(ctx)
	for page := 1; ; page++ {
		buf, _, err := gh.Issues.ListByRepo(ctx, org, repo, &github.IssueListByRepoOptions{
			ListOptions: github.ListOptions{
				Page: page, PerPage: 100,
			},
		})
		out = append(out, buf...)
		if err != nil {
			return out, err
		} else if len(buf) == 0 {
			break
		}
	}
	g.writeCache(key, out)
	return out, nil
}
func (g *Github) eachIssue(ctx context.Context, org, repo string, fnc func(*github.Issue) bool) error {
	list, err := g.listIssues(ctx, org, repo)
	for _, is := range list {
		if !fnc(is) {
			break
		}
	}
	return err
}
func (g *Github) resolveHashRef(ctx context.Context, org, repo string, ref int) (string, error) {
	iss, err := g.getIssue(ctx, org, repo, ref)
	if err != nil {
		return "", err
	}
	return iss.GetURL(), nil
}
