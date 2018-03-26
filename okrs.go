package okrs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

func ReadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	ext := filepath.Ext(path)
	var c Config
	switch ext {
	case ".json":
		err = json.NewDecoder(f).Decode(&c)
	case ".yml", ".yaml":
		err = yaml.NewDecoder(f).Decode(&c)
	default:
		return nil, fmt.Errorf("unknown file extension: %v", ext)
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

type Output struct {
	Path   string `json:"path,omitempty" yaml:"path,omitempty"`
	Format string `json:"format,omitempty" yaml:"format,omitempty"`
}

func (o Output) WriteTree(tr *Tree) error {
	var wr *TreeWriterDesc
	if ext := filepath.Ext(o.Path); ext != "" && o.Format == "" {
		for _, w := range TreeWriters() {
			if ext == "."+w.Ext {
				wr = &w
				break
			}
		}
	} else {
		wr = TreeWriter(o.Format)
	}
	if wr == nil {
		return fmt.Errorf("unknown format %q", o.Format)
	}
	name := o.Path
	var w io.Writer = os.Stdout
	if name != "" && name != "-" {
		if wr.Ext != "" && !strings.HasSuffix(name, "."+wr.Ext) {
			name += "." + wr.Ext
		}
		f, err := os.Create(name)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}
	return wr.Write(w, tr.root)
}

type Config struct {
	Github   *Github  `json:"github,omitempty" yaml:"github,omitempty"`
	Markdown []string `json:"markdown,omitempty" yaml:"markdown,omitempty"`
	Output   []Output `json:"output,omitempty" yaml:"output,omitempty"`
}

func (c *Config) Run(ctx context.Context) error {
	tr := NewTree()
	for _, path := range c.Markdown {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		err = ParseMDTree(f, tr)
		f.Close()
		if err != nil {
			return err
		}
	}
	if c.Github != nil {
		if err := c.Github.LoadTree(ctx, tr); err != nil {
			return err
		}
	}
	if len(c.Output) == 0 {
		return fmt.Errorf("no outputs specified")
	}
	for _, o := range c.Output {
		if err := o.WriteTree(tr); err != nil {
			return err
		}
	}
	return nil
}
