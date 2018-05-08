
package packet

import (
	"io"
	"io/ioutil"
	"strings"
)

type UserId struct {
	Id string 

	Name, Comment, Email string
}

func hasInvalidCharacters(s string) bool {
	for _, c := range s {
		switch c {
		case '(', ')', '<', '>', 0:
			return true
		}
	}
	return false
}

func NewUserId(name, comment, email string) *UserId {

	if hasInvalidCharacters(name) || hasInvalidCharacters(comment) || hasInvalidCharacters(email) {
		return nil
	}

	uid := new(UserId)
	uid.Name, uid.Comment, uid.Email = name, comment, email
	uid.Id = name
	if len(comment) > 0 {
		if len(uid.Id) > 0 {
			uid.Id += " "
		}
		uid.Id += "("
		uid.Id += comment
		uid.Id += ")"
	}
	if len(email) > 0 {
		if len(uid.Id) > 0 {
			uid.Id += " "
		}
		uid.Id += "<"
		uid.Id += email
		uid.Id += ">"
	}
	return uid
}

func (uid *UserId) parse(r io.Reader) (err error) {

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return
	}
	uid.Id = string(b)
	uid.Name, uid.Comment, uid.Email = parseUserId(uid.Id)
	return
}

func (uid *UserId) Serialize(w io.Writer) error {
	err := serializeHeader(w, packetTypeUserId, len(uid.Id))
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(uid.Id))
	return err
}

func parseUserId(id string) (name, comment, email string) {
	var n, c, e struct {
		start, end int
	}
	var state int

	for offset, rune := range id {
		switch state {
		case 0:

			n.start = offset
			state = 1
			fallthrough
		case 1:

			if rune == '(' {
				state = 2
				n.end = offset
			} else if rune == '<' {
				state = 5
				n.end = offset
			}
		case 2:

			c.start = offset
			state = 3
			fallthrough
		case 3:

			if rune == ')' {
				state = 4
				c.end = offset
			}
		case 4:

			if rune == '<' {
				state = 5
			}
		case 5:

			e.start = offset
			state = 6
			fallthrough
		case 6:

			if rune == '>' {
				state = 7
				e.end = offset
			}
		default:

		}
	}
	switch state {
	case 1:

		n.end = len(id)
	case 3:

		c.end = len(id)
	case 6:

		e.end = len(id)
	}

	name = strings.TrimSpace(id[n.start:n.end])
	comment = strings.TrimSpace(id[c.start:c.end])
	email = strings.TrimSpace(id[e.start:e.end])
	return
}
