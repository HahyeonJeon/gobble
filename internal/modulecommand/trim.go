// Package modulecommand contains pure command construction shared by the modules
// and Gobble's review verifier. It has no engine, Project or execution access.
package modulecommand

import "strconv"

const TrimImage = "community.wave.seqera.io/library/trim-galore:2.1.0--27e6376b8f6c1872@sha256:9d747504e44dbf5dfa8a2d66cbbd3bd80f897cc2e17ebe406821b4809b34a3a4"

type TrimOptions struct {
	Read1, Read2, OutDir, Prefix                                               string
	Cores, ClipR1, ClipR2, ThreePrimeClipR1, ThreePrimeClipR2, Quality, Length int
	Adapter, Adapter2                                                          string
}

// Trim returns argv only; callers own validation and execution policy.
func Trim(o TrimOptions) []string {
	command := []string{"trim_galore", "--cores", strconv.Itoa(o.Cores), "--gzip", "--output_dir", o.OutDir, "--basename", o.Prefix}
	if o.Read2 != "" {
		command = append(command, "--paired")
	}
	for _, v := range []struct {
		flag  string
		value int
	}{{"--clip_R1", o.ClipR1}, {"--clip_R2", o.ClipR2}, {"--three_prime_clip_R1", o.ThreePrimeClipR1}, {"--three_prime_clip_R2", o.ThreePrimeClipR2}, {"--quality", o.Quality}, {"--length", o.Length}} {
		if v.value > 0 {
			command = append(command, v.flag, strconv.Itoa(v.value))
		}
	}
	if o.Adapter != "" {
		command = append(command, "--adapter", o.Adapter)
	}
	if o.Adapter2 != "" {
		command = append(command, "--adapter2", o.Adapter2)
	}
	command = append(command, o.Read1)
	if o.Read2 != "" {
		command = append(command, o.Read2)
	}
	return command
}
