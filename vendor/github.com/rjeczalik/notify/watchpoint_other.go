
// +build !windows

package notify

func eventmask(ei EventInfo, extra Event) Event {
	return ei.Event() | extra
}

func matches(set, event Event) bool {
	return (set&omit)^(event&omit) == 0 && set&event == event
}
