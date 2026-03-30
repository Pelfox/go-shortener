package pool

import (
	"testing"
)

// testType это вспомогательный (dummy) тип для тестов.
type testType struct {
	value int
}

func (t *testType) Reset() {
	t.value = 0
}

func TestPool_GetCreatesNew(t *testing.T) {
	pool := New(func() *testType {
		return &testType{value: 10}
	})

	object := pool.Get()
	if object == nil {
		t.Fatal("expected non-nil object")
	}

	if object.value != 10 {
		t.Fatalf("expected value 10, got %d", object.value)
	}
}

func TestPool_PutResetsObject(t *testing.T) {
	pool := New(func() *testType {
		return &testType{}
	})

	object := &testType{value: 100}
	pool.Put(object)

	got := pool.Get()
	if got.value != 0 {
		t.Fatalf("expected value reset to 0, got %d", got.value)
	}
}

func TestPool_ReusesObject(t *testing.T) {
	pool := New(func() *testType {
		return &testType{}
	})

	object := &testType{value: 55}
	pool.Put(object)

	got := pool.Get()
	if got != object {
		t.Fatal("expected same object to be reused")
	}
}

func TestPool_MultipleObjects(t *testing.T) {
	pool := New(func() *testType {
		return &testType{}
	})

	// кладём несколько объектов
	for i := range 5 {
		pool.Put(&testType{value: i + 1})
	}

	// достаём и проверяем reset
	for range 5 {
		object := pool.Get()
		if object.value != 0 {
			t.Fatalf("expected reset value 0, got %d", object.value)
		}
	}
}
