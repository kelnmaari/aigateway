package metrics

import (
	"testing"
	"time"
)

func TestRingBuffer_Add(t *testing.T) {
	rb := NewRingBuffer(3)

	// Добавляем данные
	rb.Add(1.0, nil)
	rb.Add(2.0, nil)
	rb.Add(3.0, nil)

	if rb.Size() != 3 {
		t.Errorf("Expected size 3, got %d", rb.Size())
	}

	if !rb.IsFull() {
		t.Error("Expected buffer to be full")
	}

	// Добавляем еще одну запись (должна перезаписать первую)
	rb.Add(4.0, nil)

	if rb.Size() != 3 {
		t.Errorf("Expected size 3, got %d", rb.Size())
	}

	data := rb.GetAll()
	if len(data) != 3 {
		t.Errorf("Expected 3 data points, got %d", len(data))
	}

	// Проверяем что первая запись перезаписалась
	if data[0].Value != 2.0 {
		t.Errorf("Expected first value 2.0, got %f", data[0].Value)
	}
}

func TestRingBuffer_GetAll(t *testing.T) {
	rb := NewRingBuffer(5)

	// Добавляем несколько записей
	for i := 1; i <= 3; i++ {
		rb.Add(float64(i), nil)
	}

	data := rb.GetAll()

	if len(data) != 3 {
		t.Errorf("Expected 3 data points, got %d", len(data))
	}

	// Проверяем порядок (старые -> новые)
	for i, dp := range data {
		expected := float64(i + 1)
		if dp.Value != expected {
			t.Errorf("Expected value %f at index %d, got %f", expected, i, dp.Value)
		}
	}
}

func TestRingBuffer_GetLast(t *testing.T) {
	rb := NewRingBuffer(10)

	// Добавляем 5 записей
	for i := 1; i <= 5; i++ {
		rb.Add(float64(i), nil)
	}

	// Получаем последние 3
	last := rb.GetLast(3)

	if len(last) != 3 {
		t.Errorf("Expected 3 data points, got %d", len(last))
	}

	// Проверяем что это действительно последние записи
	expectedValues := []float64{3.0, 4.0, 5.0}
	for i, dp := range last {
		if dp.Value != expectedValues[i] {
			t.Errorf("Expected value %f at index %d, got %f", expectedValues[i], i, dp.Value)
		}
	}
}

func TestRingBuffer_GetRange(t *testing.T) {
	rb := NewRingBuffer(100)

	// Добавляем записи с известными timestamps
	now := time.Now()
	for i := 0; i < 10; i++ {
		rb.Add(float64(i), nil)
		time.Sleep(10 * time.Millisecond) // Небольшая задержка для разных timestamps
	}

	// Получаем данные за последние 50ms
	from := now.Add(-50 * time.Millisecond)
	to := now.Add(100 * time.Millisecond)

	data := rb.GetRange(from, to)

	if len(data) == 0 {
		t.Error("Expected some data points in range")
	}

	// Проверяем что все записи в диапазоне
	for _, dp := range data {
		if dp.Timestamp.Before(from) || dp.Timestamp.After(to) {
			t.Errorf("Data point timestamp %v outside range [%v, %v]", dp.Timestamp, from, to)
		}
	}
}

func TestRingBuffer_Clear(t *testing.T) {
	rb := NewRingBuffer(5)

	// Добавляем данные
	for i := 0; i < 3; i++ {
		rb.Add(float64(i), nil)
	}

	if rb.Size() != 3 {
		t.Errorf("Expected size 3 before clear, got %d", rb.Size())
	}

	// Очищаем
	rb.Clear()

	if rb.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", rb.Size())
	}

	if rb.IsFull() {
		t.Error("Expected buffer not to be full after clear")
	}

	data := rb.GetAll()
	if len(data) != 0 {
		t.Errorf("Expected empty data after clear, got %d points", len(data))
	}
}

func TestRingBuffer_Concurrency(t *testing.T) {
	rb := NewRingBuffer(1000)
	done := make(chan bool)

	// Запускаем несколько горутин для одновременной записи
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				rb.Add(float64(id*100+j), map[string]string{"goroutine": string(rune(id))})
			}
			done <- true
		}(i)
	}

	// Ждем завершения всех горутин
	for i := 0; i < 10; i++ {
		<-done
	}

	// Проверяем что данные записались
	if rb.Size() == 0 {
		t.Error("Expected data in buffer after concurrent writes")
	}

	// Buffer не должен превысить capacity
	if rb.Size() > rb.Capacity() {
		t.Errorf("Buffer size %d exceeded capacity %d", rb.Size(), rb.Capacity())
	}
}

func TestRingBuffer_Labels(t *testing.T) {
	rb := NewRingBuffer(10)

	// Добавляем данные с labels
	labels := map[string]string{
		"endpoint": "/api/test",
		"method":   "GET",
	}

	rb.Add(1.0, labels)

	data := rb.GetAll()
	if len(data) != 1 {
		t.Fatalf("Expected 1 data point, got %d", len(data))
	}

	// Проверяем что labels сохранились
	if data[0].Labels["endpoint"] != "/api/test" {
		t.Errorf("Expected endpoint label '/api/test', got '%s'", data[0].Labels["endpoint"])
	}

	if data[0].Labels["method"] != "GET" {
		t.Errorf("Expected method label 'GET', got '%s'", data[0].Labels["method"])
	}
}

