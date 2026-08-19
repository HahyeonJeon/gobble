package assets

// AppendExtraArgs appends extra tokens after command. Extra is argv, not a
// shell string. Gobble does not split extra-args.
func AppendExtraArgs(command, extra []string) []string {
	out := make([]string, 0, len(command)+len(extra))
	out = append(out, command...)
	out = append(out, extra...)
	return out
}
