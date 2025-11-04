INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('2.4.8', '2025-11-04', '## [2.4.8] - 2025-11-04

### Added

- **Embeddings & Vector Search** 🧮:
  - Ollama embeddings integration (`mxbai-embed-large`, `nomic-embed-text`)
  - pgvector store for similarity search with HNSW index
  - Automatic index creation with configurable parameters (m=16, ef_construction=64)
  - Connection string support for direct PostgreSQL connection
  - Configurable dimensions (768, 1024) and distance metrics (cosine, l2, inner_product)
  - **RAG Worker embeddings generation**:
    - Batch embeddings for all RAG data sources (API, DB, Web)
    - Automatic vector storage in pgvector after chunk creation
    - Graceful fallback to simple mode if embedder not configured
    - `generateAndStoreEmbeddings()` method for efficient processing
  - Full vs Simple mode: seamless operation with or without embeddings
  - RAG Orchestrator integrates embedder and vector store for enhanced search

- **RAG Worker Full Implementation** 🤖:
  - Complete API Sync implementation (`internal/rag/worker/api_sync.go`)
    - REST API data synchronization (GET/POST methods)
    - JSON and plain text response parsing
    - Configurable `data_path` extraction (e.g., "data.items")
    - Custom `text_fields` selection for chunking
    - Batch processing for array responses
  - Complete Database Sync implementation (`internal/rag/worker/db_sync.go`)
    - PostgreSQL and MySQL database connectivity
    - SQL query execution with full row scanning
    - Dynamic connection string building from config + encrypted credentials
    - Automatic document and chunk creation from query results
  - Complete Web Scraping implementation (`internal/rag/worker/web_scrape.go`)
    - HTML page scraping with User-Agent spoofing
    - Text extraction from HTML (script/style tag removal)
    - HTML entities decoding (&nbsp;, &lt;, &mdash;, etc.)
    - Whitespace cleanup and normalization
    - Multiple URLs batch processing
    - Partial success handling (some URLs may fail)
  - **Chunking Strategies**:
    - **Fixed**: Fixed-size chunks with configurable overlap
    - **Sentence**: Smart sentence-based chunking (punctuation detection)
    - **Paragraph**: Paragraph-based chunking (double newline split)
  - Token counting: Approximate `tokens = len(text) / 4`
  - RAG Worker logging to separate file: `logs/rag-worker.log`

- **UI Improvements** 🎨:
  - **RAG Data Sources Checkboxes**:
    - Replaced confusing multi-select dropdown with intuitive checkboxes
    - Each source shows name and type badge (api/database/web)
    - Hover effects and visual feedback
    - No more "Hold Ctrl/Cmd" requirement
    - Scrollable list for many sources (max-height: 150px)
  - **Thinking Spoiler for Reasoning Models**:
    - Compact single-line spoiler with expand/collapse toggle (DeepSeek-R1, o1-preview)
    - Dark theme styling (blue accent, #1a1a2e background)
    - `<think>` and `<reasoning>` tag support in streaming chunks
    - Frontend extraction and rendering without breaking markdown

### Changed

- **RAG Job Processing** 🔧:
  - Job queue polling interval: 5 seconds
  - Max retry attempts: 3
  - Job locking: 5 minutes timeout
  - Status flow: `pending` → `processing` → `completed`/`failed`
  - Automatic expired job unlocking

- **RAG Data Source Configuration**:
  - **API Sources**: `url`, `method`, `headers`, `data_path`, `text_fields`
  - **Database Sources**: `connection_string`, `query`, `database_type`
  - **Web Sources**: `url` or `urls` (array)
  - **Indexing Config**: `chunk_strategy`, `max_chunk_size` (default: 500), `chunk_overlap` (default: 50)

### Fixed

- **RAG Data Source Update Config Loss** 🔧:
  - Fixed critical bug where editing RAG source would lose advanced config fields
  - Frontend now preserves existing config when editing

- **RAG Sync Deduplication** 🔄:
  - Implemented `DeleteChunksBySource` to prevent data duplication on repeated syncs
  - Each sync now deletes old chunks/vectors/documents before creating new ones
  - Applied to all sync types: API, Database, Web Scraping

- **RAG Chunk ID Collisions** 🔧:
  - Changed chunk ID generation from `timestamp+random` to **UUID v4**
  - Prevents collisions during high-speed chunk creation (100+ chunks/sec)

### Technical

- **File Structure**:
  - `internal/rag/worker/`: Main worker loop and job dispatching
  - `internal/rag/processor/`: Document processing pipeline
- **Documentation**: New comprehensive guide: `docs/RAG_WORKER_GUIDE.md`');

