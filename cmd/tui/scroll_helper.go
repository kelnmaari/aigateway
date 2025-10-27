package main

import (
	"strings"
)

// applyScroll применяет прокрутку к контенту
// Возвращает видимую часть контента с учетом offset и высоты экрана
func applyScroll(content string, scrollOffset int, visibleHeight int) string {
	lines := strings.Split(content, "\n")

	// Если контент помещается целиком, возвращаем как есть
	if len(lines) <= visibleHeight {
		return content
	}

	// Ограничиваем offset
	maxOffset := len(lines) - visibleHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	if scrollOffset > maxOffset {
		scrollOffset = maxOffset
	}
	if scrollOffset < 0 {
		scrollOffset = 0
	}

	// Вырезаем видимую часть
	endLine := scrollOffset + visibleHeight
	if endLine > len(lines) {
		endLine = len(lines)
	}

	visibleLines := lines[scrollOffset:endLine]

	// Добавляем индикаторы прокрутки
	result := strings.Join(visibleLines, "\n")

	// Добавляем индикатор что можно прокручивать
	if scrollOffset > 0 {
		result = "⬆️  Прокрутите вверх (↑/k/PgUp/Home)\n" + result
	}
	if endLine < len(lines) {
		result = result + "\n⬇️  Прокрутите вниз (↓/j/PgDn)"
	}

	return result
}

// splitIntoColumns разделяет контент на N колонок
func splitIntoColumns(content string, columns int, columnWidth int) string {
	lines := strings.Split(content, "\n")

	if columns <= 1 || len(lines) < 10 {
		return content
	}

	// Вычисляем количество строк на колонку
	linesPerColumn := (len(lines) + columns - 1) / columns

	// Разделяем на колонки
	cols := make([][]string, columns)
	for i := 0; i < columns; i++ {
		start := i * linesPerColumn
		end := start + linesPerColumn
		if end > len(lines) {
			end = len(lines)
		}
		if start < len(lines) {
			cols[i] = lines[start:end]
		}
	}

	// Собираем колонки в строки
	maxRows := 0
	for _, col := range cols {
		if len(col) > maxRows {
			maxRows = len(col)
		}
	}

	result := make([]string, maxRows)
	for row := 0; row < maxRows; row++ {
		rowParts := make([]string, columns)
		for col := 0; col < columns; col++ {
			if row < len(cols[col]) {
				// Обрезаем строку до ширины колонки
				line := cols[col][row]
				if len(line) > columnWidth {
					line = line[:columnWidth-3] + "..."
				}
				// Дополняем пробелами до ширины колонки
				rowParts[col] = line + strings.Repeat(" ", columnWidth-len(line))
			} else {
				rowParts[col] = strings.Repeat(" ", columnWidth)
			}
		}
		result[row] = strings.Join(rowParts, " │ ")
	}

	return strings.Join(result, "\n")
}

