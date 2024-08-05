package queue

import (
	"fmt"
	"sync"
)

type PortQueue struct {
	mu    sync.Mutex
	ports []int
}

func NewPortQueue(start, end int) *PortQueue {
	ports := make([]int, end-start+1)
	for i := start; i <= end; i++ {
		ports[i-start] = i
	}
	return &PortQueue{ports: ports}
}

func (q *PortQueue) Pop() (int, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.ports) == 0 {
		return 0, fmt.Errorf("queue is empty")
	}

	port := q.ports[0]
	q.ports = q.ports[1:]
	return port, nil
}

func (q *PortQueue) PutBack(port int) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.ports = append(q.ports, port)
}

func (q *PortQueue) Length() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	return len(q.ports)
}
