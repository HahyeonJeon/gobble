// Package modules defines the shared contract for first-party command modules.
//
// Each direct child package owns exactly one executable command or named
// subcommand and records exactly one Gobble task. A child package exists only
// when a supported pipeline needs that command. Module packages do not own
// assay stage order, sample expansion, joins, reference policy, downloads, or
// engine controls.
//
// A child package declares a command-specific Options struct that embeds
// [Options]. Its named fields own common and safety-relevant flags. It also
// declares a command-specific Ports struct whose named gobble.Handle fields
// correspond to declared File, gobble.Group, or gobble.Tree outputs. There
// is intentionally no generic Ports map: command-specific names and artifact
// kinds are part of the supported contract. Adders copy Options before
// retaining mutable values.
//
// A module adder accepts existing input handles and a [Parent], adds no
// pipeline input, and returns its Ports and any option error. Its standalone
// adapter creates explicit inputs and calls the same adder. Invalid options,
// image pins, and known flag collisions return a structured compose error;
// callers record it with gobble.Pipeline.RecordComposeError. Module code does
// not panic for authored paths or caller values. A command failure remains
// contained to the one recorded Gobble task.
//
// ExtraArgs are argv tokens, never a shell string. Each child documents their
// exact insertion point and passes its active named flags to [AppendExtraArgs].
// The final argv, image, resources, binds, parameters, and destinations remain
// visible in the Gobble plan and normal resume identity.
//
// Each module default uses one constant [Image] selected by its product's
// dated benchmark and benchmark-informed task resources. A caller-supplied
// replacement must also be digest-pinned, changes task identity, and is outside
// the behavioral support claim for the default image.
//
// Command packages moved at the graph-stable checkpoint retain their pre-lift
// option shapes, image strings, unchecked ExtraArgs, and authored-path panic
// behavior through explicitly named compatibility helpers in this package.
// That exception exists only to preserve graph identity during the ownership
// move. Each later product lift adopts the complete contract above as a
// separate graph generation.
package modules
