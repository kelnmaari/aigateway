// Package queue provides priority queue for GitLab MR reviews
package queue

import (
	"container/heap"
	"strings"
	"sync"
	"time"
)

// Priority levels
type Priority int

const (
	PriorityLow      Priority = 0
	PriorityNormal   Priority = 1
	PriorityHigh     Priority = 2
	PriorityCritical Priority = 3
)

// Job represents a review job in the queue
type Job struct {
	ID           string
	ReviewID     string
	ProjectID    string
	MRIID        int
	TargetBranch string
	SourceBranch string
	Priority     Priority
	CreatedAt    time.Time
	ScheduledAt  time.Time
	
	// Internal priority heap index
	index int
}

// PriorityQueue implements a priority queue with branch-based prioritization
type PriorityQueue struct {
	mu           sync.RWMutex
	items        jobHeap
	branchRules  []BranchRule
	defaultPrio  Priority
}

// BranchRule defines priority rules for branches
type BranchRule struct {
	Pattern  string   // Branch name pattern (glob-like)
	Priority Priority
	Branches []string // Exact branch names
}

// NewPriorityQueue creates a new priority queue
func NewPriorityQueue(rules []BranchRule) *PriorityQueue {
	pq := &PriorityQueue{
		items:       make(jobHeap, 0),
		branchRules: rules,
		defaultPrio: PriorityNormal,
	}
	
	// Add default rules if none provided
	if len(pq.branchRules) == 0 {
		pq.branchRules = DefaultBranchRules()
	}
	
	heap.Init(&pq.items)
	return pq
}

// DefaultBranchRules returns default priority rules
func DefaultBranchRules() []BranchRule {
	return []BranchRule{
		{
			Priority: PriorityCritical,
			Branches: []string{"main", "master", "production", "prod"},
		},
		{
			Priority: PriorityHigh,
			Pattern:  "release/*",
			Branches: []string{"release", "staging", "develop"},
		},
		{
			Priority: PriorityHigh,
			Pattern:  "hotfix/*",
		},
		{
			Priority: PriorityNormal,
			Pattern:  "feature/*",
		},
		{
			Priority: PriorityLow,
			Pattern:  "wip/*",
			Branches: []string{"experimental"},
		},
	}
}

// Push adds a job to the queue with automatic priority detection
func (pq *PriorityQueue) Push(job *Job) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	
	// Determine priority based on target branch
	if job.Priority == 0 {
		job.Priority = pq.determinePriority(job.TargetBranch)
	}
	
	// Set timestamps
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	if job.ScheduledAt.IsZero() {
		job.ScheduledAt = time.Now()
	}
	
	heap.Push(&pq.items, job)
}

// Pop removes and returns the highest priority job
func (pq *PriorityQueue) Pop() *Job {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	
	if pq.items.Len() == 0 {
		return nil
	}
	
	item := heap.Pop(&pq.items).(*Job)
	return item
}

// Peek returns the highest priority job without removing it
func (pq *PriorityQueue) Peek() *Job {
	pq.mu.RLock()
	defer pq.mu.RUnlock()
	
	if pq.items.Len() == 0 {
		return nil
	}
	
	return pq.items[0]
}

// Len returns the number of jobs in the queue
func (pq *PriorityQueue) Len() int {
	pq.mu.RLock()
	defer pq.mu.RUnlock()
	return pq.items.Len()
}

// Remove removes a job by ID
func (pq *PriorityQueue) Remove(jobID string) bool {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	
	for i, job := range pq.items {
		if job.ID == jobID {
			heap.Remove(&pq.items, i)
			return true
		}
	}
	return false
}

// UpdatePriority updates a job's priority
func (pq *PriorityQueue) UpdatePriority(jobID string, priority Priority) bool {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	
	for i, job := range pq.items {
		if job.ID == jobID {
			job.Priority = priority
			heap.Fix(&pq.items, i)
			return true
		}
	}
	return false
}

