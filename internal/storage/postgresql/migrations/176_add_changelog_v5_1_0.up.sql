INSERT INTO changelogs (version, release_date, content) VALUES
('5.1.0', '2026-03-26', '## [5.1.0] - 2026-03-26

### Added
- **Inference Container Parameters**: ~27 new configurable parameters across all 5 inference providers with dedicated UI controls and tooltips
- **vLLM**: quantization, dtype, kv_cache_dtype, max_num_seqs, enforce_eager, enable_prefix_caching, enable_chunked_prefill, swap_space
- **SGLang**: data_parallel, context_len, chunked_prefill, quantization, attention_backend, extra_args
- **TGI**: max_concurrent_reqs, max_input_len, max_total_tokens, quantize, cuda_memory_fraction, extra_args
- **TEI**: max_batch_tokens, max_concurrent_reqs, pooling, dtype, extra_args
- **llama.cpp**: batch_size, ubatch_size, cache_type_k/v, mlock
- **FormLabel Tooltips**: All ~41 inference parameters now have tooltip icons with descriptions

### Changed
- Replaced native title tooltips with FormLabel + Tooltip components across models admin page
- TEI provider now has its own configuration section')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
