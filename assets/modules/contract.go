package modules

import (
	"net"
	"strings"
	"unicode"

	"github.com/HahyeonJeon/gobble"
)

// Parent records the one task owned by a command module.
type Parent interface {
	AddTask(spec gobble.TaskSpec) *gobble.Task
}

// Input describes one input to a command module's standalone adapter.
// A non-nil Group records AddInputGroup. A present Tree records AddInputTree.
// Otherwise Standalone records Spec with AddInput.
type Input struct {
	Name  string
	Spec  gobble.PathSpec
	Group gobble.Group
	Tree  gobble.Tree
}

// Standalone creates a pipeline from explicit inputs and delegates task
// construction to build. It is the compatibility adapter used by modules at
// the graph-stable migration checkpoint.
func Standalone(name string, inputs []Input, build func(Parent, []gobble.Handle)) *gobble.Pipeline {
	p := gobble.NewPipeline(name)
	handles := make([]gobble.Handle, len(inputs))
	for i, in := range inputs {
		switch {
		case in.Group != nil:
			handles[i] = p.AddInputGroup(in.Name, in.Group)
		case !in.Tree.IsZero():
			handles[i] = p.AddInputTree(in.Name, in.Tree)
		default:
			handles[i] = p.AddInput(in.Name, in.Spec)
		}
	}
	build(p, handles)
	return p
}

// StandaloneChecked creates a standalone module pipeline and records any
// command-specific option or path error as a compose defect. Lifted modules
// use this form so expected input failures never panic.
func StandaloneChecked(name string, inputs []Input, build func(Parent, []gobble.Handle) error) *gobble.Pipeline {
	p := gobble.NewPipeline(name)
	handles := make([]gobble.Handle, len(inputs))
	for i, in := range inputs {
		switch {
		case in.Group != nil:
			handles[i] = p.AddInputGroup(in.Name, in.Group)
		case !in.Tree.IsZero():
			handles[i] = p.AddInputTree(in.Name, in.Tree)
		default:
			handles[i] = p.AddInput(in.Name, in.Spec)
		}
	}
	if err := build(p, handles); err != nil {
		p.RecordComposeError(err)
	}
	return p
}

// CommandPath renders spec as one command token.
func CommandPath(spec gobble.PathSpec) (string, error) {
	return spec.Render()
}

// HandlePath renders a regular-file handle for unit. Invalid authored paths
// become compose-time invalid-path defects owned by the command module.
func HandlePath(unit string, handle gobble.Handle) (string, error) {
	path, err := CommandPath(handle.Spec())
	if err != nil {
		return "", ComposeDefect(gobble.DefectInvalidPath, unit, "command input path is invalid")
	}
	return path, nil
}

// MustCommandPath preserves the pre-migration authored-path behavior. Lifted
// product builders replace this compatibility helper with structured defects.
func MustCommandPath(spec gobble.PathSpec) string {
	path, err := CommandPath(spec)
	if err != nil {
		panic(err)
	}
	return path
}

// AppendLegacyExtraArgs preserves the pre-migration argv position without
// applying the lifted module collision policy.
func AppendLegacyExtraArgs(command, extra []string) []string {
	out := make([]string, 0, len(command)+len(extra))
	out = append(out, command...)
	out = append(out, extra...)
	return out
}

// ThreadCount returns the historical integer thread count for a CPU request.
func ThreadCount(cpu float64) int {
	if cpu < 1 {
		return 0
	}
	return int(cpu)
}

// Image is a complete immutable container reference in
// registry/repository:tag@sha256:digest form. Module defaults declare Image as
// constants so a tag and its resolved content digest are one source fact.
type Image string

// TaskReference returns the pinned image string for a Gobble task. Invalid or
// mutable references return a structured compose defect assigned to unit.
func (i Image) TaskReference(unit string) (string, error) {
	image := string(i)
	if message := invalidImageMessage(image); message != "" {
		return "", composeInvalidValue(unit, message)
	}
	return image, nil
}

// Options contains the fields common to every command-specific Options type.
// Image, Resources, and ExtraArgs are analysis and task-build policy. They are
// not samplesheet data or engine run controls.
type Options struct {
	Image     Image
	Resources gobble.Resources
	ExtraArgs []string
}

// Clone returns an Options value that does not alias ExtraArgs.
func (o Options) Clone() Options {
	out := o
	out.ExtraArgs = append([]string(nil), o.ExtraArgs...)
	return out
}

// ResolveOptions applies one command's immutable image and resource defaults,
// validates the selected image, and appends ExtraArgs after collision checks.
// The returned values do not alias caller-owned slices.
func ResolveOptions(unit string, options Options, defaultImage Image, defaultResources gobble.Resources, command, namedFlags []string) ([]string, string, gobble.Resources, error) {
	options = options.Clone()
	image := options.Image
	if image == "" {
		image = defaultImage
	}
	imageRef, err := image.TaskReference(unit)
	if err != nil {
		return nil, "", gobble.Resources{}, err
	}
	resources := options.Resources
	if resources.CPU == 0 && resources.Memory == "" {
		resources = defaultResources
	}
	resolved, err := AppendExtraArgs(unit, command, options.ExtraArgs, namedFlags)
	if err != nil {
		return nil, "", gobble.Resources{}, err
	}
	return resolved, imageRef, resources, nil
}

