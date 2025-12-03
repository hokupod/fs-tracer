//go:build darwin

package procinfo

import (
	"fmt"
	"syscall"
	"unsafe"
)

// Constants from <libproc.h>
const (
	procInfoCallPidInfo = 2 // PROC_INFO_CALL_PIDINFO
	procPidListThreads  = 3 // PROC_PIDLISTTHREADS
	threadIDSize        = int(unsafe.Sizeof(uint64(0)))
	sysProcInfo         = uintptr(syscall.SYS_PROC_INFO)
)

// ListThreads returns thread handles (as returned by PROC_PIDLISTTHREADS).
// fs_usage emits these handles as the trailing numeric ID (ruby.<handle>). On
// recent macOS, PROC_PIDTHREADINFO often returns ESRCH for non-system targets,
// so we rely on the handles alone, which are stable for the lifetime of the
// process.
// Requires appropriate privileges (root for other users' processes).
func ListThreads(pid int) ([]uint64, error) {
	return listThreadHandles(pid)
}

func listThreadHandles(pid int) ([]uint64, error) {
	// Start with room for 256 threads and grow as needed.
	size := 256
	for {
		buf := make([]uint64, size)
		nbytes, err := callProcInfoListThreads(pid, buf)
		if err != nil {
			return nil, err
		}
		if nbytes == len(buf)*threadIDSize {
			size *= 2
			continue
		}
		count := nbytes / threadIDSize
		return buf[:count], nil
	}
}

func callProcInfoListThreads(pid int, buf []uint64) (int, error) {
	if len(buf) == 0 {
		buf = make([]uint64, 1)
	}
	nbytes, _, errno := syscall.Syscall6(
		sysProcInfo,
		uintptr(procInfoCallPidInfo),
		uintptr(pid),
		uintptr(procPidListThreads),
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)*threadIDSize),
	)
	if errno != 0 {
		return 0, fmt.Errorf("proc_info list threads: %w", errno)
	}
	return int(nbytes), nil
}
