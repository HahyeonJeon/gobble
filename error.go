package gobble

import "strconv"

// DefectCode is a stable machine-readable failure class.
type DefectCode string

const (
	// DefectCycle means the composed graph contains a directed cycle.
	DefectCycle DefectCode = "cycle"
	// DefectMissingInput means a required input bind or input port is missing.
	DefectMissingInput DefectCode = "missing-input"
	// DefectMissingOutput means a required output bind or output port is missing.
	DefectMissingOutput DefectCode = "missing-output"
	// DefectMissingCommand means a task has an empty command.
	DefectMissingCommand DefectCode = "missing-command"
	// DefectInvalidName means a name is empty, duplicate, or illegally spelled.
	DefectInvalidName DefectCode = "invalid-name"
	// DefectInvalidPath means a PathSpec cannot be rendered or used.
	DefectInvalidPath DefectCode = "invalid-path"
	// DefectConflict means two binds render to the same path.
	DefectConflict DefectCode = "conflict"
	// DefectUnsupportedBackend means a task names a backend other than local.
	DefectUnsupportedBackend DefectCode = "unsupported-backend"
)

// Defect is one named failure inside an [Error].
type Defect struct {
	// Code is the machine-readable class of this defect.
	Code DefectCode `json:"code"`
	// Unit is the graph id or bind that failed.
	Unit string `json:"unit"`
	// Message is the human-readable explanation.
	Message string `json:"message"`
	// Paths are authored or rendered paths involved in the defect.
	Paths []string `json:"paths"`
}

// Error is a structured failure from compose, validate, plan, or render.
// Callers inspect it with errors.As. JSON keys are op and defects.
type Error struct {
	// Op is the failing operation: compose, validate, plan, or render.
	Op string `json:"op"`
	// Defects is the list of named failures.
	Defects []Defect `json:"defects"`
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	switch len(e.Defects) {
	case 0:
		if e.Op == "" {
			return "failed"
		}
		return e.Op + " failed"
	case 1:
		msg := e.Defects[0].Message
		if msg == "" {
			msg = string(e.Defects[0].Code)
		}
		if e.Op == "" {
			return msg
		}
		return e.Op + ": " + msg
	default:
		n := strconv.Itoa(len(e.Defects))
		if e.Op == "" {
			return n + " defects"
		}
		return e.Op + ": " + n + " defects"
	}
}

func renderInvalid(message string, paths ...string) *Error {
	return &Error{
		Op: "render",
		Defects: []Defect{{
			Code:    DefectInvalidPath,
			Message: message,
			Paths:   paths,
		}},
	}
}
