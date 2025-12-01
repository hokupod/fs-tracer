package args

// Options holds CLI flags parsed from arguments.
type Options struct {
	Events          bool
	JSON            bool
	SplitAccess     bool
	SandboxSnippet  bool
	DirsOnly        bool
	AllowProcesses  []string
	IgnoreProcesses []string
	IgnorePrefixes  []string
	NoSudo          bool
	Raw             bool
	NoPIDFilter     bool
	IgnoreCWD       bool
	MaxDepth        int
	Command         []string
}
