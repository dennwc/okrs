package okrs

import (
	"encoding/json"
	"fmt"
	"io"
)

func init() {
	RegisterTreeWriter(TreeWriterDesc{
		Name: "mindmup", Ext: "mup",
		Write: func(w io.Writer, t *Node) error {
			enc := json.NewEncoder(w)
			enc.SetIndent("", "\t")
			return enc.Encode(asMindMup(t))
		},
	})
}

func asMindMup(t *Node) interface{} {
	type Attrs struct {
		BNode string `json:"bnode,omitempty"`
		URL   string `json:"url,omitempty"`
	}
	type MupNode struct {
		ID    int             `json:"id"`
		Title string          `json:"title"`
		Attrs *Attrs          `json:"attr,omitempty"`
		Sub   map[int]MupNode `json:"ideas,omitempty"`
	}
	var last int
	var conv func(t *Node) MupNode
	conv = func(t *Node) MupNode {
		last++
		id := last
		n := MupNode{ID: id, Title: t.Title, Sub: make(map[int]MupNode)}
		n.Attrs = &Attrs{BNode: fmt.Sprintf("%p", t), URL: t.URL}
		for i, s := range t.Sub {
			n.Sub[i+1] = conv(s)
		}
		return n
	}

	root := conv(t)
	return struct {
		Vers  int             `json:"formatVersion"`
		ID    string          `json:"id"`
		Title string          `json:"title"`
		Sub   map[int]MupNode `json:"ideas,omitempty"`
		Attrs interface{}     `json:"attr,omitempty"`
		Theme interface{}     `json:"theme,omitempty"`
	}{
		Vers:  3,
		ID:    "root",
		Title: root.Title,
		Sub:   root.Sub,
		Attrs: json.RawMessage(mupAttrs),
		Theme: json.RawMessage(mupTheme),
	}
}

const mupAttrs = `{
    "theme": "topdownStandard",
    "progress-statuses": {
      "not-started": {
        "description": "Not Started",
        "priority": 1,
        "icon": {
          "height": 25,
          "width": 25,
          "position": "left",
          "url": "/assets/progress/tasks/flat/not-started.png"
        }
      },
      "passing": {
        "description": "Done",
        "icon": {
          "height": 25,
          "width": 25,
          "position": "left",
          "url": "/assets/progress/tasks/flat/passing.png"
        }
      },
      "under-review": {
        "description": "Under review",
        "icon": {
          "height": 25,
          "width": 25,
          "position": "left",
          "url": "/assets/progress/tasks/flat/under-review.png"
        }
      },
      "in-progress": {
        "description": "In Progress",
        "priority": 3,
        "icon": {
          "height": 25,
          "width": 25,
          "position": "left",
          "url": "/assets/progress/tasks/flat/in-progress.png"
        }
      },
      "blocked": {
        "description": "Blocked",
        "priority": 4,
        "icon": {
          "height": 25,
          "width": 25,
          "position": "left",
          "url": "/assets/progress/tasks/flat/blocked.png"
        }
      },
      "parked": {
        "description": "Parked",
        "priority": 2,
        "icon": {
          "height": 25,
          "width": 25,
          "position": "left",
          "url": "/assets/progress/tasks/flat/parked.png"
        }
      }
    },
    "icon": {
      "height": 25,
      "width": 25,
      "position": "left",
      "url": "/assets/progress/tasks/flat/in-progress.png"
    },
    "progress": "in-progress",
    "measurements-config": [
      "progress"
    ]
  }`

const mupTheme = `{
    "name": "MindMup Top Down Default",
    "layout": {
      "orientation": "top-down",
      "spacing": {
        "h": 20,
        "v": 100
      }
    },
    "node": [
      {
        "name": "default",
        "cornerRadius": 10,
        "backgroundColor": "#E0E0E0",
        "border": {
          "type": "surround",
          "line": {
            "color": "#707070",
            "width": 1
          }
        },
        "shadow": [
          {
            "color": "#070707",
            "opacity": 0.4,
            "offset": {
              "width": 2,
              "height": 2
            },
            "radius": 2
          }
        ],
        "text": {
          "margin": 5,
          "alignment": "center",
          "maxWidth": 146,
          "color": "#4F4F4F",
          "lightColor": "#EEEEEE",
          "darkColor": "#000000",
          "font": {
            "lineSpacing": 2.5,
            "lineSpacingPx": 3.25,
            "size": 9,
            "sizePx": 12,
            "weight": "bold"
          }
        },
        "connections": {
          "default": {
            "h": "center",
            "v": "base"
          },
          "from": {
            "horizontal": {
              "h": "center",
              "v": "base"
            }
          },
          "to": {
            "h": "center",
            "v": "top"
          }
        },
        "decorations": {
          "height": 20,
          "edge": "top",
          "overlap": true,
          "position": "end"
        }
      },
      {
        "name": "level_1",
        "backgroundColor": "#22AAE0"
      },
      {
        "name": "activated",
        "border": {
          "type": "surround",
          "line": {
            "color": "#22AAE0",
            "width": 3,
            "style": "dotted"
          }
        }
      },
      {
        "name": "level_1.activated",
        "border": {
          "type": "surround",
          "line": {
            "color": "#EEEEEE",
            "width": 3,
            "style": "dotted"
          }
        }
      },
      {
        "name": "selected",
        "shadow": [
          {
            "color": "#000000",
            "opacity": 0.9,
            "offset": {
              "width": 2,
              "height": 2
            },
            "radius": 2
          }
        ]
      },
      {
        "name": "collapsed",
        "shadow": [
          {
            "color": "#888888",
            "offset": {
              "width": 0,
              "height": 1
            },
            "radius": 0
          },
          {
            "color": "#FFFFFF",
            "offset": {
              "width": 0,
              "height": 3
            },
            "radius": 0
          },
          {
            "color": "#888888",
            "offset": {
              "width": 0,
              "height": 4
            },
            "radius": 0
          },
          {
            "color": "#FFFFFF",
            "offset": {
              "width": 0,
              "height": 6
            },
            "radius": 0
          },
          {
            "color": "#888888",
            "offset": {
              "width": 0,
              "height": 7
            },
            "radius": 0
          }
        ]
      },
      {
        "name": "collapsed.selected",
        "shadow": [
          {
            "color": "#FFFFFF",
            "offset": {
              "width": 0,
              "height": 1
            },
            "radius": 0
          },
          {
            "color": "#888888",
            "offset": {
              "width": 0,
              "height": 3
            },
            "radius": 0
          },
          {
            "color": "#FFFFFF",
            "offset": {
              "width": 0,
              "height": 6
            },
            "radius": 0
          },
          {
            "color": "#555555",
            "offset": {
              "width": 0,
              "height": 7
            },
            "radius": 0
          },
          {
            "color": "#FFFFFF",
            "offset": {
              "width": 0,
              "height": 10
            },
            "radius": 0
          },
          {
            "color": "#333333",
            "offset": {
              "width": 0,
              "height": 11
            },
            "radius": 0
          }
        ]
      }
    ],
    "connector": {
      "default": {
        "type": "top-down-s-curve",
        "label": {
          "position": {
            "aboveEnd": 15
          },
          "backgroundColor": "white",
          "borderColor": "white",
          "text": {
            "color": "#4F4F4F",
            "font": {
              "size": 9,
              "sizePx": 12,
              "weight": "normal"
            }
          }
        },
        "line": {
          "color": "#707070",
          "width": 2
        }
      }
    }
  }`
