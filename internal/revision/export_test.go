package revision

// Environment is the environment every git subprocess this package starts is
// run with, reachable from the test package.
//
// It is exported to the suite and to nothing else because the rule it carries
// is about the **call site** rather than about any answer: ADR-0071 requires
// every object read here to run with lazy fetching off, and a case that could
// only see what a read answered could not tell a read that fetched from one
// that did not.
func Environment() []string { return environment() }
