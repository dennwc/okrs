package okrs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type Github struct {
	Token string  `json:"token,omitempty" yaml:"token,omitempty"`
	Orgs  []GHOrg `json:"orgs,omitempty" yaml:"orgs,omitempty"`

	cli *github.Client
}

type GHOrg struct {
	Name     string      `json:"name" yaml:"name"`
	Projects []GHProject `json:"projects,omitempty" yaml:"projects,omitempty"`
	Repos    []GHRepo    `json:"repos,omitempty" yaml:"repos,omitempty"`
}

type GHRepo struct {
	Name string `json:"name" yaml:"name"`
}

type GHProject struct {
	ID   int64  `json:"id" yaml:"id"`
	Name string `json:"name" yaml:"name"`
}

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

func (g *Github) LoadTree(ctx context.Context) (*TreeNode, error) {
	root := &TreeNode{}
	for _, org := range g.Orgs {
		node, err := g.loadOrgTree(ctx, org)
		if err != nil {
			return root, err
		}
		root.Sub = append(root.Sub, node)
	}
	return root, nil
}

func (g *Github) loadOrgTree(ctx context.Context, org GHOrg) (*TreeNode, error) {
	proj, err := g.listProjects(ctx, org.Name)
	if err != nil {
		return nil, err
	}
	root := &TreeNode{Title: org.Name}
	if len(org.Projects) != 0 { // load selected projects
		for _, p := range org.Projects {
			id := p.ID
			if id == 0 {
				for _, p2 := range proj {
					if p.Name == p2.GetName() {
						id = p2.GetID()
						break
					}
				}
				if id == 0 {
					return root, fmt.Errorf("cannot find project with name %q", p.Name)
				}
			}
			node, err := g.loadProjTree(ctx, org.Name, id)
			if err != nil {
				return root, err
			}
			if node.Title == "" && p.Name != "" {
				node.Title = p.Name
			}
			root.Sub = append(root.Sub, node)
		}
	}
	if len(org.Repos) != 0 { // load selected repositories
		for _, repo := range org.Repos {
			arr, err := g.listIssues(ctx, org.Name, repo.Name)
			if err != nil {
				return root, err
			}
			m := make(map[string]*TreeNode)
			left := make(map[string][]*TreeNode)
			for _, iss := range arr {
				nd, err := g.loadIssue(ctx, org.Name, repo.Name, iss)
				if err != nil {
					return root, err
				}
				if nd.parent == "" {
					root.Sub = append(root.Sub, nd)
				} else {
					if par := m[nd.parent]; par != nil {
						par.Sub = append(par.Sub, nd)
					} else {
						left[nd.parent] = append(left[nd.parent], nd)
					}
				}
				if list := left[nd.ID]; len(list) > 0 {
					nd.Sub = append(nd.Sub, list...)
					delete(left, nd.ID)
				}
				m[nd.ID] = nd
			}
			for _, list := range left {
				root.Sub = append(root.Sub, list...)
			}
		}
	}
	if len(org.Projects) == 0 && len(org.Repos) == 0 {
		// defaults to loading projects
		for _, p := range proj {
			node, err := g.loadProjTree(ctx, org.Name, p.GetID())
			if err != nil {
				return root, err
			}
			if node.Title == "" {
				node.Title = p.GetName()
			}
			root.Sub = append(root.Sub, node)
		}
	}
	return root, nil
}
func (g *Github) loadProjTree(ctx context.Context, org string, proj int64) (*TreeNode, error) {
	cols, err := g.listProjColumns(ctx, proj)
	if err != nil {
		return nil, err
	}
	root := &TreeNode{}
	for _, col := range cols {
		cards, err := g.listProjCards(ctx, col.GetID())
		if err != nil {
			return root, err
		}
		for _, c := range cards {
			url := c.GetContentURL()
			if url == "" && c.Note != nil {
				root.Sub = append(root.Sub, &TreeNode{Desc: c.GetNote()})
				continue
			}
			nd, err := g.loadByURL(ctx, url)
			if err != nil {
				return root, err
			}
			root.Sub = append(root.Sub, nd)
		}
	}
	return root, nil
}

var (
	reSubtask       = regexp.MustCompile(`([\t ]*)-\s+\[(\s|x)\]\s+([^\s].*[^\s])`)
	reHashRef       = regexp.MustCompile(`#(\d+)`)
	reURL           = regexp.MustCompile(`\(?(?:\[[^]]+\]\()?(http(?:s)?://[^)\s]+)\)?\)?`)
	rePriority      = regexp.MustCompile(`\[P(\d+)\]\s*`)
	reParent        = regexp.MustCompile(`\*\*Parent(?:\s[^\s]+)?:\*\*\s+([^\n\r]+)`)
	reProgressPerc  = regexp.MustCompile(`\*\*Progress:\*\*\s+([\d]+)%`)
	reProgressParts = regexp.MustCompile(`\*\*Progress:\*\*\s+([\d]+)/([\d]+)`)
)

