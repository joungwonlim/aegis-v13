package feed

import (
	"container/heap"
	"sync"

	"github.com/wonny/aegis/v13/backend/internal/realtime"
)

// SymbolPriorityItem represents a symbol with its priority in the queue
type SymbolPriorityItem struct {
	Priority *realtime.SymbolPriority
	Index    int // Index in the heap
}

// PriorityQueue implements a max-heap for symbol priorities
// ⭐ SSOT: WebSocket 40개 심볼 선택은 이 큐에서만
type PriorityQueue struct {
	mu    sync.RWMutex
	items []*SymbolPriorityItem
	index map[string]int // code -> index mapping for O(1) lookup
}

// NewPriorityQueue creates a new priority queue
func NewPriorityQueue() *PriorityQueue {
	pq := &PriorityQueue{
		items: make([]*SymbolPriorityItem, 0),
		index: make(map[string]int),
	}
	heap.Init(pq)
	return pq
}

// Len returns the number of items in the queue
func (pq *PriorityQueue) Len() int {
	return len(pq.items)
}

// Less compares two items (max-heap: higher score first)
func (pq *PriorityQueue) Less(i, j int) bool {
	return pq.items[i].Priority.Score > pq.items[j].Priority.Score
}

// Swap swaps two items
func (pq *PriorityQueue) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
	pq.items[i].Index = i
	pq.items[j].Index = j
	pq.index[pq.items[i].Priority.Code] = i
	pq.index[pq.items[j].Priority.Code] = j
}

// Push adds an item to the queue
func (pq *PriorityQueue) Push(x interface{}) {
	item := x.(*SymbolPriorityItem)
	item.Index = len(pq.items)
	pq.items = append(pq.items, item)
	pq.index[item.Priority.Code] = item.Index
}

// Pop removes and returns the highest priority item
func (pq *PriorityQueue) Pop() interface{} {
	old := pq.items
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.Index = -1
	pq.items = old[0 : n-1]
	delete(pq.index, item.Priority.Code)
	return item
}

// Update updates the priority of a symbol
func (pq *PriorityQueue) Update(priority *realtime.SymbolPriority) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	priority.CalculateScore()

	if idx, exists := pq.index[priority.Code]; exists {
		// Update existing item
		pq.items[idx].Priority = priority
		heap.Fix(pq, idx)
	} else {
		// Add new item
		item := &SymbolPriorityItem{
			Priority: priority,
		}
		heap.Push(pq, item)
	}
}

// Remove removes a symbol from the queue
func (pq *PriorityQueue) Remove(code string) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if idx, exists := pq.index[code]; exists {
		heap.Remove(pq, idx)
	}
}

// GetTop returns the top N symbols by priority (without removing them)
func (pq *PriorityQueue) GetTop(n int) []*realtime.SymbolPriority {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	if n > len(pq.items) {
		n = len(pq.items)
	}

	result := make([]*realtime.SymbolPriority, n)
	for i := 0; i < n; i++ {
		result[i] = pq.items[i].Priority
	}
	return result
}

// Contains checks if a symbol is in the queue
func (pq *PriorityQueue) Contains(code string) bool {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	_, exists := pq.index[code]
	return exists
}

// GetLowestScore returns the score of the lowest priority item in the queue
func (pq *PriorityQueue) GetLowestScore() float64 {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	if len(pq.items) == 0 {
		return 0
	}

	// In a max-heap, the lowest score is at the last leaf
	minScore := pq.items[0].Priority.Score
	for _, item := range pq.items {
		if item.Priority.Score < minScore {
			minScore = item.Priority.Score
		}
	}
	return minScore
}
