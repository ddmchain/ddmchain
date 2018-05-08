
package notify

type eventDiff [2]Event

func (diff eventDiff) Event() Event {
	return diff[1] &^ diff[0]
}

type watchpoint map[chan<- EventInfo]Event

var none eventDiff

var rec = func() (ch chan<- EventInfo) {
	ch = make(chan<- EventInfo)
	close(ch)
	return
}()

func (wp watchpoint) dryAdd(ch chan<- EventInfo, e Event) eventDiff {
	if e &^= internal; wp[ch]&e == e {
		return none
	}
	total := wp[ch] &^ internal
	return eventDiff{total, total | e}
}

func (wp watchpoint) Add(c chan<- EventInfo, e Event) (diff eventDiff) {
	wp[c] |= e
	diff[0] = wp[nil]
	diff[1] = diff[0] | e
	wp[nil] = diff[1] &^ omit

	diff[0] &^= internal
	diff[1] &^= internal
	if diff[0] == diff[1] {
		return none
	}
	return
}

func (wp watchpoint) Del(c chan<- EventInfo, e Event) (diff eventDiff) {
	wp[c] &^= e
	if wp[c] == 0 {
		delete(wp, c)
	}
	diff[0] = wp[nil]
	delete(wp, nil)
	if len(wp) != 0 {

		for _, e := range wp {
			diff[1] |= e
		}
		wp[nil] = diff[1] &^ omit
	}

	diff[0] &^= internal
	diff[1] &^= internal
	if diff[0] == diff[1] {
		return none
	}
	return
}

func (wp watchpoint) Dispatch(ei EventInfo, extra Event) {
	e := eventmask(ei, extra)
	if !matches(wp[nil], e) {
		return
	}
	for ch, eset := range wp {
		if ch != nil && matches(eset, e) {
			select {
			case ch <- ei:
			default: 
				dbgprintf("dropped %s on %q: receiver too slow", ei.Event(), ei.Path())
			}
		}
	}
}

func (wp watchpoint) Total() Event {
	return wp[nil] &^ internal
}

func (wp watchpoint) IsRecursive() bool {
	return wp[nil]&recursive != 0
}
