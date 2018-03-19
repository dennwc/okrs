package okrs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGHSubtask(t *testing.T) {
	const sample = `
- [x] Run MinHashCUDA on weighted sets and store the hashes in a database #1105 
- [ ] Add the precise similarity refiner for the communities #1141
`
	sub := reSubtask.FindAllStringSubmatch(sample, -1)
	require.Equal(t, [][]string{
		{
			"- [x] Run MinHashCUDA on weighted sets and store the hashes in a database #1105 \n",
			"x",
			"Run MinHashCUDA on weighted sets and store the hashes in a database #1105",
		},
		{
			"- [ ] Add the precise similarity refiner for the communities #1141\n",
			" ",
			"Add the precise similarity refiner for the communities #1141",
		},
	}, sub)
}
