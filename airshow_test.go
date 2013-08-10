package airshow

import (
	"testing"
)

type TestWorker struct {
	getImageCallCount int
}

func (t *TestWorker) GetImage() ([]byte, error) {
	t.getImageCallCount += 1
	return nil, nil
}

func TestAddWorker(t *testing.T) {
	a := New()
	if len(a.workers) != 0 {
		t.Error("should workers length is 0")
	}

	worker := &TestWorker{}

	a.AddWorker(worker)

	if len(a.workers) != 1 {
		t.Error("should workers length is 0")
	}
}

func TestgetImageFromWorker(t *testing.T) {
	a := New()

	t1 := &TestWorker{}
	t1.getImageCallCount = 0

	t2 := &TestWorker{}
	t2.getImageCallCount = 0

	a.AddWorker(t1)
	a.AddWorker(t2)

	for i := 0; i < 10; i++ {
		a.getImageFromWorker()
	}

	if !(t1.getImageCallCount > 0 && t2.getImageCallCount > 0) {
		t.Error("GetImage call random")
	}
}
