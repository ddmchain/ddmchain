package set

import (
	"sync"
)

type Set struct {
	set
	l sync.RWMutex 
}

func New(items ...interface{}) *Set {
	s := &Set{}
	s.m = make(map[interface{}]struct{})

	var _ Interface = s

	s.Add(items...)
	return s
}

func (s *Set) New(items ...interface{}) Interface {
	return New(items...)
}

func (s *Set) Add(items ...interface{}) {
	if len(items) == 0 {
		return
	}

	s.l.Lock()
	defer s.l.Unlock()

	for _, item := range items {
		s.m[item] = keyExists
	}
}

func (s *Set) Remove(items ...interface{}) {
	if len(items) == 0 {
		return
	}

	s.l.Lock()
	defer s.l.Unlock()

	for _, item := range items {
		delete(s.m, item)
	}
}

func (s *Set) Pop() interface{} {
	s.l.RLock()
	for item := range s.m {
		s.l.RUnlock()
		s.l.Lock()
		delete(s.m, item)
		s.l.Unlock()
		return item
	}
	s.l.RUnlock()
	return nil
}

func (s *Set) Has(items ...interface{}) bool {

	if len(items) == 0 {
		return false
	}

	s.l.RLock()
	defer s.l.RUnlock()

	has := true
	for _, item := range items {
		if _, has = s.m[item]; !has {
			break
		}
	}
	return has
}

func (s *Set) Size() int {
	s.l.RLock()
	defer s.l.RUnlock()

	l := len(s.m)
	return l
}

func (s *Set) Clear() {
	s.l.Lock()
	defer s.l.Unlock()

	s.m = make(map[interface{}]struct{})
}

func (s *Set) IsEqual(t Interface) bool {
	s.l.RLock()
	defer s.l.RUnlock()

	if conv, ok := t.(*Set); ok {
		conv.l.RLock()
		defer conv.l.RUnlock()
	}

	if sameSize := len(s.m) == t.Size(); !sameSize {
		return false
	}

	equal := true
	t.Each(func(item interface{}) bool {
		_, equal = s.m[item]
		return equal 
	})

	return equal
}

func (s *Set) IsSubset(t Interface) (subset bool) {
	s.l.RLock()
	defer s.l.RUnlock()

	subset = true

	t.Each(func(item interface{}) bool {
		_, subset = s.m[item]
		return subset
	})

	return
}

func (s *Set) Each(f func(item interface{}) bool) {
	s.l.RLock()
	defer s.l.RUnlock()

	for item := range s.m {
		if !f(item) {
			break
		}
	}
}

func (s *Set) List() []interface{} {
	s.l.RLock()
	defer s.l.RUnlock()

	list := make([]interface{}, 0, len(s.m))

	for item := range s.m {
		list = append(list, item)
	}

	return list
}

func (s *Set) Copy() Interface {
	return New(s.List()...)
}

func (s *Set) Merge(t Interface) {
	s.l.Lock()
	defer s.l.Unlock()

	t.Each(func(item interface{}) bool {
		s.m[item] = keyExists
		return true
	})
}
