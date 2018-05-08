package ast

import (
	"fmt"
	"github.com/robertkrimen/otto/file"
)

type CommentPosition int

const (
	_        CommentPosition = iota
	LEADING                  
	TRAILING                 
	KEY                      
	COLON                    
	FINAL                    
	IF                       
	WHILE                    
	DO                       
	FOR                      
	WITH                     
	TBD
)

type Comment struct {
	Begin    file.Idx
	Text     string
	Position CommentPosition
}

func NewComment(text string, idx file.Idx) *Comment {
	comment := &Comment{
		Begin:    idx,
		Text:     text,
		Position: TBD,
	}

	return comment
}

func (cp CommentPosition) String() string {
	switch cp {
	case LEADING:
		return "Leading"
	case TRAILING:
		return "Trailing"
	case KEY:
		return "Key"
	case COLON:
		return "Colon"
	case FINAL:
		return "Final"
	case IF:
		return "If"
	case WHILE:
		return "While"
	case DO:
		return "Do"
	case FOR:
		return "For"
	case WITH:
		return "With"
	default:
		return "???"
	}
}

func (c Comment) String() string {
	return fmt.Sprintf("Comment: %v", c.Text)
}

type Comments struct {

	CommentMap CommentMap

	Comments []*Comment

	future []*Comment

	Current Expression

	wasLineBreak bool

	primary bool

	afterBlock bool
}

func NewComments() *Comments {
	comments := &Comments{
		CommentMap: CommentMap{},
	}

	return comments
}

func (c *Comments) String() string {
	return fmt.Sprintf("NODE: %v, Comments: %v, Future: %v(LINEBREAK:%v)", c.Current, len(c.Comments), len(c.future), c.wasLineBreak)
}

func (c *Comments) FetchAll() []*Comment {
	defer func() {
		c.Comments = nil
		c.future = nil
	}()

	return append(c.Comments, c.future...)
}

func (c *Comments) Fetch() []*Comment {
	defer func() {
		c.Comments = nil
	}()

	return c.Comments
}

func (c *Comments) ResetLineBreak() {
	c.wasLineBreak = false
}

func (c *Comments) MarkPrimary() {
	c.primary = true
	c.wasLineBreak = false
}

func (c *Comments) AfterBlock() {
	c.afterBlock = true
}

func (c *Comments) AddComment(comment *Comment) {
	if c.primary {
		if !c.wasLineBreak {
			c.Comments = append(c.Comments, comment)
		} else {
			c.future = append(c.future, comment)
		}
	} else {
		if !c.wasLineBreak || (c.Current == nil && !c.afterBlock) {
			c.Comments = append(c.Comments, comment)
		} else {
			c.future = append(c.future, comment)
		}
	}
}

func (c *Comments) MarkComments(position CommentPosition) {
	for _, comment := range c.Comments {
		if comment.Position == TBD {
			comment.Position = position
		}
	}
	for _, c := range c.future {
		if c.Position == TBD {
			c.Position = position
		}
	}
}

func (c *Comments) Unset() {
	if c.Current != nil {
		c.applyComments(c.Current, c.Current, TRAILING)
		c.Current = nil
	}
	c.wasLineBreak = false
	c.primary = false
	c.afterBlock = false
}

func (c *Comments) SetExpression(node Expression) {

	if c.Current == node {
		return
	}
	if c.Current != nil && c.Current.Idx1() == node.Idx1() {
		c.Current = node
		return
	}
	previous := c.Current
	c.Current = node

	c.applyComments(node, previous, TRAILING)
}

func (c *Comments) PostProcessNode(node Node) {
	c.applyComments(node, nil, TRAILING)
}

func (c *Comments) applyComments(node, previous Node, position CommentPosition) {
	if previous != nil {
		c.CommentMap.AddComments(previous, c.Comments, position)
		c.Comments = nil
	} else {
		c.CommentMap.AddComments(node, c.Comments, position)
		c.Comments = nil
	}

	if previous != nil {
		c.CommentMap.AddComments(node, c.future, position)
		c.future = nil
	}
}

func (c *Comments) AtLineBreak() {
	c.wasLineBreak = true
}

type CommentMap map[Node][]*Comment

func (cm CommentMap) AddComment(node Node, comment *Comment) {
	list := cm[node]
	list = append(list, comment)

	cm[node] = list
}

func (cm CommentMap) AddComments(node Node, comments []*Comment, position CommentPosition) {
	for _, comment := range comments {
		if comment.Position == TBD {
			comment.Position = position
		}
		cm.AddComment(node, comment)
	}
}

func (cm CommentMap) Size() int {
	size := 0
	for _, comments := range cm {
		size += len(comments)
	}

	return size
}

func (cm CommentMap) MoveComments(from, to Node, position CommentPosition) {
	for i, c := range cm[from] {
		if c.Position == position {
			cm.AddComment(to, c)

			cm[from][i] = cm[from][len(cm[from])-1]
			cm[from][len(cm[from])-1] = nil
			cm[from] = cm[from][:len(cm[from])-1]
		}
	}
}
