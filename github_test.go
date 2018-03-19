package okrs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGHSubtask(t *testing.T) {
	const sample = `
- [x] Task 1 #1105 
  - [ ] Subtask number 1.1 #1141
`
	sub := reSubtask.FindAllStringSubmatch(sample, -1)
	require.Equal(t, [][]string{
		{
			"- [x] Task 1 #1105",
			"",
			"x",
			"Task 1 #1105",
		},
		{
			"  - [ ] Subtask number 1.1 #1141",
			"  ",
			" ",
			"Subtask number 1.1 #1141",
		},
	}, sub)
}
