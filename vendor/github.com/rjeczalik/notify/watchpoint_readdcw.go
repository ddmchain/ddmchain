
// +build windows

package notify

func eventmask(ei EventInfo, extra Event) (e Event) {
	if e = ei.Event() | extra; e&fileActionAll != 0 {
		if ev, ok := ei.(*event); ok {
			switch ev.ftype {
			case fTypeFile:
				e |= FileNotifyChangeFileName
			case fTypeDirectory:
				e |= FileNotifyChangeDirName
			case fTypeUnknown:
				e |= fileNotifyChangeModified
			}
			return e &^ fileActionAll
		}
	}
	return
}

func matches(set, event Event) bool {
	return (set&omit)^(event&omit) == 0 && (set&event == event || set&fileNotifyChangeModified&event != 0)
}