// ShellRedirect renders one argv command followed only by a stdout file
// redirection. Every token is single-quoted, so ExtraArgs remain argv tokens
// and cannot introduce shell syntax. Use it only for tools whose documented
// output contract is stdout.
func ShellRedirect(command []string, output string) string {
	return ShellCommand(command) + " > " + ShellQuote(output)
}

// ShellCommand renders argv as one shell-safe command. It is used only when a
// task must select a typed runtime variant before invoking the owned tool.
func ShellCommand(command []string) string {
	quoted := make([]string, 0, len(command))
	for _, token := range command {
		quoted = append(quoted, ShellQuote(token))
	}
	return strings.Join(quoted, " ")
}

// ShellQuote renders one literal POSIX shell word.
func ShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

// StrandedCommand renders a runtime dispatch over the only three typed RNA
// strandedness values produced by inference.
func StrandedCommand(strandednessPath string, unstranded, forward, reverse []string) string {
	return "strand=$(cat " + ShellQuote(strandednessPath) + ")\n" +
		"case \"$strand\" in\n" +
		"  unstranded) " + ShellCommand(unstranded) + " ;;\n" +
		"  forward) " + ShellCommand(forward) + " ;;\n" +
		"  reverse) " + ShellCommand(reverse) + " ;;\n" +
		"  *) echo \"invalid inferred strandedness: $strand\" >&2; exit 2 ;;\n" +
		"esac"
}

// ComposeDefect returns one structured command-module compose defect.
func ComposeDefect(code gobble.DefectCode, unit, message string, paths ...string) *gobble.Error {
	return &gobble.Error{
		Op: "compose",
		Defects: []gobble.Defect{{
			Code:    code,
			Unit:    unit,
			Message: message,
			Paths:   append([]string(nil), paths...),
		}},
	}
}

// AppendExtraArgs copies command and appends copied extraArgs. namedFlags are
// the flags already owned by active named options at this command-specific
// insertion point. Exact flag tokens, --flag=value forms, and short options
// with attached values return a structured compose defect instead of creating
// order-dependent argv. A child module remains responsible for any additional
// collision forms defined by its command.
func AppendExtraArgs(unit string, command, extraArgs, namedFlags []string) ([]string, error) {
	for _, arg := range extraArgs {
		for _, flag := range namedFlags {
			if flag == "" {
				continue
			}
			if extraArgOwnsFlag(arg, flag) {
				return nil, composeInvalidValue(unit, "ExtraArgs conflicts with named option "+flag)
			}
		}
	}
	out := make([]string, 0, len(command)+len(extraArgs))
	out = append(out, command...)
	out = append(out, extraArgs...)
	return out, nil
}

// RejectExtraArgs rejects command tokens that would select unsupported
// behavior. Exact flags, --flag=value forms, and short options with attached
// values are recognized. Command modules use this before AppendExtraArgs when
// a flag would change the owned command's product meaning rather than customize
// that command.
func RejectExtraArgs(unit string, extraArgs, forbiddenFlags []string) error {
	for _, arg := range extraArgs {
		for _, flag := range forbiddenFlags {
			if extraArgOwnsFlag(arg, flag) {
				return composeInvalidValue(unit, "ExtraArgs contains unsupported option "+flag)
			}
		}
	}
	return nil
}

// RejectExtraArgPrefixes rejects command tokens that select protected options
// for a command parser with case-insensitive long-option abbreviation. Every
// non-empty long-option prefix is recognized. Exact and attached short options
// use the same rules as RejectExtraArgs.
func RejectExtraArgPrefixes(unit string, extraArgs, protectedFlags []string) error {
	if flag := MatchProtectedExtraArg(extraArgs, protectedFlags); flag != "" {
		return composeInvalidValue(unit, "ExtraArgs contains protected option "+flag)
	}
	return nil
}

// MatchProtectedExtraArg returns the first canonical protected option selected
// by ExtraArgs under RejectExtraArgPrefixes rules, or an empty string.
func MatchProtectedExtraArg(extraArgs, protectedFlags []string) string {
	for _, arg := range extraArgs {
		for _, flag := range protectedFlags {
			if extraArgSelectsPrefix(arg, flag) {
				return flag
			}
		}
	}
	return ""
}

func extraArgOwnsFlag(arg, flag string) bool {
	if arg == flag || strings.HasPrefix(arg, flag+"=") {
		return true
	}
	return len(flag) == 2 && flag[0] == '-' && flag[1] != '-' && len(arg) > len(flag) && strings.HasPrefix(arg, flag)
}

func extraArgSelectsPrefix(arg, flag string) bool {
	if extraArgOwnsFlag(arg, flag) {
		return true
	}
	name, _, _ := strings.Cut(arg, "=")
	return strings.HasPrefix(flag, "--") &&
		strings.HasPrefix(name, "--") &&
		len(name) > 2 && len(name) <= len(flag) &&
		strings.EqualFold(name, flag[:len(name)])
}

