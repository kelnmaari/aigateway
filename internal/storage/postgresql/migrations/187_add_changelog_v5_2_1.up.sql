INSERT INTO changelogs (version, release_date, content) VALUES
('5.2.1', '2026-03-30', '## [5.2.1] - 2026-03-30

### Added
- **Custom Docker Image Override**: New docker_image field in model configuration. Set any image name to override the provider default (e.g. vllm-custom:patched). Works for all providers: vLLM, SGLang, TGI, TEI, llama.cpp, TensorRT-LLM.

### Changed
- **Skip pull if image exists locally**: Docker runtime no longer pulls an image if it already exists locally. Images are pulled only on first use. Allows using locally-built custom images without a Docker registry.

### Technical
- types.go — DockerImage string field in ModelSpec
- model_store.go — DockerImage in SavedModel, SaveFromSpec, ToSpec
- provider_builders.go — DockerImage override in all 6 provider builders
- runtime_docker.go — ImageExists check before pull; skips pull when image present
- inference_handler.go — docker_image in LoadRequest, UpdateSavedRequest, CreateSavedRequest, SavedModelResponse
- inference.ts — docker_image in all 4 TypeScript interfaces
- +page.svelte — Docker Image input in Add Model and Edit Saved Model forms')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
