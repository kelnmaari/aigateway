INSERT INTO changelogs (version, release_date, content) VALUES
('5.2.0', '2026-03-30', '## [5.2.0] - 2026-03-30

### Added
- **ONNX Export Feature**: Full UI + backend for exporting HuggingFace models to ONNX format. Enables TEI to run reranker/embedding models in CPU-only mode when ONNX is required (e.g. bge-reranker-v2-m3 which crashes on CPU without ONNX). Export runs via optimum-cli inside a python:3.11-slim Docker container, mounts the HF cache and writes ONNX files directly into {snapshot_dir}/onnx/ so TEI finds them automatically. Features: task selector, dtype selector (float32/float16/int8/uint8), live log streaming with auto-scroll, job status tracking. ONNX button appears on TEI provider model cards in both Running Models and Saved Models lists.

### Technical
- internal/inference/onnx_exporter.go — new OnnxExporter service with StartExport, GetJob, ListJobs methods; Docker-based export with 45-minute timeout and streaming logs
- internal/api/handlers/inference_handler.go — PostStartOnnxExport, GetOnnxExportJob, GetOnnxExportJobs handlers
- internal/api/router/router.go — POST/GET /api/system/inference/onnx-export routes
- web-svelte/src/lib/api/inference.ts — OnnxExportRequest, OnnxExportJob types and API methods
- web-svelte/src/routes/(protected)/admin/models/+page.svelte — ONNX export modal with form, job polling, log viewer')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
