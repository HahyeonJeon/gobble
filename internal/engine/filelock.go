package engine

import "os"

// openLockFile keeps creation separate from the descriptor used for flock.
// Docker Desktop VirtioFS can grant conflicting locks when one descriptor also
// created the file. Reopening restores exclusion with other readers/writers.
// The caller owns the returned descriptor and the lock acquired on it.
func openLockFile(path string, flag int, perm os.FileMode) (*os.File, error) {
	if flag&os.O_CREATE != 0 {
		created, err := os.OpenFile(path, flag, perm)
		if err != nil {
			return nil, err
		}
		if err := created.Close(); err != nil {
			return nil, err
		}
	}
	return os.OpenFile(path, flag&^os.O_CREATE, perm)
}
