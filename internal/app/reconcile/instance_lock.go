package reconcile

import (
	"fmt"
	"os"
	"syscall"
)

type InstanceLock struct{ file *os.File }

func AcquireInstanceLock(path string) (*InstanceLock, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		return nil, fmt.Errorf("another netlab instance owns %s: %w", path, err)
	}
	return &InstanceLock{file: file}, nil
}
func (l *InstanceLock) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	_ = syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	return l.file.Close()
}