const (
	maxRepositoryNameLength = 255
	maxTagLength            = 128
)

func invalidImageMessage(image string) string {
	if image == "" {
		return "empty module image"
	}
	if strings.IndexFunc(image, unicode.IsSpace) >= 0 {
		return "module image contains whitespace"
	}
	if strings.Count(image, "@") != 1 {
		return "module image must contain one content digest"
	}
	reference, digest, _ := strings.Cut(image, "@")
	slash := strings.IndexByte(reference, '/')
	if slash <= 0 {
		return "module image must name an explicit registry"
	}
	registry := reference[:slash]
	if !validRegistry(registry) {
		return "module image must name a valid explicit registry"
	}
	repositoryAndTag := reference[slash+1:]
	tagSeparator := strings.LastIndexByte(repositoryAndTag, ':')
	if tagSeparator < 0 || tagSeparator == len(repositoryAndTag)-1 {
		return "module image must include an explicit tag"
	}
	repository := repositoryAndTag[:tagSeparator]
	if len(registry)+1+len(repository) > maxRepositoryNameLength || !validRepository(repository) {
		return "module image must name a valid repository"
	}
	tag := repositoryAndTag[tagSeparator+1:]
	if !validTag(tag) {
		return "module image must include a valid explicit tag"
	}
	if tag == "latest" {
		return "module image tag latest is mutable"
	}
	if !validSHA256(digest) {
		return "module image must include a sha256 content digest"
	}
	return ""
}

func validRegistry(registry string) bool {
	host := registry
	hasPort := false

	if strings.HasPrefix(registry, "[") {
		closingBracket := strings.IndexByte(registry, ']')
		if closingBracket < 0 {
			return false
		}
		address := registry[1:closingBracket]
		if !strings.Contains(address, ":") || net.ParseIP(address) == nil {
			return false
		}
		host = registry[:closingBracket+1]
		remainder := registry[closingBracket+1:]
		if remainder != "" {
			if remainder[0] != ':' || !validPort(remainder[1:]) {
				return false
			}
		}
		return true
	}

	if strings.Count(registry, ":") > 1 {
		return false
	}
	if colon := strings.LastIndexByte(registry, ':'); colon >= 0 {
		host = registry[:colon]
		if !validPort(registry[colon+1:]) {
			return false
		}
		hasPort = true
	}
	if !validDomain(host) {
		return false
	}
	return host == "localhost" || strings.Contains(host, ".") || hasPort
}

func validPort(port string) bool {
	if port == "" {
		return false
	}
	for i := range len(port) {
		if port[i] < '0' || port[i] > '9' {
			return false
		}
	}
	return true
}

func validDomain(host string) bool {
	if host == "" {
		return false
	}
	for component := range strings.SplitSeq(host, ".") {
		if component == "" || !isAlphaNumeric(component[0]) || !isAlphaNumeric(component[len(component)-1]) {
			return false
		}
		for i := 1; i < len(component)-1; i++ {
			if !isAlphaNumeric(component[i]) && component[i] != '-' {
				return false
			}
		}
	}
	return true
}

func validRepository(repository string) bool {
	if repository == "" {
		return false
	}
	for component := range strings.SplitSeq(repository, "/") {
		if !validRepositoryComponent(component) {
			return false
		}
	}
	return true
}

func validRepositoryComponent(component string) bool {
	if component == "" {
		return false
	}
	for i := 0; i < len(component); {
		if !isLowerAlphaNumeric(component[i]) {
			return false
		}
		for i < len(component) && isLowerAlphaNumeric(component[i]) {
			i++
		}
		if i == len(component) {
			return true
		}

		switch component[i] {
		case '.', '_':
			separator := component[i]
			i++
			if separator == '_' && i < len(component) && component[i] == '_' {
				i++
			}
		case '-':
			for i < len(component) && component[i] == '-' {
				i++
			}
		default:
			return false
		}
		if i == len(component) || !isLowerAlphaNumeric(component[i]) {
			return false
		}
	}
	return true
}

func validTag(tag string) bool {
	if len(tag) == 0 || len(tag) > maxTagLength || (!isAlphaNumeric(tag[0]) && tag[0] != '_') {
		return false
	}
	for i := 1; i < len(tag); i++ {
		if !isAlphaNumeric(tag[i]) && tag[i] != '_' && tag[i] != '.' && tag[i] != '-' {
			return false
		}
	}
	return true
}

func isLowerAlphaNumeric(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= '0' && b <= '9'
}

func isAlphaNumeric(b byte) bool {
	return isLowerAlphaNumeric(b) || b >= 'A' && b <= 'Z'
}

func validSHA256(digest string) bool {
	const prefix = "sha256:"
	if !strings.HasPrefix(digest, prefix) || len(digest) != len(prefix)+64 {
		return false
	}
	for _, r := range digest[len(prefix):] {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

func composeInvalidValue(unit, message string) *gobble.Error {
	return &gobble.Error{
		Op: "compose",
		Defects: []gobble.Defect{{
			Code:    gobble.DefectInvalidValue,
			Unit:    unit,
			Message: message,
		}},
	}
}
