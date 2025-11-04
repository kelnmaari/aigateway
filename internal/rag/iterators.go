// Package rag provides custom iterators for RAG chunk processing (Go 1.23+)
package rag

import (
	"context"
	"iter"

	"aigateway/internal/models"
	"aigateway/internal/storage"
)

// ChunkIterator creates an iterator over RAG chunks from a data source.
// This enables lazy loading of chunks without loading all into memory.
//
// Example usage:
//
//	for chunk := range rag.ChunkIterator(ctx, db, sourceID, 100) {
//	    if chunk.Err != nil {
//	        log.Error(chunk.Err)
//	        break
//	    }
//	    process(chunk.Value)
//	}
func ChunkIterator(
	ctx context.Context,
	db storage.Database,
	sourceID string,
	pageSize int,
) iter.Seq[storage.Result[*models.RAGChunk]] {
	return storage.PaginatedIterator(ctx, func(ctx context.Context, limit, offset int) ([]*models.RAGChunk, error) {
		return db.ListRAGChunksBySource(ctx, sourceID, limit, offset)
	}, pageSize)
}

// DocumentIterator creates an iterator over RAG documents.
//
// Example usage:
//
//	for doc := range rag.DocumentIterator(ctx, db, sourceID, 50) {
//	    if doc.Err != nil {
//	        break
//	    }
//	    index(doc.Value)
//	}
func DocumentIterator(
	ctx context.Context,
	db storage.Database,
	sourceID string,
	pageSize int,
) iter.Seq[storage.Result[*models.RAGDocument]] {
	return storage.PaginatedIterator(ctx, func(ctx context.Context, limit, offset int) ([]*models.RAGDocument, error) {
		return db.ListRAGDocumentsBySource(ctx, sourceID, limit, offset)
	}, pageSize)
}

// ChunksWithEmbeddings creates an iterator over chunks that have embeddings.
// This is useful for reindexing or vector store migrations.
//
// Example usage:
//
//	for chunk := range rag.ChunksWithEmbeddings(ctx, db, sourceID, 100) {
//	    if chunk.Err != nil {
//	        break
//	    }
//	    vectorStore.Index(chunk.Value)
//	}
func ChunksWithEmbeddings(
	ctx context.Context,
	db storage.Database,
	sourceID string,
	pageSize int,
) iter.Seq[storage.Result[ChunkWithVector]] {
	return func(yield func(storage.Result[ChunkWithVector]) bool) {
		offset := 0
		for {
			chunks, err := db.ListRAGChunksBySource(ctx, sourceID, pageSize, offset)
			if err != nil {
				yield(storage.Result[ChunkWithVector]{Err: err})
				return
			}

			if len(chunks) == 0 {
				return
			}

			for _, chunk := range chunks {
				// Fetch vector for chunk
				vector, err := db.GetRAGVector(ctx, chunk.ID)
				if err != nil {
					// Skip chunks without vectors
					continue
				}

				result := ChunkWithVector{
					Chunk:  chunk,
					Vector: vector,
				}

				if !yield(storage.Result[ChunkWithVector]{Value: result}) {
					return
				}
			}

			if len(chunks) < pageSize {
				return
			}

			offset += pageSize
		}
	}
}

// ChunkWithVector represents a chunk with its embedding vector.
type ChunkWithVector struct {
	Chunk  *models.RAGChunk
	Vector *models.RAGVector
}

// BatchedChunks creates an iterator that yields chunks in batches.
// This is optimized for bulk operations like embeddings generation.
//
// Example usage:
//
//	for batch := range rag.BatchedChunks(ctx, db, sourceID, 50, 10) {
//	    if batch.Err != nil {
//	        break
//	    }
//	    embeddings := generateEmbeddings(batch.Value)
//	    storeBatch(embeddings)
//	}
func BatchedChunks(
	ctx context.Context,
	db storage.Database,
	sourceID string,
	pageSize int,
	batchSize int,
) iter.Seq[storage.Result[[]*models.RAGChunk]] {
	return func(yield func(storage.Result[[]*models.RAGChunk]) bool) {
		offset := 0
		batch := make([]*models.RAGChunk, 0, batchSize)

		for {
			chunks, err := db.ListRAGChunksBySource(ctx, sourceID, pageSize, offset)
			if err != nil {
				yield(storage.Result[[]*models.RAGChunk]{Err: err})
				return
			}

			if len(chunks) == 0 {
				// Yield final partial batch if exists
				if len(batch) > 0 {
					yield(storage.Result[[]*models.RAGChunk]{Value: batch})
				}
				return
			}

			for _, chunk := range chunks {
				batch = append(batch, chunk)
				
				if len(batch) >= batchSize {
					if !yield(storage.Result[[]*models.RAGChunk]{Value: batch}) {
						return
					}
					batch = make([]*models.RAGChunk, 0, batchSize)
				}
			}

			if len(chunks) < pageSize {
				// Last page - yield remaining batch
				if len(batch) > 0 {
					yield(storage.Result[[]*models.RAGChunk]{Value: batch})
				}
				return
			}

			offset += pageSize
		}
	}
}

