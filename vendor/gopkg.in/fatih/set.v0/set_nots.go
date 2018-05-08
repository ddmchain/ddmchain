package set

import (
	"fmt"
	"strings"
)

type set struct {
	m map[interface{}]struct{} 
}

type SetNonTS struct {
	set
}

func NewNonTS(items ...interface{}) *SetNonTS {
	s := &SetNonTS{}
	s.m = make(map[interface{}]struct{})

	var _ Interface = s

	s.Add(items...)
	return s
}

func (s *set) New(items ...interface{}) Interface {
	return NewNonTS(items...)
}

func (s *set) Add(items ...interface{}) {
	if len(items) == 0 {
		return
	}

	for _, item := range items {
		s.m[item] = keyExists
	}
}

func (s *set) Remove(items ...interface{}) {
	if len(items) == 0 {
		return
	}

	for _, item := range items {
		delete(s.m, item)
	}
}

func (s *set) Pop() interface{} {
	for item := range s.m {
		delete(s.m, item)
		return item
	}
	return nil
}

func (s *set) Has(items ...interface{}) bool {

	if len(items) == 0 {
		return false
	}

	has := true
	for _, item := range items {
		if _, has = s.m[item]; !has {
			break
		}
	}
	return has
}

func (s *set) Size() int {
	return len(s.m)
}

func (s *set) Clear() {
	s.m = make(map[interface{}]struct{})
}

func (s *set) IsEmpty() bool {
	return s.Size() == 0
}

func (s *set) IsEqual(t Interface) bool {

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

func (s *set) IsSubset(t Interface) (subset bool) {
	subset = true

	t.Each(func(item interface{}) bool {
		_, subset = s.m[item]
		return subset
	})

	return
}

func (s *set) IsSuperset(t Interface) bool {
	return t.IsSubset(s)
}

func (s *set) Each(f func(item interface{}) bool) {
	for item := range s.m {
		if !f(item) {
			break
		}
	}
}

func (s *set) String() string {
	t := make([]string, 0, len(s.List()))
	for _, item := range s.List() {
		t = append(t, fmt.Sprintf("%v", item))
	}

	return fmt.Sprintf("[%s]", strings.Join(t, ", "))
}

func (s *set) List() []interface{} {
	list := make([]interface{}, 0, len(s.m))

	for item := range s.m {
		list = append(list, item)
	}

	return list
}

func (s *set) Copy() Interface {
	return NewNonTS(s.List()...)
}

func (s *set) Merge(t Interface) {
	t.Each(func(item interface{}) bool {
		s.m[item] = keyExists
		return true
	})
}

func (s *set) Separate(t Interface) {
	s.Remove(t.List()...)
}
