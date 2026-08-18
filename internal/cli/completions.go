package cli

import (
	"fmt"
	"io"
	"strings"
)

// RunCompletions implements `hyper completions <shell>` — the second of the
// two commands standing outside §9's tree of sixteen, and the first time that
// tree reaches a shell at all (issue #104).
//
// Like RunVersion it is a sibling of RunCheck taking neither the environment
// nor a working directory: §9 exempts it from the version pin gate, and the
// exemption is stated in the signature rather than enforced by a branch, so
// shell setup in a dotfiles bootstrap works before any repository exists
// (ADR-0020). Nothing here reads a file or reaches a network, on any path.
//
// Exactly one positional. Naming nothing is a usage error like it is
// everywhere else, and so is naming two (ADR-0060). The known set is matched
// byte-exact and case-sensitively, consistent with how every other name in
// the tool is matched (§9): `hyper completions BASH` exits 2.
func RunCompletions(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintf(stderr, "hyper completions: %s\n  known shells: %s\n", arityFault("shell", args), strings.Join(shells, ", "))
		return ExitUsage
	}

	script, known := completionScripts[args[0]]
	if !known {
		fmt.Fprintf(stderr, "hyper completions: unknown shell %q\n  known shells: %s\n", args[0], strings.Join(shells, ", "))
		return ExitUsage
	}

	if _, err := io.WriteString(stdout, script); err != nil {
		fmt.Fprintf(stderr, "hyper completions: %s\n", err)
		return ExitUsage
	}
	return ExitClean
}

// completionScripts is the three scripts, one per shell, assembled once from
// the tree in tree.go. They are compiled in: assembly reads nothing but
// package-level lists, so a script is byte-identical on every machine and on
// every invocation, and no framework generates one from a command object at
// run time (issue #104).
//
// Three hand-written scripts rather than one generator's output is the whole
// point — what a shell offers is a design decision per shell, and the only
// thing the three share is the surface they describe.
var completionScripts = map[string]string{
	"bash": bashScript(),
	"fish": fishScript(),
	"zsh":  zshScript(),
}

// preamble is the header every script carries, stating in the file itself the
// decision that shapes all three: a completion is a keypress, and a keypress
// may not run a gated command. Completing an artefact name, a Provider name,
// an Operation name or a Run id at the cursor would mean invoking `hyper`
// behind TAB — which can Refuse 77, can read a repository the user is not in,
// and can block on a slow filesystem. None of that belongs there.
//
// The second paragraph is the inconsistency the design accepts and states: the
// script describes the surface §9 fixes rather than the state of the build, so
// the shell knows `run`, `review` and `records` before those commands exist.
// It closes as the milestones land.
func preamble(shell, install string) string {
	return "" +
		"# " + shell + " completion for hyper(1), written by: hyper completions " + shell + "\n" +
		"#\n" +
		"# The script is a compiled-in constant — the same bytes on every machine and\n" +
		"# every invocation. It never invokes hyper, reads no file of its own, and\n" +
		"# reaches no network: a completion is a keypress, and no artefact name,\n" +
		"# Provider name, Operation name or Run id is ever completed here.\n" +
		"#\n" +
		"# It describes the command surface the specification fixes rather than the\n" +
		"# state of this build, so a name offered below may still answer 'unknown\n" +
		"# command' until the release that carries it.\n" +
		"#\n" +
		"# " + install + "\n"
}

// bashScript is the bash completion, a single function registered with
// complete -F. Its only external calls are compgen and shopt, both bash
// builtins.
func bashScript() string {
	return preamble("bash", "Install: source <(hyper completions bash)") + `
_hyper() {
    local commands='` + words(Commands()) + `'
    local globals='` + words(globals) + `'
    local shells='` + words(shells) + `'
    local store_sub_verbs='` + words(storeSubVerbs) + `'
    local cur prev

    COMPREPLY=()
    cur=${COMP_WORDS[COMP_CWORD]}
    prev=${COMP_WORDS[COMP_CWORD-1]}

    # Position one: the command names, and never anything else.
    if [ "$COMP_CWORD" -eq 1 ]; then
        COMPREPLY=($(compgen -W "$commands" -- "$cur"))
        return
    fi

    # The one path hyper ever completes, and the only reply whose bytes are
    # not hyper's own — which is what the three precautions are for. A
    # directory named "my repo" is one completion and not two (IFS), a
    # directory named "v*" is itself and not what it matches (noglob,
    # restored rather than cleared), and what is inserted at the cursor is
    # quoted for the shell to read back (filenames, where the bash running
    # this is new enough to have compopt).
    #
    # The --repo-dir=DIR spelling is deliberately not completed: offering it
    # would need the whole prefix rewritten into every reply, and the
    # two-word spelling is the one the scripts show.
    if [ "$prev" = '--repo-dir' ]; then
        local reset
        reset=$(shopt -po noglob)
        set -o noglob
        local IFS=$'\n'
        COMPREPLY=($(compgen -d -- "$cur"))
        $reset
        if type compopt >/dev/null 2>&1; then
            compopt -o filenames
        fi
        return
    fi

    case ${COMP_WORDS[1]} in
        store)
            # store's sub-verb at position two, the globals after it.
            if [ "$COMP_CWORD" -eq 2 ]; then
                COMPREPLY=($(compgen -W "$store_sub_verbs $globals" -- "$cur"))
            else
                COMPREPLY=($(compgen -W "$globals" -- "$cur"))
            fi
            ;;
        completions)
            # The three shells, and no flag: this command takes none.
            if [ "$COMP_CWORD" -eq 2 ]; then
                COMPREPLY=($(compgen -W "$shells" -- "$cur"))
            fi
            ;;
        ` + alternation(treeExcept("store")) + `)
            COMPREPLY=($(compgen -W "$globals" -- "$cur"))
            ;;
    esac

    # version and any word that names no command fall through to nothing:
    # version takes no argument at all, and an unknown name has none to offer.
}

complete -F _hyper hyper
`
}