func (g *Github) loadByURL(ctx context.Context, url string) (*TreeNode, error) {
	nd := &TreeNode{URL: url}
	if strings.Contains(url, "/issues/") {
		return g.loadIssueTreeByURL(ctx, url)
	} else {
		log.Println("unknown url format:", url)
	}
	return nd, nil
}
func (g *Github) loadIssue(ctx context.Context, org, repo string, iss *github.Issue) (*TreeNode, error) {
	nd := &TreeNode{
		ID:    iss.GetURL(),
		URL:   iss.GetHTMLURL(),
		Title: iss.GetTitle(),
	}
	err := g.parseIssueBody(ctx, org, repo, iss.GetBody(), nd)
	return nd, err
}
func (g *Github) loadIssueTreeByNum(ctx context.Context, org, repo string, num int) (*TreeNode, error) {
	iss, err := g.getIssue(ctx, org, repo, num)
	if err != nil {
		return nil, err
	}
	return g.loadIssue(ctx, org, repo, iss)
}
func (g *Github) loadIssueTreeByURL(ctx context.Context, url string) (*TreeNode, error) {
	nd := &TreeNode{URL: url}
	i := strings.Index(url, "/repos/")
	i += 1 + len("repos/")
	url = strings.Trim(url[i:], "/")
	sub := strings.Split(url, "/")
	if len(sub) != 4 {
		log.Printf("unexpected url: %s", url)
		return nd, nil
	}
	org, repo, nums := sub[0], sub[1], sub[3]
	num, err := strconv.Atoi(nums)
	if err != nil {
		return nil, fmt.Errorf("cannot parse issues number %s: %v", nums, err)
	}
	return g.loadIssueTreeByNum(ctx, org, repo, num)
}
func (g *Github) parseIssueBody(ctx context.Context, org, repo, body string, nd *TreeNode) error {
	if sub := reParent.FindStringSubmatch(body); len(sub) > 0 {
		_, links, err := g.parseLinks(ctx, org, repo, sub[1])
		if err != nil {
			return err
		} else if len(links) > 1 {
			return fmt.Errorf("only one parent should be specified, got: %v", links)
		}
		if len(links) == 1 {
			nd.parent = links[0]
		}
	}
	if sub := reProgressPerc.FindStringSubmatch(body); len(sub) > 0 {
		perc, err := strconv.ParseFloat(sub[1], 64)
		if err != nil {
			return fmt.Errorf("cannot parse percents: %v", err)
		}
		nd.Progress = &Progress{Done: int(perc), Total: 100}
	} else if sub = reProgressParts.FindStringSubmatch(body); len(sub) > 0 {
		done, err := strconv.ParseInt(sub[1], 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse done parts: %v", err)
		}
		total, err := strconv.ParseInt(sub[2], 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse total parts: %v", err)
		}
		nd.Progress = &Progress{Done: int(done), Total: int(total)}
	}
	var err error
	nd.Sub, err = g.parseIssueItems(ctx, org, repo, body)
	return err
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
func (g *Github) parseIssueItems(ctx context.Context, org, repo, body string) ([]*TreeNode, error) {
	type Task struct {
		Node *TreeNode
		Lvl  int
	}
	var tasks []Task
	depth := make(map[int]int)
	for _, sub := range reSubtask.FindAllStringSubmatch(body, -1) {
		lvl, done, title := len(sub[1]), sub[2] == "x", sub[3]
		_ = lvl
		var (
			links []string
			err   error
		)
		title, links, err = g.parseLinks(ctx, org, repo, title)
		if err != nil {
			return nil, err
		}
		var pri *int
		if sub := rePriority.FindStringSubmatch(title); len(sub) > 0 {
			title = strings.Replace(title, sub[0], "", 1)
			pr, err := strconv.Atoi(sub[1])
			if err != nil {
				return nil, fmt.Errorf("cannot parse priority: %v", err)
			}
			pri = &pr
		}
		nd := &TreeNode{Title: strings.TrimSpace(title), Priority: pri, Progress: &Progress{Total: 1}}
		if len(links) == 1 {
			nd.URL = links[0]
		} else if len(links) > 1 {
			for _, l := range links {
				sn, err := g.loadByURL(ctx, l)
				if err != nil {
					return nil, err
				}
				nd.Sub = append(nd.Sub, sn)
			}
		}
		if done {
			nd.Progress.Done = 1
		}
		tasks = append(tasks, Task{Node: nd, Lvl: lvl})
		depth[lvl] = lvl
	}
	var lvls []int
	for l := range depth {
		lvls = append(lvls, l)
	}
	sort.Ints(lvls)
	for i, l := range lvls {
		depth[l] = i
	}
	root := &TreeNode{}
	curAt := func(dst int) *TreeNode {
		n, lvl := root, 0
		for lvl < dst {
			if len(n.Sub) == 0 {
				n.Sub = append(n.Sub, &TreeNode{})
			}
			n = n.Sub[len(n.Sub)-1]
			lvl++
		}
		return n
	}
	for _, t := range tasks {
		par := curAt(depth[t.Lvl])
		par.Sub = append(par.Sub, t.Node)
	}
	var fix func(*TreeNode)
	fix = func(n *TreeNode) {
		for i := range n.Sub {
			fix(n.Sub[i])
		}
		if len(n.Sub) == 1 {
			s := n.Sub[0]
			if s.Title == "" && s.URL == "" && s.Desc == "" && s.ID == "" {
				n.Sub = s.Sub
			}
		}
	}
	fix(root)
	return root.Sub, nil
}

const cacheDir = "./.cache/"

func (g *Github) cachePath(key string) string {
	return filepath.Join(cacheDir, "gh_"+key+".json")
}
func (g *Github) fromCache(key string, out interface{}) bool {
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
	_ = os.MkdirAll(cacheDir, 0755)
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