// GetStats returns queue statistics
func (pq *PriorityQueue) GetStats() QueueStats {
	pq.mu.RLock()
	defer pq.mu.RUnlock()
	
	stats := QueueStats{
		Total:    pq.items.Len(),
		ByPriority: make(map[Priority]int),
	}
	
	for _, job := range pq.items {
		stats.ByPriority[job.Priority]++
		
		waitTime := time.Since(job.ScheduledAt)
		stats.TotalWaitTime += waitTime
		if waitTime > stats.MaxWaitTime {
			stats.MaxWaitTime = waitTime
		}
	}
	
	if stats.Total > 0 {
		stats.AvgWaitTime = stats.TotalWaitTime / time.Duration(stats.Total)
	}
	
	return stats
}

// QueueStats contains queue statistics
type QueueStats struct {
	Total         int
	ByPriority    map[Priority]int
	TotalWaitTime time.Duration
	AvgWaitTime   time.Duration
	MaxWaitTime   time.Duration
}

// determinePriority determines job priority based on target branch
func (pq *PriorityQueue) determinePriority(branch string) Priority {
	branch = strings.ToLower(branch)
	
	for _, rule := range pq.branchRules {
		// Check exact matches
		for _, b := range rule.Branches {
			if strings.ToLower(b) == branch {
				return rule.Priority
			}
		}
		
		// Check pattern match
		if rule.Pattern != "" && matchPattern(rule.Pattern, branch) {
			return rule.Priority
		}
	}
	
	return pq.defaultPrio
}

// matchPattern performs simple glob-like pattern matching
func matchPattern(pattern, value string) bool {
	pattern = strings.ToLower(pattern)
	value = strings.ToLower(value)
	
	// Handle wildcard at end: "release/*"
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(value, prefix+"/")
	}
	
	// Handle wildcard at start: "*/release"
	if strings.HasPrefix(pattern, "*/") {
		suffix := strings.TrimPrefix(pattern, "*/")
		return strings.HasSuffix(value, "/"+suffix) || value == suffix
	}
	
	// Handle wildcard in middle: "feature/*/main"
	if strings.Contains(pattern, "/*/") {
		parts := strings.Split(pattern, "/*/")
		return strings.HasPrefix(value, parts[0]+"/") && strings.HasSuffix(value, "/"+parts[1])
	}
	
	return pattern == value
}

// jobHeap implements heap.Interface for priority queue
type jobHeap []*Job

func (h jobHeap) Len() int { return len(h) }

func (h jobHeap) Less(i, j int) bool {
	// Higher priority first
	if h[i].Priority != h[j].Priority {
		return h[i].Priority > h[j].Priority
	}
	// Same priority: earlier scheduled first (FIFO within priority)
	return h[i].ScheduledAt.Before(h[j].ScheduledAt)
}

func (h jobHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *jobHeap) Push(x interface{}) {
	n := len(*h)
	job := x.(*Job)
	job.index = n
	*h = append(*h, job)
}

func (h *jobHeap) Pop() interface{} {
	old := *h
	n := len(old)
	job := old[n-1]
	old[n-1] = nil // avoid memory leak
	job.index = -1
	*h = old[0 : n-1]
	return job
}

// PriorityConfig configuration for priority queue
type PriorityConfig struct {
	Rules        []BranchRule `json:"rules" yaml:"rules"`
	DefaultPrio  Priority     `json:"default_priority" yaml:"default_priority"`
}

// ParsePriorityConfig parses priority configuration
func ParsePriorityConfig(config PriorityConfig) *PriorityQueue {
	rules := config.Rules
	if len(rules) == 0 {
		rules = DefaultBranchRules()
	}
	
	pq := NewPriorityQueue(rules)
	if config.DefaultPrio > 0 {
		pq.defaultPrio = config.DefaultPrio
	}
	
	return pq
}

