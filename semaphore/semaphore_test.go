package semaphore

import (
	"testing"
)

func TestSemaphore_AcquireUnfairCounterSemaphore(t *testing.T) {
	sema := NewSemaphore()
	id1, err1 := sema.AcquireUnfairCounterSemaphore("hello", 2)
	if id1 == "" || err1 != nil {
		t.Error(err1)
	}
	id2, err2 := sema.AcquireUnfairCounterSemaphore("hello", 2)
	if id2 == "" || err2 != nil {
		t.Error(err2)
	}
	id3, err3 := sema.AcquireUnfairCounterSemaphore("hello", 2)
	if id3 != "" {
		t.Error(err3)
	}
}
