# vLLM + TurboQuant Docker Image

Custom Docker image based on `vllm/vllm-openai` with the **TurboQuant** plugin
for KV-cache compression (up to 3.76× with asymmetric K/V support).

## Build

```bash
# From repo root
docker build \
  --build-arg VLLM_VERSION=v0.18.0 \
  --build-arg TURBOQUANT_VERSION=1.4.1 \
  -t aigateway/vllm-turboquant:v0.18.0-tq1.4.1 \
  -f packaging/docker/vllm-turboquant/Dockerfile .

# Tag as latest for convenience
docker tag aigateway/vllm-turboquant:v0.18.0-tq1.4.1 aigateway/vllm-turboquant:latest
```

## Usage via AIGateway UI

1. Go to **Admin → Models → Load Model**
2. Set **Provider**: `vllm`
3. Set **Docker Image Override**: `aigateway/vllm-turboquant:latest`
4. Enable **TurboQuant** checkbox
5. Configure **K Bits** (default: 4) and **V Bits** (default: 3)
6. Click **Load & Start**

AIGateway will:
- Launch container using the `vllm-turboquant` image (local, no pull)
- Inject `TQ4_K_BITS` / `TQ4_V_BITS` as environment variables
- Append `--attention-backend CUSTOM` to the vLLM command

## Push to Worker

To use on a remote agent, push the local image via admin UI:

```bash
# Admin → Workers → [worker] → Images → Push
# Or via API:
curl -X POST https://aigateway/api/admin/workers/:id/images/push \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"image": "aigateway/vllm-turboquant:latest"}'
```

AIGateway runs `docker save | docker load` through the encrypted agent channel.

## Supported Hardware

- **NVIDIA RTX A6000** / SM 8.6
- **NVIDIA GB10** / SM 12.1
- Other CUDA GPUs (fallback to standard vLLM attention)

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `TQ4_K_BITS` | `4` | Bits per key element in KV cache (2-8) |
| `TQ4_V_BITS` | `3` | Bits per value element in KV cache (2-8) |

Asymmetric K/V: keys can be higher precision than values because attention is more
sensitive to key quantization errors. Recommended: `K=4, V=3` or `K=4, V=2`.

## References

- Article: https://dev.to/albertocodes/compressed-vlm-inference-from-a-single-containerfile-turboquant-vllm-v11-35bm
- PyPI: https://pypi.org/project/turboquant-vllm/
- GitHub: https://github.com/mitkox/vllm-turboquant
