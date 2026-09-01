package featurecounts

import "github.com/HahyeonJeon/gobble"

func featureCountsStrand(s string) string {
	if s == "" {
		s = gobble.DefaultRNAStrandedness
	}
	switch s {
	case gobble.StrandednessUnstranded:
		return "0"
	case gobble.StrandednessForward:
		return "1"
	case gobble.StrandednessReverse:
		return "2"
	default:
		return "0"
	}
}
