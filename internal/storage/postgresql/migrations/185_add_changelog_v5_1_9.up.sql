INSERT INTO changelogs (version, release_date, content) VALUES
('5.1.9', '2026-03-30', '## [5.1.9] - 2026-03-30

### Fixed

- **TEI CPU Mode: --gpus all Still Passed**: Despite selecting CPU Mode and using the cpu-1.8 image, Docker runtime unconditionally added --gpus all. CPU-only image crashed immediately when NVIDIA runtime was injected. Added CPUOnly bool field to ContainerStartRequest; when set, GPU device requests and --gpus flag are completely omitted from both Docker SDK and CLI paths.')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
