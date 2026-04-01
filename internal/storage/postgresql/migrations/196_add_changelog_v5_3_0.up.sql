INSERT INTO changelogs (version, release_date, content) VALUES
('5.3.0', '2026-04-01', '## [5.3.0] - 2026-04-01

### Security
- **CVE-2024-41110 (CRITICAL)**: updated github.com/docker/docker v25.0.5 -> v25.0.13 (AuthZ bypass + firewalld bridge isolation)
- **CVE-2025-66630 (CRITICAL)**: updated github.com/gofiber/fiber/v2 v2.52.9 -> v2.52.11 (predictable UUIDs on crypto/rand error)
- **CVE-2026-24051 (HIGH)**: updated go.opentelemetry.io/otel/sdk v1.38.0 -> v1.40.0 + all related otel packages (PATH hijacking on macOS)
- **CVE-2024-24792 / GO-2026-4815 (HIGH)**: updated golang.org/x/image -> v0.38.0 (OOM on malformed TIFF IFD offset)
- **GO-2025-3829**: fixed firewalld reload bridge network isolation in docker/docker v25.0.13
- **@sveltejs/kit CVEs**: updated 2.49.2 -> 2.49.5+ (SSRF + DoS in prerendering)
- **kysely CVEs**: updated 0.27.6 -> 0.28.14 (SQL injection in JSON path and string literals)
- **devalue, picomatch, rollup**: updated to patched versions (DoS / ReDoS / path traversal)

### Added
- Security CI stage: security:trivy (SARIF -> GitLab Security Dashboard) + security:govulncheck (callgraph CVE analysis)

### Fixed
- Inference Evict: StopByAlias added to ContainerRuntime interface; orphan container cleanup on StatusStarting race; Router.Evict holds alias lock')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
