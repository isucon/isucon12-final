package data

import (
	"sync"
)

type Model interface {
	GetID() int64
}

type Set[T Model] struct {
	mu   sync.RWMutex
	list []T
	dict map[int]T
}

func (s *Set[T]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.list)
}

func (s *Set[T]) At(index int) T {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.list == nil {
		return *new(T)
	}

	return s.list[index]
}

func (s *Set[T]) Get(id int) (T, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.dict == nil {
		return *new(T), false
	}

	model, ok := s.dict[id]
	return model, ok
}

func (s *Set[T]) Pop() (T, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.list) == 0 {
		return *new(T), false
	}
	i := 0
	popT := s.list[i]

	s.list[i] = s.list[len(s.list)-1]
	if len(s.list) == 1 {
		s.list = []T{}
	} else {
		s.list = s.list[:len(s.list)-1]
	}

	return popT, true
}

func (s *Set[T]) Add(model T) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := model.GetID()
	if id == 0 {
		return false
	}

	if len(s.list) == 0 {
		s.list = []T{model}
	} else {
		pos := 0

		for i := 0; i < len(s.list)-1; i++ {
			m := s.list[i]
			pos = i
			if m.GetID() > model.GetID() {
				break
			}
		}

		s.list = append(s.list[:pos+1], s.list[pos:]...)
		s.list[pos] = model
	}

	if s.dict == nil {
		s.dict = make(map[int]T, 0)
	}
	s.dict[int(id)] = model

	return true
}

type empty struct{}

type LightSet struct {
	mu sync.RWMutex

	inner map[int64]empty
}

func NewLightSet() *LightSet {
	m := make(map[int64]empty, 10000)
	return &LightSet{inner: m}
}

func (l *LightSet) Exists(ID int64) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if len(l.inner) == 0 {
		return false
	}

	if l.inner == nil {
		return false
	}

	_, ok := l.inner[ID]
	return ok
}

func (l *LightSet) Add(ID int64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.inner[ID] = empty{}
}

func (l *LightSet) Remove(ID int64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.inner, ID)
}
