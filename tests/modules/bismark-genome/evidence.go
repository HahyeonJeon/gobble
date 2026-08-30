// Package bismarkgenomeevidence owns expected Bismark genome module facts.
package bismarkgenomeevidence

import pc "github.com/HahyeonJeon/gobble/tests/internal/plancheck"

// Image is the graph-stable Bismark image reference.
const Image = "community.wave.seqera.io/library/bismark:0.25.1--1f50935de5d79c47"

// Members returns the expected ordered Bismark index members under dir.
func Members(dir string) []pc.Member {
	ct := dir + "/Bisulfite_Genome/CT_conversion"
	ga := dir + "/Bisulfite_Genome/GA_conversion"
	return []pc.Member{
		{Name: "CT_1", Path: ct + "/BS_CT.1.bt2"},
		{Name: "CT_2", Path: ct + "/BS_CT.2.bt2"},
		{Name: "CT_3", Path: ct + "/BS_CT.3.bt2"},
		{Name: "CT_4", Path: ct + "/BS_CT.4.bt2"},
		{Name: "CT_rev1", Path: ct + "/BS_CT.rev.1.bt2"},
		{Name: "CT_rev2", Path: ct + "/BS_CT.rev.2.bt2"},
		{Name: "CT_mfa", Path: ct + "/genome_mfa.CT_conversion.fa"},
		{Name: "GA_1", Path: ga + "/BS_GA.1.bt2"},
		{Name: "GA_2", Path: ga + "/BS_GA.2.bt2"},
		{Name: "GA_3", Path: ga + "/BS_GA.3.bt2"},
		{Name: "GA_4", Path: ga + "/BS_GA.4.bt2"},
		{Name: "GA_rev1", Path: ga + "/BS_GA.rev.1.bt2"},
		{Name: "GA_rev2", Path: ga + "/BS_GA.rev.2.bt2"},
		{Name: "GA_mfa", Path: ga + "/genome_mfa.GA_conversion.fa"},
	}
}

// PublishedPaths returns only the expected member paths.
func PublishedPaths(dir string) []string {
	members := Members(dir)
	paths := make([]string, len(members))
	for i, member := range members {
		paths[i] = member.Path
	}
	return paths
}
