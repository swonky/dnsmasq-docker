package main

import (
	"iter"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"
	"syscall"
)

const (
	prefix     = "DNSMASQ_"
	dnsmasqBin = "dnsmasq"
)

// convertToOption converts a DNSMASQ_-prefixed environment variable name into
// a dnsmasq CLI flag. The prefix is stripped, underscores are replaced with
// hyphens, and the result is lowercased and prepended with "--".
// Example: DNSMASQ_NO_RESOLV → --no-resolv
func convertToOption(s string) string {
	return "--" + strings.ToLower(
		strings.ReplaceAll(
			strings.TrimPrefix(s, prefix),
			"_", "-",
		),
	)
}

// getEnv returns an iterator over environment variables that match the given
// prefix. Each yielded pair is (key, value), where value has been normalised:
// truthy values ("true", "yes", "on") are yielded as an empty string so they
// map to bare CLI flags; falsy values ("false", "no", "off") are skipped
// entirely.
func getEnv(prefix string) iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		for _, env := range os.Environ() {
			if !strings.HasPrefix(env, prefix) {
				continue
			}
			parts := strings.SplitN(env, "=", 2)
			if len(parts) != 2 {
				continue
			}
			value := parts[1]
			switch strings.ToLower(value) {
			case "true", "yes", "on":
				value = ""
			case "false", "no", "off":
				continue
			}
			if !yield(parts[0], value) {
				return
			}
		}
	}
}

// formatArgument converts a key/value pair into a dnsmasq CLI argument string.
// If value is empty the flag is returned bare (e.g. "--no-resolv"); otherwise
// key and value are joined with "=" (e.g. "--listen-address=127.0.0.1").
func formatArgument(key, value string) string {
	key = convertToOption(key)
	if value == "" {
		return key
	}
	return key + "=" + value
}

// logArguments writes the resolved dnsmasq arguments to stderr in a compact,
// human-readable format, one flag per line, sorted for deterministic output.
func logArguments(args []string) {
	log.Printf("starting dnsmasq with %d option(s):", len(args))
	for _, arg := range args {
		log.Printf("  %s", arg)
	}
}

func main() {
	log.SetPrefix("[dnsmasq-init] ")
	log.SetFlags(0)

	bin, err := exec.LookPath(dnsmasqBin)
	if err != nil {
		log.Fatalf("could not find dnsmasq binary: %v", err)
	}

	args := make([]string, 0)
	for k, v := range getEnv(prefix) {
		args = append(args, formatArgument(k, v))
	}
	sort.Strings(args)

	logArguments(args)

	// Replace the current process with dnsmasq. argv[0] must be the binary
	// path itself, followed by the actual arguments.
	argv := append([]string{bin}, args...)
	if err := syscall.Exec(bin, argv, os.Environ()); err != nil {
		log.Fatalf("exec failed: %v", err)
	}
}
