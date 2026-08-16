// Package version names the version of the running binary — the fact the
// version pin gate compares against the Repository declaration's pin (§11,
// ADR-0020).
package version

// Version is the running binary's version. It is a placeholder until
// `hyper project`'s release machinery lands (milestone 10) and starts
// stamping it at build time via -ldflags.
const Version = "0.0.0-dev"
