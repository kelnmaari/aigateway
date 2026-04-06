# syntax=docker/dockerfile:1.6
# vLLM with TurboQuant — KV-cache compression plugin (up to 3.76x)
#
# Based on: https://dev.to/albertocodes/compressed-vlm-inference-from-a-single-containerfile-turboquant-vllm-v11-35bm
# Plugin:   https://pypi.org/project/turboquant-vllm/
# Upstream: https://github.com/mitkox/vllm-turboquant
#
# Build:
#   docker build \
#     --build-arg VLLM_VERSION=v0.18.0 \
#     --build-arg TURBOQUANT_VERSION=1.4.1 \
#     -t aigateway/vllm-turboquant:v0.18.0-tq1.4.1 \
#     -f packaging/docker/vllm-turboquant/Dockerfile .
#
# Run:
#   docker run --gpus all -p 8000:8000 \
#     -e TQ4_K_BITS=4 -e TQ4_V_BITS=3 \
#     aigateway/vllm-turboquant:v0.18.0-tq1.4.1 \
#     --model meta-llama/Llama-3.1-8B-Instruct \
#     --attention-backend CUSTOM

ARG VLLM_VERSION=v0.18.0
FROM vllm/vllm-openai:${VLLM_VERSION}

ARG TURBOQUANT_VERSION=1.4.1

# Install TurboQuant plugin. The [vllm] extra is NOT added because vLLM
# is already installed in the base image (adding it would reinstall).
RUN pip install --no-cache-dir "turboquant-vllm==${TURBOQUANT_VERSION}"

# Verify plugin registration at build time. vLLM's entry-point discovery
# fails silently — this assert catches misconfiguration before deployment.
RUN python -c "\
from importlib.metadata import entry_points; \
eps = [ep for ep in entry_points(group='vllm.platform_plugins') if 'turbo' in ep.name.lower()] \
      + [ep for ep in entry_points(group='vllm.general_plugins') if 'turbo' in ep.name.lower()]; \
print('Discovered TurboQuant entry points:', [ep.name for ep in eps]); \
assert len(eps) >= 1, 'TurboQuant entry point not found — plugin registration failed'"

# Default environment: asymmetric 4-bit keys / 3-bit values.
# Override at runtime with -e TQ4_K_BITS=X -e TQ4_V_BITS=Y.
ENV TQ4_K_BITS=4 \
    TQ4_V_BITS=3

LABEL org.opencontainers.image.source="https://github.com/mitkox/vllm-turboquant" \
      org.opencontainers.image.description="vLLM with TurboQuant KV-cache compression plugin" \
      org.opencontainers.image.vendor="AIGateway" \
      aigateway.plugin="turboquant" \
      aigateway.turboquant.version="${TURBOQUANT_VERSION}"

# Entrypoint inherited from base image — just run `vllm serve ...`
