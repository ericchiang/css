package css

// queue is a ring buffer implementation of a fixed size queue.
//
// This is an internal implementation aimed at queueing peeks into the token
// stream. Method intentionally panic when misused.
type queue struct {
	vals  []token
	start int
	n     int
}

// newQueue creates a queue with a max of size elements. Any attempt to push
// onto a full queue will panic.
func newQueue(size int) *queue {
	return &queue{vals: make([]token, size)}
}

// get returns the token n elements into the queue. get panics if there aren't
// enough elements to satisfy the request.
func (q *queue) get(n int) token {
	if n >= q.n {
		panic("queue: out of index lookup")
	}
	return q.vals[q.index(n)]
}

// index is an internal method that returns the index of the particular offset,
// performing the ring buffer logic.
func (q *queue) index(n int) int {
	// Visual example of logic where 'x' is p.start and 'y' is the target index:
	//
	//   [_, x, _, _] start = 1, len = 4
	//   [_, x, y, _] n = 1, y = (1 + 1) % 4 = 2
	//   [_, x, _, y] n = 2, y = (1 + 2) % 4 = 3
	//   [y, x, _, _] n = 3, y = (1 + 3) % 4 = 0
	//
	return (q.start + n) % len(q.vals)
}

// push enqueues an element. It panics if the queue is full.
func (q *queue) push(t token) {
	if q.n == len(q.vals) {
		panic("queue: too many elements added to queue")
	}
	q.vals[q.index(q.n)] = t
	q.n++
}

// pop dequeues an element. It panics if the queue is empty.
func (q *queue) pop() token {
	if q.n == 0 {
		panic("queue: pop from an empty queue")
	}
	t := q.vals[q.start]
	q.start = q.index(1)
	q.n--
	return t
}

func (q *queue) len() int {
	return q.n
}
