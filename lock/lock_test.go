package lock

import (
	"testing"
	"time"
)

func TestLocker(t *testing.T) {
	locker := NewLocker()
	id1, isLock1 := locker.AcquireLock("hello", 2*time.Second)
	if isLock1 == false {
		t.Error("err")
	}
	_, isLock2 := locker.AcquireLock("hello", 2*time.Second)
	if isLock2 {
		t.Error("err")
	}
	locker.ReleaseLock("hello", id1)
	_, isLock3 := locker.AcquireLock("hello", 2*time.Second)
	if isLock3 == false {
		t.Error("err")
	}
}
