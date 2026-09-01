package bwamem

import "github.com/HahyeonJeon/gobble"

var indexMemberNames = []string{"amb", "ann", "bwt", "pac", "sa"}

func inputIndexGroup() gobble.Group {
	group := make(gobble.Group, 0, len(indexMemberNames))
	for _, name := range indexMemberNames {
		group = append(group, gobble.Member{Name: name})
	}
	return group
}
