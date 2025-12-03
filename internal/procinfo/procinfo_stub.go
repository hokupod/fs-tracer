//go:build !darwin

package procinfo

import "fmt"

func ListThreads(pid int) ([]uint64, error) {
    return nil, fmt.Errorf("ListThreads is supported only on darwin")
}