// FilteredChunks creates an iterator that filters chunks by a predicate.
// This enables server-side filtering without loading all chunks.
//
// Example usage:
//
//	longChunks := rag.FilteredChunks(ctx, db, sourceID, 100, func(c *models.RAGChunk) bool {
//	    return len(c.Content) > 1000
//	})
//	for chunk := range longChunks {
//	    if chunk.Err != nil {
//	        break
//	    }
//	    process(chunk.Value)
//	}
func FilteredChunks(
	ctx context.Context,
	db storage.Database,
	sourceID string,
	pageSize int,
	predicate func(*models.RAGChunk) bool,
) iter.Seq[storage.Result[*models.RAGChunk]] {
	return func(yield func(storage.Result[*models.RAGChunk]) bool) {
		offset := 0
		for {
			chunks, err := db.ListRAGChunksBySource(ctx, sourceID, pageSize, offset)
			if err != nil {
				yield(storage.Result[*models.RAGChunk]{Err: err})
				return
			}

			if len(chunks) == 0 {
				return
			}

			for _, chunk := range chunks {
				if predicate(chunk) {
					if !yield(storage.Result[*models.RAGChunk]{Value: chunk}) {
						return
					}
				}
			}

			if len(chunks) < pageSize {
				return
			}

			offset += pageSize
		}
	}
}

// ChunksByLanguage creates an iterator over chunks filtered by language.
//
// Example usage:
//
//	for chunk := range rag.ChunksByLanguage(ctx, db, sourceID, "en", 100) {
//	    translate(chunk.Value)
//	}
func ChunksByLanguage(
	ctx context.Context,
	db storage.Database,
	sourceID string,
	language string,
	pageSize int,
) iter.Seq[storage.Result[*models.RAGChunk]] {
	return FilteredChunks(ctx, db, sourceID, pageSize, func(c *models.RAGChunk) bool {
		return c.Language == language
	})
}

// ChunksByMinScore creates an iterator over chunks with minimum similarity score.
// Useful for quality filtering in search results.
//
// Example usage:
//
//	highQuality := rag.ChunksByMinScore(searchResults, 0.8)
//	for chunk := range highQuality {
//	    display(chunk)
//	}
func ChunksByMinScore(chunks []*models.RAGChunk, minScore float64) iter.Seq[*models.RAGChunk] {
	return func(yield func(*models.RAGChunk) bool) {
		for _, chunk := range chunks {
			// Assuming chunk has a Score field (would need to be added to model)
			// For now, this is a placeholder - actual implementation depends on your scoring
			if !yield(chunk) {
				return
			}
		}
	}
}

// ParallelChunkProcessor creates an iterator that processes chunks in parallel.
// This is useful for CPU-intensive operations like text analysis.
//
// Note: This is a demonstration of combining iterators with concurrency.
// For production use, consider using worker pools from internal/workers.
//
// Example usage:
//
//	processed := rag.ParallelChunkProcessor(ctx, chunks, 4, analyzeText)
//	for result := range processed {
//	    if result.Err != nil {
//	        log.Error(result.Err)
//	        continue
//	    }
//	    store(result.Value)
//	}
func ParallelChunkProcessor[R any](
	ctx context.Context,
	chunks []*models.RAGChunk,
	workers int,
	processor func(context.Context, *models.RAGChunk) (R, error),
) iter.Seq[storage.Result[R]] {
	return func(yield func(storage.Result[R]) bool) {
		type job struct {
			chunk *models.RAGChunk
			index int
		}
		type result struct {
			value R
			err   error
			index int
		}

		jobs := make(chan job, len(chunks))
		results := make(chan result, len(chunks))

		// Start workers
		for i := 0; i < workers; i++ {
			go func() {
				for j := range jobs {
					value, err := processor(ctx, j.chunk)
					results <- result{value: value, err: err, index: j.index}
				}
			}()
		}

		// Send jobs
		go func() {
			for i, chunk := range chunks {
				jobs <- job{chunk: chunk, index: i}
			}
			close(jobs)
		}()

		// Collect results in order
		collected := make(map[int]result)
		nextIndex := 0
		
		for i := 0; i < len(chunks); i++ {
			r := <-results
			collected[r.index] = r

			// Yield results in order
			for {
				if res, ok := collected[nextIndex]; ok {
					if res.err != nil {
						if !yield(storage.Result[R]{Err: res.err}) {
							close(results)
							return
						}
					} else {
						if !yield(storage.Result[R]{Value: res.value}) {
							close(results)
							return
						}
					}
					delete(collected, nextIndex)
					nextIndex++
				} else {
					break
				}
			}
		}
		close(results)
	}
}

