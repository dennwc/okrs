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

func (g *Github) LoadTree(ctx context.Context) (TreeNode, error) {
	var root TreeNode
	for _, org := range g.Orgs {
		node, err := g.loadOrgTree(ctx, org)
		if err != nil {
			return root, err
		}
		root.Sub = append(root.Sub, node)
	}
	return root, nil
}

func (g *Github) loadOrgTree(ctx context.Context, org GHOrg) (TreeNode, error) {
	proj, err := g.listProjects(ctx, org.Name)
	if err != nil {
		return TreeNode{}, err
	}
	root := TreeNode{Title: org.Name}
	if len(org.Projects) == 0 {
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
	} else {
		for _, p := range org.Projects {
			id := p.ID
			if id == 0 {
				for _, p2 := range proj {
					if p.Name == p2.GetName() {
						id = p2.GetID()
						break
					}
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
	return root, nil
}
func (g *Github) loadProjTree(ctx context.Context, org string, proj int64) (TreeNode, error) {
	cols, err := g.listProjColumns(ctx, proj)
	if err != nil {
		return TreeNode{}, err
	}
	var root TreeNode
	for _, col := range cols {
		cards, err := g.listProjCards(ctx, col.GetID())
		if err != nil {
			return root, err
		}
		for _, c := range cards {
			url := c.GetContentURL()
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
	reSubtask = regexp.MustCompile(`-\s+\[(\s|x)\]\s+([^\s].*[^\s])\s*`)
	reHashRef = regexp.MustCompile(`\s+#(\d+)`)
	reURL     = regexp.MustCompile(`\s+\(?(?:\[[^]]+\]\()?(http(?:s)?://[^)\s]+)\)?\)?`)
)

func (g *Github) loadByURL(ctx context.Context, url string) (TreeNode, error) {
	nd := TreeNode{URL: url}
	if strings.Contains(url, "/issues/") {
		return g.loadIssueTreeByURL(ctx, url)
	} else {
		log.Println("unknown url format:", url)
	}
	return nd, nil
}
func (g *Github) loadIssueTreeByURL(ctx context.Context, url string) (TreeNode, error) {
	i := strings.Index(url, "/repos/")
	i += 1 + len("repos/")
	url = strings.Trim(url[i:], "/")
	sub := strings.Split(url, "/")
	if len(sub) != 4 {
		return TreeNode{}, fmt.Errorf("unexpected url: %s", url)
	}
	org, repo, nums := sub[0], sub[1], sub[3]
	num, err := strconv.Atoi(nums)
	if err != nil {
		return TreeNode{}, fmt.Errorf("cannot parse issues number %s: %v", nums, err)
	}
	iss, err := g.getIssue(ctx, org, repo, num)
	if err != nil {
		return TreeNode{}, err
	}
	nd := TreeNode{
		ID:    iss.GetURL(),
		URL:   iss.GetHTMLURL(),
		Title: iss.GetTitle(),
	}
	nd.Sub, err = g.parseIssueBody(ctx, org, repo, iss.GetBody())
	return nd, err
}
func (g *Github) parseIssueBody(ctx context.Context, org, repo, body string) ([]TreeNode, error) {
	var out []TreeNode
	// TODO: preserve subtasks hierarchy
	for _, sub := range reSubtask.FindAllStringSubmatch(body, -1) {
		done, title := sub[1] == "x", sub[2]
		var links []string
		for _, s := range reHashRef.FindAllStringSubmatch(title, -1) {
			title = strings.Replace(title, s[0], "", 1)
			ref, _ := strconv.Atoi(s[1])
			link, err := g.resolveHashRef(ctx, org, repo, ref)
			if err != nil {
				return out, err
			}
			links = append(links, link)
		}
		for _, s := range reURL.FindAllStringSubmatch(title, -1) {
			title = strings.Replace(title, s[0], "", 1)
			links = append(links, s[1])
		}
		nd := TreeNode{Title: title, Progress: &Progress{Total: 1}}
		if len(links) == 1 {
			nd.URL = links[0]
		} else if len(links) > 1 {
			for _, l := range links {
				sn, err := g.loadByURL(ctx, l)
				if err != nil {
					return out, err
				}
				nd.Sub = append(nd.Sub, sn)
			}
		}
		if done {
			nd.Progress.Done = 1
		}
		out = append(out, nd)
	}
	return out, nil
}

const cacheDir = "./.cache/"

func (g *Github) cachePath(key string) string {
	return filepath.Join(cacheDir, "gh_"+key+".json")
}
func (g *Github) fromCache(key string, out interface{}) bool {
	f, err := os.Open(g.cachePath(key))
	if err != nil {
		log.Println(err)
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