// zshScript is the zsh completion, autoloadable as _hyper on $fpath and
// sourceable as it stands — the funcstack test at the foot is what serves
// both. Its only external calls are compadd and _path_files, which are zsh's
// own.
func zshScript() string {
	return "#compdef hyper\n" +
		preamble("zsh", "Install: save as _hyper on $fpath, or source it after compinit has run") + `
_hyper() {
    # The four lists carry the function's own name as a prefix because zsh
    # gives several of these words to parameters of its own — commands is
    # zsh/parameter's readonly map of every command on $path, and the
    # completion system loads that module.
    local -a _hyper_commands _hyper_globals _hyper_shells _hyper_store_sub_verbs
    _hyper_commands=(` + words(Commands()) + `)
    _hyper_globals=(` + words(globals) + `)
    _hyper_shells=(` + words(shells) + `)
    _hyper_store_sub_verbs=(` + words(storeSubVerbs) + `)

    # Position one: the command names, and never anything else.
    if (( CURRENT == 2 )); then
        compadd -- $_hyper_commands
        return
    fi

    # The one path hyper ever completes. _path_files quotes what it inserts
    # and marks a directory as one, so the precautions bash needs are zsh's
    # own here.
    if [[ ${words[CURRENT-1]} == --repo-dir ]]; then
        _path_files -/
        return
    fi

    case ${words[2]} in
        store)
            # store's sub-verb at position two, the globals after it.
            if (( CURRENT == 3 )); then
                compadd -- $_hyper_store_sub_verbs
            fi
            compadd -- $_hyper_globals
            ;;
        completions)
            # The three shells, and no flag: this command takes none.
            if (( CURRENT == 3 )); then
                compadd -- $_hyper_shells
            fi
            ;;
        ` + alternation(treeExcept("store")) + `)
            compadd -- $_hyper_globals
            ;;
    esac

    # version and any word that names no command fall through to nothing:
    # version takes no argument at all, and an unknown name has none to offer.
}

if [ "$funcstack[1]" = '_hyper' ]; then
    _hyper "$@"
else
    compdef _hyper hyper
fi
`
}

// fishScript is the fish completion: no function of its own, one complete
// line per rule. The conditions and the directory list are fish's own shipped
// helpers, so nothing here starts a process either.
//
// fish's conditions read the whole line rather than a position in it, which
// is the one place the three scripts cannot be written alike:
// __fish_seen_subcommand_from store is true for `hyper store init` as much as
// for `hyper store`. Where bash and zsh compare a cursor position, this
// script says what must not already have been typed — which closes the two
// cases that matter and leaves a name typed as some other command's argument
// still able to trip a rule. That last is a wart rather than a hazard: no
// command here takes a positional that could be one of these words, and
// nothing is completed that could reach the world.
func fishScript() string {
	var b strings.Builder
	b.WriteString(preamble("fish", "Install: hyper completions fish > ~/.config/fish/completions/hyper.fish"))

	b.WriteString(`
# hyper completes no path of its own; --repo-dir's directory below is the one
# exception, so file completion is off everywhere else.
complete -c hyper -f

# Position one: the command names, and never anything else.
complete -c hyper -n __fish_use_subcommand -a '` + words(Commands()) + `'

# store's sub-verb, the whole of the tree's nesting, offered until it is there.
complete -c hyper -n '__fish_seen_subcommand_from store; and not __fish_seen_subcommand_from ` + words(storeSubVerbs) + `' -a '` + words(storeSubVerbs) + `'

# The three shells, and no flag: this command takes none. One is named at
# most once, so the condition drops away as soon as one has been.
complete -c hyper -n '__fish_seen_subcommand_from completions; and not __fish_seen_subcommand_from ` + words(shells) + `' -a '` + words(shells) + `'

# The globals, offered after the tree's commands alone — version and
# completions are outside it and take none.
`)

	seen := "__fish_seen_subcommand_from " + words(tree)
	for _, flag := range globals {
		line := "complete -c hyper -n '" + seen + "' -l " + strings.TrimPrefix(flag, "--")
		if flag == "--repo-dir" {
			line += " -r -a '(__fish_complete_directories)'"
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

// words joins a list the way a shell reads one, which is what every script
// interpolates: bash's compgen -W, zsh's array literal and fish's -a all take
// the same space-separated form. No name in the surface carries whitespace,
// and none can — §9's names are words the glossary defines.
func words(list []string) string {
	return strings.Join(list, " ")
}

// alternation joins a list as a case pattern, the form bash and zsh share.
// Spelling the branch out rather than defaulting to it is what keeps the
// scripts honest at the edge: a word matching no command completes nothing,
// instead of being offered flags for a command that does not exist.
func alternation(list []string) string {
	return strings.Join(list, "|")
}

// treeExcept is the tree minus the one command a script handles in a branch
// of its own, which is `store` — it needs its sub-verb offered before the
// globals.
func treeExcept(handled string) []string {
	rest := make([]string, 0, len(tree))
	for _, name := range tree {
		if name != handled {
			rest = append(rest, name)
		}
	}
	return rest
}
