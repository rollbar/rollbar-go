package rollbar

import "sync"

// NewQueue returns a new queue with the given initial size.
func NewQueue(size int) *Queue {
	return &Queue{
		nodes: make([]interface{}, size),
		size:  size,
	}
}

// Queue is a basic FIFO queue based on a circular list that resizes as needed.
type Queue struct {
	nodes []interface{}
	size  int
	head  int
	tail  int
	count int

	lock sync.RWMutex
}

// Push adds a node to the queue.
func (q *Queue) Push(n interface{}) {
	q.lock.Lock()
	defer q.lock.Unlock()
	if q.head == q.tail && q.count > 0 {
		nodes := make([]interface{}, len(q.nodes)+q.size)
		copy(nodes, q.nodes[q.head:])
		copy(nodes[len(q.nodes)-q.head:], q.nodes[:q.head])
		q.head = 0
		q.tail = len(q.nodes)
		q.nodes = nodes
	}
	q.nodes[q.tail] = n
	q.tail = (q.tail + 1) % len(q.nodes)
	q.count++
}

// Pop removes and returns a node from the queue in first to last order.
func (q *Queue) Pop() interface{} {
	q.lock.Lock()
	defer q.lock.Unlock()
	if q.count == 0 {
		return nil
	}
	node := q.nodes[q.head]
	q.head = (q.head + 1) % len(q.nodes)
	q.count--
	return node
}

// Items returns all populated (non nil) items
func (q *Queue) Items() []interface{} {
	q.lock.RLock()
	defer q.lock.RUnlock()
	return q.nodes[:q.count]
}
