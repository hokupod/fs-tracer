package fsusage

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// FsUsageRunner abstracts fs_usage invocation for production and tests.
type FsUsageRunner interface {
	Run(pid int, comm string) (io.ReadCloser, error)
}

// SudoFsUsageRunner runs fs_usage via sudo (default) or directly (--no-sudo).
type SudoFsUsageRunner struct {
	NoSudo bool
	All    bool
}

func (r SudoFsUsageRunner) Run(pid int, comm string) (io.ReadCloser, error) {
	// Use both filesys and pathname to capture open/stat plus path resolution.
	cmdArgs := []string{"fs_usage", "-w", "-f", "filesys,pathname"}
	if !r.All && pid > 0 {
		cmdArgs = append(cmdArgs, fmt.Sprintf("%d", pid))
	}
	if !r.NoSudo {
		cmdArgs = append([]string{"sudo"}, cmdArgs...)
	}
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	// Allow sudo to prompt for password when needed.
	cmd.Stdin = os.Stdin
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = os.Stderr
	if os.Getenv("FS_TRACER_DEBUG") != "" {
		fmt.Fprintln(os.Stderr, "debug: fs_usage cmd:", strings.Join(cmdArgs, " "))
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &cmdReadCloser{rc: stdout, cmd: cmd}, nil
}

type cmdReadCloser struct {
	rc  io.ReadCloser
	cmd *exec.Cmd
}

func (c *cmdReadCloser) Read(p []byte) (int, error) {
	return c.rc.Read(p)
}

func (c *cmdReadCloser) Close() error {
	_ = c.rc.Close()
	if c.cmd.Process != nil && c.cmd.ProcessState == nil {
		// Ask fs_usage to terminate gracefully to flush any buffered output.
		_ = c.cmd.Process.Signal(syscall.SIGINT)
	}
	return c.cmd.Wait()
}
