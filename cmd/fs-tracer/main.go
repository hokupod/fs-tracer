package main

import (
	"fmt"
	"os"

	"github.com/carapace-sh/carapace"
	"github.com/hokupod/fs-tracer/internal/app"
	"github.com/hokupod/fs-tracer/internal/args"
	"github.com/spf13/cobra"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(90)
	}
}

func newRootCmd() *cobra.Command {
	var (
		optEvents       bool
		optJSON         bool
		optSplitAccess  bool
		optSandbox      bool
		optDirs         bool
		optAllowProc    []string
		optIgnoreProc   []string
		optIgnorePrefix []string
		optNoSudo       bool
		optRaw          bool
		optNoPIDFilter  bool
		optIgnoreCWD    bool
		optMaxDepth     int
	)

	rootCmd := &cobra.Command{
		Use:   "fs-tracer [OPTIONS] -- yourcmd [ARG ...]",
		Short: "Trace filesystem accesses of a command via fs_usage",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("yourcmd is required after --")
			}
			if optSandbox && optEvents {
				return fmt.Errorf("--events cannot be used with --sandbox-snippet")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, positional []string) error {
			opts := args.Options{
				Events:          optEvents,
				JSON:            optJSON,
				SplitAccess:     optSplitAccess,
				SandboxSnippet:  optSandbox,
				DirsOnly:        optDirs,
				AllowProcesses:  optAllowProc,
				IgnoreProcesses: optIgnoreProc,
				IgnorePrefixes:  optIgnorePrefix,
				NoSudo:          optNoSudo,
				Raw:             optRaw,
				NoPIDFilter:     optNoPIDFilter,
				IgnoreCWD:       optIgnoreCWD,
				MaxDepth:        optMaxDepth,
				Command:         append([]string(nil), positional...),
			}
			code := app.Run(app.Config{Options: opts})
			os.Exit(code)
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	flags := rootCmd.Flags()
	flags.BoolVarP(&optEvents, "events", "v", false, "emit detailed event log")
	flags.BoolVar(&optJSON, "json", false, "output JSON")
	flags.BoolVar(&optSplitAccess, "split-access", false, "separate read/write sets")
	flags.BoolVar(&optSandbox, "sandbox-snippet", false, "emit sandbox-exec s-expressions (exclusive with --events)")
	flags.BoolVar(&optDirs, "dirs", false, "emit parent directories only")
	flags.StringSliceVar(&optAllowProc, "allow-process", nil, "only include events from process name (repeatable)")
	flags.StringSliceVar(&optIgnoreProc, "ignore-process", nil, "process name to ignore (repeatable)")
	flags.StringSliceVar(&optIgnorePrefix, "ignore-prefix", nil, "path prefix to ignore (repeatable)")
	flags.BoolVar(&optNoSudo, "no-sudo", false, "run fs_usage without sudo")
	flags.BoolVar(&optRaw, "raw", false, "disable ignore filters")
	flags.BoolVar(&optNoPIDFilter, "no-pid-filter", false, "do not restrict events to target PID")
	flags.BoolVar(&optIgnoreCWD, "ignore-cwd", false, "ignore events under current working directory")
	flags.IntVar(&optMaxDepth, "max-depth", 0, "truncate paths to at most N components (0 = unlimited)")

	carapace.Gen(rootCmd).Standalone()
	carapace.Gen(rootCmd).FlagCompletion(carapace.ActionMap{
		"allow-process":  carapace.ActionValues(), // no-op completion placeholder
		"ignore-process": carapace.ActionValues(),
		"ignore-prefix":  carapace.ActionDirectories(),
	})
	// Positional: suggest executables, then files/dirs.
	carapace.Gen(rootCmd).PositionalCompletion(
		carapace.ActionExecutables(),
	)
	carapace.Gen(rootCmd).PositionalAnyCompletion(
		carapace.ActionFiles().Chdir("."),
	)

	rootCmd.AddCommand(newCompletionCmd(rootCmd))
	return rootCmd
}

func newCompletionCmd(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Args:  cobra.ExactValidArgs(1),
		ValidArgs: []string{
			"bash", "zsh", "fish", "powershell",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return root.GenBashCompletion(os.Stdout)
			case "zsh":
				return root.GenZshCompletion(os.Stdout)
			case "fish":
				return root.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return root.GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
		},
	}
}
