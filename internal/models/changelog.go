// Package models содержит структуры данных для changelog системы
package models

import "time"

// Changelog представляет запись об изменениях в версии
type Changelog struct {
	Version     string    `json:"version"`
	ReleaseDate time.Time `json:"release_date"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
}

// ChangelogListResponse ответ со списком изменений
type ChangelogListResponse struct {
	Changelogs []*Changelog `json:"changelogs"`
	Total      int          `json:"total"`
}
