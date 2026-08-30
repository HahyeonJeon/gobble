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

// AppendExtraArgs copies command and appends copied extraArgs. namedFlags are
// the flags already owned by active named options at this command-specific
// insertion point. An exact flag token or --flag=value form in extraArgs
// returns a structured compose defect instead of creating order-dependent
// argv. A child module remains responsible for any additional collision forms
// defined by its command.
func AppendExtraArgs(unit string, command, extraArgs, namedFlags []string) ([]string, error) {
	for _, arg := range extraArgs {
		for _, flag := range namedFlags {
			if flag == "" {
				continue
			}
			if arg == flag || strings.HasPrefix(arg, flag+"=") {
				return nil, composeInvalidValue(unit, "ExtraArgs conflicts with named option "+flag)
			}
		}
	}
	out := make([]string, 0, len(command)+len(extraArgs))
	out = append(out, command...)
	out = append(out, extraArgs...)
	return out, nil
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
