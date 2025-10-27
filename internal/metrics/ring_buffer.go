package metrics

import (
	"sync"
	"time"
)

// DataPoint представляет точку данных с timestamp
type DataPoint struct {
	Timestamp time.Time
	Value     float64
	Labels    map[string]string // Дополнительные метки (endpoint, method, etc.)
}

// RingBuffer - потокобезопасный кольцевой буфер для хранения метрик
type RingBuffer struct {
	mu       sync.RWMutex
	data     []DataPoint
	capacity int
	head     int  // Индекс для записи
	size     int  // Текущий размер
	isFull   bool // Флаг заполненности
}

// NewRingBuffer создает новый ring buffer с заданной емкостью
func NewRingBuffer(capacity int) *RingBuffer {
	return &RingBuffer{
		data:     make([]DataPoint, capacity),
		capacity: capacity,
		head:     0,
		size:     0,
		isFull:   false,
	}
}

// Add добавляет новую точку данных в buffer
func (rb *RingBuffer) Add(value float64, labels map[string]string) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.data[rb.head] = DataPoint{
		Timestamp: time.Now(),
		Value:     value,
		Labels:    labels,
	}

	rb.head = (rb.head + 1) % rb.capacity

	if rb.isFull {
		// Buffer уже заполнен, перезаписываем старые данные
	} else {
		rb.size++
		if rb.size == rb.capacity {
			rb.isFull = true
		}
	}
}

// GetAll возвращает все данные в хронологическом порядке (старые -> новые)
func (rb *RingBuffer) GetAll() []DataPoint {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.size == 0 {
		return []DataPoint{}
	}

	result := make([]DataPoint, rb.size)

	if rb.isFull {
		// Если buffer заполнен, начинаем с head (самая старая запись)
		for i := 0; i < rb.capacity; i++ {
			idx := (rb.head + i) % rb.capacity
			result[i] = rb.data[idx]
		}
	} else {
		// Если не заполнен, просто копируем с начала
		copy(result, rb.data[:rb.size])
	}

	return result
}

// GetRange возвращает данные за указанный период времени
func (rb *RingBuffer) GetRange(from, to time.Time) []DataPoint {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.size == 0 {
		return []DataPoint{}
	}

	var result []DataPoint

	if rb.isFull {
		for i := 0; i < rb.capacity; i++ {
			idx := (rb.head + i) % rb.capacity
			dp := rb.data[idx]
			if (dp.Timestamp.Equal(from) || dp.Timestamp.After(from)) &&
				(dp.Timestamp.Equal(to) || dp.Timestamp.Before(to)) {
				result = append(result, dp)
			}
		}
	} else {
		for i := 0; i < rb.size; i++ {
			dp := rb.data[i]
			if (dp.Timestamp.Equal(from) || dp.Timestamp.After(from)) &&
				(dp.Timestamp.Equal(to) || dp.Timestamp.Before(to)) {
				result = append(result, dp)
			}
		}
	}

	return result
}

// GetLast возвращает последние N записей
func (rb *RingBuffer) GetLast(n int) []DataPoint {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.size == 0 || n <= 0 {
		return []DataPoint{}
	}

	if n > rb.size {
		n = rb.size
	}

	result := make([]DataPoint, n)

	if rb.isFull {
		// Последние n записей находятся перед head
		start := (rb.head - n + rb.capacity) % rb.capacity
		for i := 0; i < n; i++ {
			idx := (start + i) % rb.capacity
			result[i] = rb.data[idx]
		}
	} else {
		// Просто берем последние n элементов
		start := rb.size - n
		copy(result, rb.data[start:rb.size])
	}

	return result
}

// Size возвращает текущий размер buffer
func (rb *RingBuffer) Size() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.size
}

// Capacity возвращает максимальную емкость buffer
func (rb *RingBuffer) Capacity() int {
	return rb.capacity
}

// Clear очищает buffer
func (rb *RingBuffer) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.head = 0
	rb.size = 0
	rb.isFull = false
	// Не нужно очищать data slice, просто сбрасываем индексы
}

// IsFull возвращает true если buffer заполнен
func (rb *RingBuffer) IsFull() bool {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.isFull
}

