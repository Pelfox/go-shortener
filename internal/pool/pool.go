package pool

import "sync"

// NewFuncType описывает тип функции, которая вызывается в случае отсутствия
// объектов для переиспользования в контейнере (пуле).
type NewFuncType[T Resetter] func() T

// Pool это контейнер для объектов, связанных общим типом Resetter, которые
// могут быть переиспользованы.
type Pool[T Resetter] struct {
	mu      sync.Mutex
	items   []T
	newFunc NewFuncType[T]
}

// New создаёт и возвращает новый объект Pool с заданным типом объектов
// через Generic и функцией для генерации нового объекта.
func New[T Resetter](newFunc NewFuncType[T]) *Pool[T] {
	return &Pool[T]{
		items:   make([]T, 0),
		newFunc: newFunc,
	}
}

// Get возвращает свободный объект из контейнера (пула). В случае отсутствия
// свободных объектов, создаёт новый, используя newFunc.
func (p *Pool[T]) Get() T {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.items) == 0 {
		return p.newFunc()
	}

	last := len(p.items) - 1
	lastItem := p.items[last]
	p.items = p.items[:last]

	return lastItem
}

// Put сбрасывает переданный объект и помещает его обратно в контейнер (пул).
func (p *Pool[T]) Put(item T) {
	item.Reset()

	p.mu.Lock()
	p.items = append(p.items, item)
	p.mu.Unlock()
}
