
package notify

import "sync"

func watchAdd(nd node, c chan<- EventInfo, e Event) eventDiff {
	diff := nd.Watch.Add(c, e)
	if wp := nd.Child[""].Watch; len(wp) != 0 {
		e = wp.Total()
		diff[0] |= e
		diff[1] |= e
		if diff[0] == diff[1] {
			return none
		}
	}
	return diff
}

func watchAddInactive(nd node, c chan<- EventInfo, e Event) eventDiff {
	wp := nd.Child[""].Watch
	if wp == nil {
		wp = make(watchpoint)
		nd.Child[""] = node{Watch: wp}
	}
	diff := wp.Add(c, e)
	e = nd.Watch.Total()
	diff[0] |= e
	diff[1] |= e
	if diff[0] == diff[1] {
		return none
	}
	return diff
}

func watchCopy(src, dst node) {
	for c, e := range src.Watch {
		if c == nil {
			continue
		}
		watchAddInactive(dst, c, e)
	}
	if wpsrc := src.Child[""].Watch; len(wpsrc) != 0 {
		wpdst := dst.Child[""].Watch
		for c, e := range wpsrc {
			if c == nil {
				continue
			}
			wpdst.Add(c, e)
		}
	}
}

func watchDel(nd node, c chan<- EventInfo, e Event) eventDiff {
	diff := nd.Watch.Del(c, e)
	if wp := nd.Child[""].Watch; len(wp) != 0 {
		diffInactive := wp.Del(c, e)
		e = wp.Total()

		diff[0] |= diffInactive[0] | e
		diff[1] |= diffInactive[1] | e
		if diff[0] == diff[1] {
			return none
		}
	}
	return diff
}

func watchTotal(nd node) Event {
	e := nd.Watch.Total()
	if wp := nd.Child[""].Watch; len(wp) != 0 {
		e |= wp.Total()
	}
	return e
}

func watchIsRecursive(nd node) bool {
	ok := nd.Watch.IsRecursive()

	if wp := nd.Child[""].Watch; len(wp) != 0 {

		ok = true
	}
	return ok
}

type recursiveTree struct {
	rw   sync.RWMutex 
	root root

	w interface {
		watcher
		recursiveWatcher
	}
	c chan EventInfo
}

func newRecursiveTree(w recursiveWatcher, c chan EventInfo) *recursiveTree {
	t := &recursiveTree{
		root: root{nd: newnode("")},
		w: struct {
			watcher
			recursiveWatcher
		}{w.(watcher), w},
		c: c,
	}
	go t.dispatch()
	return t
}

func (t *recursiveTree) dispatch() {
	for ei := range t.c {
		dbgprintf("dispatching %v on %q", ei.Event(), ei.Path())
		go func(ei EventInfo) {
			nd, ok := node{}, false
			dir, base := split(ei.Path())
			fn := func(it node, isbase bool) error {
				if isbase {
					nd = it
				} else {
					it.Watch.Dispatch(ei, recursive)
				}
				return nil
			}
			t.rw.RLock()
			defer t.rw.RUnlock()

			if err := t.root.WalkPath(dir, fn); err != nil {
				dbgprint("dispatch did not reach leaf:", err)
				return
			}

			nd.Watch.Dispatch(ei, 0)

			if nd, ok = nd.Child[base]; ok {
				nd.Watch.Dispatch(ei, 0)
			}
		}(ei)
	}
}

func (t *recursiveTree) Watch(path string, c chan<- EventInfo, events ...Event) error {
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
	eventset := joinevents(events)
	if isrec {
		eventset |= recursive
	}
	t.rw.Lock()
	defer t.rw.Unlock()

	parent := node{}
	self := false
	err = t.root.WalkPath(path, func(nd node, isbase bool) error {
		if watchTotal(nd) != 0 {
			parent = nd
			self = isbase
			return errSkip
		}
		return nil
	})
	cur := t.root.Add(path) 
	if err == nil && parent.Watch != nil {

		var diff eventDiff
		if self {
			diff = watchAdd(cur, c, eventset)
		} else {
			diff = watchAddInactive(parent, c, eventset)
		}
		switch {
		case diff == none:

		case diff[0] == 0:

			panic("dangling watchpoint: " + parent.Name)
		default:
			if isrec || watchIsRecursive(parent) {
				err = t.w.RecursiveRewatch(parent.Name, parent.Name, diff[0], diff[1])
			} else {
				err = t.w.Rewatch(parent.Name, diff[0], diff[1])
			}
			if err != nil {
				watchDel(parent, c, diff.Event())
				return err
			}
			watchAdd(cur, c, eventset)

			return nil
		}
		if !self {
			watchAdd(cur, c, eventset)
		}
		return nil
	}

	var children []node
	fn := func(nd node) error {
		if len(nd.Watch) == 0 {
			return nil
		}
		children = append(children, nd)
		return errSkip
	}
	switch must(cur.Walk(fn)); len(children) {
	case 0:

	case 1:
		watchAdd(cur, c, eventset) 
		watchCopy(children[0], cur)
		err = t.w.RecursiveRewatch(children[0].Name, cur.Name, watchTotal(children[0]),
			watchTotal(cur))
		if err != nil {

			cur.Child[""] = node{}
			delete(cur.Watch, c)
			return err
		}
		return nil
	default:
		watchAdd(cur, c, eventset)

		for _, nd := range children {
			watchCopy(nd, cur)
		}

		if err = t.w.RecursiveWatch(cur.Name, watchTotal(cur)); err != nil {

			cur.Child[""] = node{}
			delete(cur.Watch, c)
			return err
		}

		var e error
		for _, nd := range children {
			if watchIsRecursive(nd) {
				e = t.w.RecursiveUnwatch(nd.Name)
			} else {
				e = t.w.Unwatch(nd.Name)
			}
			if e != nil {
				err = nonil(err, e)

			}
		}
		return err
	}

	switch diff := watchAdd(cur, c, eventset); {
	case diff == none:

		panic("watch requested but no parent watchpoint found: " + cur.Name)
	case diff[0] == 0:
		if isrec {
			err = t.w.RecursiveWatch(cur.Name, diff[1])
		} else {
			err = t.w.Watch(cur.Name, diff[1])
		}
		if err != nil {
			watchDel(cur, c, diff.Event())
			return err
		}
	default:

		panic("watch requested but no parent watchpoint found: " + cur.Name)
	}
	return nil
}

func (t *recursiveTree) Stop(c chan<- EventInfo) {
	var err error
	fn := func(nd node) (e error) {
		diff := watchDel(nd, c, all)
		switch {
		case diff == none && watchTotal(nd) == 0:

			return nil
		case diff == none:

		case diff[1] == 0:
			if watchIsRecursive(nd) {
				e = t.w.RecursiveUnwatch(nd.Name)
			} else {
				e = t.w.Unwatch(nd.Name)
			}
		default:
			if watchIsRecursive(nd) {
				e = t.w.RecursiveRewatch(nd.Name, nd.Name, diff[0], diff[1])
			} else {
				e = t.w.Rewatch(nd.Name, diff[0], diff[1])
			}
		}
		fn := func(nd node) error {
			watchDel(nd, c, all)
			return nil
		}
		err = nonil(err, e, nd.Walk(fn))

		return errSkip
	}
	t.rw.Lock()
	e := t.root.Walk("", fn) 
	t.rw.Unlock()
	if e != nil {
		err = nonil(err, e)
	}
	dbgprintf("Stop(%p) error: %v\n", c, err)
}

func (t *recursiveTree) Close() error {
	err := t.w.Close()
	close(t.c)
	return err
}
