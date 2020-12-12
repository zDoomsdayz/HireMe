package queue

import (
	"errors"
)

// History struct
type History struct {
	Time     string
	Activity string
}

type Node struct {
	item History
	next *Node
}

type Queue struct {
	front *Node
	back  *Node
	size  int
}

func (p *Queue) Enqueue(h History) error {
	newNode := &Node{
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
	return nil
}

func (p *Queue) Dequeue() (History, error) {
	var item History

	if p.front == nil {
		return History{}, errors.New("Empty Queue!")
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

func (p *Queue) PrintAllNodes() []History {
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
