
package notify

import "sync"

type nonrecursiveTree struct {
	rw   sync.RWMutex 
	root root
	w    watcher
	c    chan EventInfo
	rec  chan EventInfo
}

func newNonrecursiveTree(w watcher, c, rec chan EventInfo) *nonrecursiveTree {
	if rec == nil {
		rec = make(chan EventInfo, buffer)
	}
	t := &nonrecursiveTree{
		root: root{nd: newnode("")},
		w:    w,
		c:    c,
		rec:  rec,
	}
	go t.dispatch(c)
	go t.internal(rec)
	return t
}

func (t *nonrecursiveTree) dispatch(c <-chan EventInfo) {
	for ei := range c {
		dbgprintf("dispatching %v on %q", ei.Event(), ei.Path())
		go func(ei EventInfo) {
			var nd node
			var isrec bool
			dir, base := split(ei.Path())
			fn := func(it node, isbase bool) error {
				isrec = isrec || it.Watch.IsRecursive()
				if isbase {
					nd = it
				} else {
					it.Watch.Dispatch(ei, recursive)
				}
				return nil
			}
			t.rw.RLock()

			if err := t.root.WalkPath(dir, fn); err != nil {
				dbgprint("dispatch did not reach leaf:", err)
				t.rw.RUnlock()
				return
			}

			nd.Watch.Dispatch(ei, 0)
			isrec = isrec || nd.Watch.IsRecursive()

			if nd, ok := nd.Child[base]; ok {
				isrec = isrec || nd.Watch.IsRecursive()
				nd.Watch.Dispatch(ei, 0)
			}
			t.rw.RUnlock()

			if !isrec || ei.Event() != Create {
				return
			}
			if ok, err := ei.(isDirer).isDir(); !ok || err != nil {
				return
			}
			t.rec <- ei
		}(ei)
	}
}

func (t *nonrecursiveTree) internal(rec <-chan EventInfo) {
	for ei := range rec {
		var nd node
		var eset = internal
		t.rw.Lock()
		t.root.WalkPath(ei.Path(), func(it node, _ bool) error {
			if e := it.Watch[t.rec]; e != 0 && e > eset {
				eset = e
			}
			nd = it
			return nil
		})
		if eset == internal {
			t.rw.Unlock()
			continue
		}
		err := nd.Add(ei.Path()).AddDir(t.recFunc(eset))
		t.rw.Unlock()
		if err != nil {
			dbgprintf("internal(%p) error: %v", rec, err)
		}
	}
}

func (t *nonrecursiveTree) watchAdd(nd node, c chan<- EventInfo, e Event) eventDiff {
	if e&recursive != 0 {
		diff := nd.Watch.Add(t.rec, e|Create|omit)
		nd.Watch.Add(c, e)
		return diff
	}
	return nd.Watch.Add(c, e)
}

func (t *nonrecursiveTree) watchDelMin(min Event, nd node, c chan<- EventInfo, e Event) eventDiff {
	old, ok := nd.Watch[t.rec]
	if ok {
		nd.Watch[t.rec] = min
	}
	diff := nd.Watch.Del(c, e)
	if ok {
		switch old &^= diff[0] &^ diff[1]; {
		case old|internal == internal:
			delete(nd.Watch, t.rec)
			if set, ok := nd.Watch[nil]; ok && len(nd.Watch) == 1 && set == 0 {
				delete(nd.Watch, nil)
			}
		default:
			nd.Watch.Add(t.rec, old|Create)
			switch {
			case diff == none:
			case diff[1]|Create == diff[0]:
				diff = none
			default:
				diff[1] |= Create
			}
		}
	}
	return diff
}

func (t *nonrecursiveTree) watchDel(nd node, c chan<- EventInfo, e Event) eventDiff {
	return t.watchDelMin(0, nd, c, e)
}

func (t *nonrecursiveTree) Watch(path string, c chan<- EventInfo, events ...Event) error {
	if c == nil {
		panic("notify: Watch using nil channel")
	}

	if len(events) == 0 {
		return nil
	}
	path, isrec, err := cleanpath(path)
	if err != nil {
		return err
	}
	eset := joinevents(events)
	t.rw.Lock()
	defer t.rw.Unlock()
	nd := t.root.Add(path)
	if isrec {
		return t.watchrec(nd, c, eset|recursive)
	}
	return t.watch(nd, c, eset)
}

func (t *nonrecursiveTree) watch(nd node, c chan<- EventInfo, e Event) (err error) {
	diff := nd.Watch.Add(c, e)
	switch {
	case diff == none:
		return nil
	case diff[1] == 0:

		panic("eset is empty: " + nd.Name)
	case diff[0] == 0:
		err = t.w.Watch(nd.Name, diff[1])
	default:
		err = t.w.Rewatch(nd.Name, diff[0], diff[1])
	}
	if err != nil {
		nd.Watch.Del(c, diff.Event())
		return err
	}
	return nil
}

func (t *nonrecursiveTree) recFunc(e Event) walkFunc {
	return func(nd node) error {
		switch diff := nd.Watch.Add(t.rec, e|omit|Create); {
		case diff == none:
		case diff[1] == 0:

			panic("eset is empty: " + nd.Name)
		case diff[0] == 0:
			t.w.Watch(nd.Name, diff[1])
		default:
			t.w.Rewatch(nd.Name, diff[0], diff[1])
		}
		return nil
	}
}

func (t *nonrecursiveTree) watchrec(nd node, c chan<- EventInfo, e Event) error {
	var traverse func(walkFunc) error

	switch diff := nd.Watch.dryAdd(t.rec, e|Create); {
	case diff == none:
		t.watchAdd(nd, c, e)
		nd.Watch.Add(t.rec, e|omit|Create)
		return nil
	case diff[1] == 0:

		panic("eset is empty: " + nd.Name)
	case diff[0] == 0:

		traverse = nd.AddDir
	default:
		traverse = nd.Walk
	}

	if err := traverse(t.recFunc(e)); err != nil {
		return err
	}
	t.watchAdd(nd, c, e)
	return nil
}

type walkWatchpointFunc func(Event, node) error

func (t *nonrecursiveTree) walkWatchpoint(nd node, fn walkWatchpointFunc) error {
	type minode struct {
		min Event
		nd  node
	}
	mnd := minode{nd: nd}
	stack := []minode{mnd}
Traverse:
	for n := len(stack); n != 0; n = len(stack) {
		mnd, stack = stack[n-1], stack[:n-1]

		if len(mnd.nd.Watch) != 0 {
			switch err := fn(mnd.min, mnd.nd); err {
			case nil:
			case errSkip:
				continue Traverse
			default:
				return err
			}
		}
		for _, nd := range mnd.nd.Child {
			stack = append(stack, minode{mnd.nd.Watch[t.rec], nd})
		}
	}
	return nil
}

func (t *nonrecursiveTree) Stop(c chan<- EventInfo) {
	fn := func(min Event, nd node) error {

		switch diff := t.watchDelMin(min, nd, c, all); {
		case diff == none:
			return nil
		case diff[1] == 0:
			t.w.Unwatch(nd.Name)
		default:
			t.w.Rewatch(nd.Name, diff[0], diff[1])
		}
		return nil
	}
	t.rw.Lock()
	err := t.walkWatchpoint(t.root.nd, fn) 
	t.rw.Unlock()
	dbgprintf("Stop(%p) error: %v\n", c, err)
}

func (t *nonrecursiveTree) Close() error {
	err := t.w.Close()
	close(t.c)
	return err
}
