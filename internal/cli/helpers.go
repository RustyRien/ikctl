package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rustyrien/ikctl/internal/printers"
	"github.com/rustyrien/ikctl/sdk"
	"github.com/rustyrien/ikctl/sdk/registry"
)

// lookupKind resolves a kind argument, optionally requiring a verb.
func lookupKind(arg string, verb registry.Verb) (registry.Kind, error) {
	k, ok := registry.Lookup(arg)
	if !ok {
		names := make([]string, 0)
		for _, k := range registry.All() {
			names = append(names, k.Name)
		}
		return registry.Kind{}, usageErrorf("unknown kind %q; valid kinds: %s (see 'ikctl api-resources')", arg, strings.Join(names, ", "))
	}
	if verb != "" && !k.Supports(verb) {
		return registry.Kind{}, usageError{&sdk.UnsupportedError{Kind: k.Name, Verb: string(verb)}}
	}
	return k, nil
}

// completeKinds offers kind names for shell completion of the first argument.
func completeKinds(verb registry.Verb) cobra.CompletionFunc {
	return func(_ *cobra.Command, args []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		var out []cobra.Completion
		for _, k := range registry.All() {
			if verb == "" || k.Supports(verb) {
				out = append(out, k.Name, k.Plural)
			}
		}
		return out, cobra.ShellCompDirectiveNoFileComp
	}
}

// addOutputFlag registers -o/--output.
func addOutputFlag(cmd *cobra.Command, p *string, def string, formats ...string) {
	cmd.Flags().StringVarP(p, "output", "o", def, "output format: "+strings.Join(formats, "|"))
	_ = cmd.RegisterFlagCompletionFunc("output", cobra.FixedCompletions(formats, cobra.ShellCompDirectiveNoFileComp))
}

// parseOutput parses -o and restricts it to the allowed formats.
func parseOutput(s string, allowed ...string) (printers.Output, error) {
	out, err := printers.ParseOutput(s)
	if err != nil {
		return out, usageError{err}
	}
	if len(allowed) > 0 && !slices.Contains(allowed, out.Format) {
		return out, usageErrorf("output format %q is not supported by this command (allowed: %s)", s, strings.Join(allowed, ", "))
	}
	return out, nil
}

// objectRef renders "kind/name" (or "kind/id") for messages.
func objectRef(k registry.Kind, obj sdk.Object) string {
	if k.NamePath != "" {
		if n := obj.String(k.NamePath); n != "" {
			return k.Name + "/" + n
		}
	}
	return k.Name + "/" + obj.ID()
}

// refFields is the minimal selection needed for objectRef.
func refFields(k registry.Kind) string {
	if k.NamePath != "" && k.NamePath != "id" {
		return "id " + k.NamePath
	}
	return "id"
}

// confirm asks a yes/no question on the terminal. In non-interactive mode it
// refuses unless the caller passed --yes.
func confirm(f *Factory, yes bool, prompt string) error {
	if yes {
		return nil
	}
	if !f.IO.InTTY {
		return usageErrorf("refusing to %s without confirmation in non-interactive mode; pass --yes", prompt)
	}
	fmt.Fprintf(f.IO.ErrOut, "Are you sure you want to %s? [y/N]: ", prompt)
	line, err := bufio.NewReader(f.IO.In).ReadString('\n')
	if err != nil && err != io.EOF {
		return err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return nil
	default:
		return fmt.Errorf("aborted")
	}
}

// parseValue interprets a command-line value: JSON literals (numbers, booleans,
// null, arrays, objects, quoted strings) are decoded, anything else is a string.
func parseValue(s string) any {
	var v any
	if err := json.Unmarshal([]byte(s), &v); err == nil {
		return v
	}
	return s
}

// parseKeyValues parses repeated key=value flags.
func parseKeyValues(pairs []string, flag string) (map[string]any, error) {
	out := map[string]any{}
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		if !ok || k == "" {
			return nil, usageErrorf("invalid --%s %q: expected key=value", flag, p)
		}
		out[k] = parseValue(v)
	}
	return out, nil
}
