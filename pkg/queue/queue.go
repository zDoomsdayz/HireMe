package queue

import (
	"errors"
)

// History struct
type History struct {
	Time     string
	Activity string
}

type node struct {
	item History
	next *node
}

// Queue struct
type Queue struct {
	front *node
	back  *node
	size  int
}

// Enqueue add history to the back
func (p *Queue) Enqueue(h History) {
	newNode := &node{
		item: h,
		next: nil,
	}
	if p.front == nil {
		p.front = newNode

	} else {
		//cap it at 10
		if p.size >= 10 {
			p.Dequeue()
		}

		p.back.next = newNode

	}
	p.back = newNode
	p.size++
}

// Dequeue remove histroy from the front
func (p *Queue) Dequeue() (History, error) {
	var item History

	if p.front == nil {
		return History{}, errors.New("empty queue")
	}

	item = p.front.item
	if p.size == 1 {
		p.front = nil
		p.back = nil
	} else {
		p.front = p.front.next
	}
	p.size--
	return item, nil
}

// AllHistory return all of the history in slice
func (p *Queue) AllHistory() []History {
	currentNode := p.front

	data := []History{}

	if currentNode == nil {
		data = append(data, History{})
		return data
	}

	for currentNode != nil {
		data = append(data, currentNode.item)
		currentNode = currentNode.next
	}

	return data
}
