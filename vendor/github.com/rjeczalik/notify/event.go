
package notify

import (
	"fmt"
	"strings"
)

type Event uint32

const (
	Create = osSpecificCreate
	Remove = osSpecificRemove
	Write  = osSpecificWrite
	Rename = osSpecificRename

	All = Create | Remove | Write | Rename
)

const internal = recursive | omit

func (e Event) String() string {
	var s []string
	for _, strmap := range []map[Event]string{estr, osestr} {
		for ev, str := range strmap {
			if e&ev == ev {
				s = append(s, str)
			}
		}
	}
	return strings.Join(s, "|")
}

type EventInfo interface {
	Event() Event     
	Path() string     
	Sys() interface{} 
}

type isDirer interface {
	isDir() (bool, error)
}

var _ fmt.Stringer = (*event)(nil)
var _ isDirer = (*event)(nil)

func (e *event) String() string {
	return e.Event().String() + `: "` + e.Path() + `"`
}

var estr = map[Event]string{
	Create: "notify.Create",
	Remove: "notify.Remove",
	Write:  "notify.Write",
	Rename: "notify.Rename",

	recursive: "recursive",
	omit:      "omit",
}
