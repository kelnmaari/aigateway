# syntax=docker/dockerfile:1.6
# vLLM with TurboQuant+ (varjoranta fork) — Weight + KV + MoE compression
#
# This image bundles the turboquant-plus-vllm plugin from Varjosoft Oy:
#   https://github.com/varjoranta/turboquant-vllm
#
# Unlike the basic TurboQuant (Alberto-Codes), this variant supports:
#   - TQ3 weight compression  (3-bit, 4.3-4.6x, zero calibration)
#   - KV cache compression    (asymmetric K/V, up to 3.7x)
#   - MoE expert pruning      (REAP, router-weighted saliency)
#
# Supported GPUs: A100 (SM80), RTX 6000 Ada (SM89), H100 (SM90)
# RTX PRO 6000 Blackwell is NOT supported (needs FlashAttention-4).
#
# Build:
#   docker build \
#     --build-arg VLLM_VERSION=v0.18.0 \
#     --build-arg TQ_PLUS_REF=main \
#     -t aigateway/vllm-turboquant-plus:v0.18.0 \
#     -f packaging/docker/vllm-turboquant-plus/Dockerfile .
#
# Run (KV cache only — Qwen3 / Llama):
#   docker run --gpus all -p 8000:8000 \
#     -e TQ_KV_ENABLED=1 -e TQ_K_BITS=4 -e TQ_V_BITS=3 \
#     aigateway/vllm-turboquant-plus:v0.18.0 \
#     --model Qwen/Qwen3-7B-Instruct --host 0.0.0.0 --port 8000
#
# Run (weight + KV compression — Gemma 26B on 24GB VRAM):
#   docker run --gpus all -p 8000:8000 \
#     -e TQ_WEIGHT_BITS=3 -e TQ_KV_ENABLED=1 -e TQ_K_BITS=4 -e TQ_V_BITS=3 \
#     aigateway/vllm-turboquant-plus:v0.18.0 \
#     --model google/gemma-4-26b --host 0.0.0.0 --port 8000
#
# Run (MoE pruning — Qwen3-30B-A3B with 20% experts pruned):
#   docker run --gpus all -p 8000:8000 \
#     -e TQ_KV_ENABLED=1 -e TQ_REAP_PRUNE_FRACTION=0.2 \
#     aigateway/vllm-turboquant-plus:v0.18.0 \
#     --model Qwen/Qwen3-30B-A3B --host 0.0.0.0 --port 8000

ARG VLLM_VERSION=v0.18.0
FROM vllm/vllm-openai:${VLLM_VERSION}

ARG TQ_PLUS_REF=main

# The plugin isn't published on PyPI yet — install directly from git.
# The [vllm] extra is omitted because vLLM is already in the base image.
RUN pip install --no-cache-dir \
    "turboquant-plus-vllm @ git+https://github.com/varjoranta/turboquant-vllm.git@${TQ_PLUS_REF}"

# Verify the plugin importable at build time
RUN python -c "\
import turboquant_vllm; \
print('turboquant-plus-vllm loaded from', turboquant_vllm.__file__); \
from turboquant_vllm import patch_vllm_attention, enable_weight_quantization; \
print('exports: patch_vllm_attention, enable_weight_quantization')"

# Install the entrypoint wrapper that applies Python-API patches
# before handing off to `vllm serve`.
COPY packaging/docker/vllm-turboquant-plus/entrypoint.py /opt/turboquant/entrypoint.py
RUN chmod +x /opt/turboquant/entrypoint.py

# Default environment: everything disabled — user must opt in explicitly.
ENV TQ_WEIGHT_BITS="" \
    TQ_KV_ENABLED="" \
    TQ_K_BITS=4 \
    TQ_V_BITS=3 \
    TQ_NORM_CORRECTION=1 \
    TQ_REAP_PRUNE_FRACTION=""

LABEL org.opencontainers.image.source="https://github.com/varjoranta/turboquant-vllm" \
      org.opencontainers.image.description="vLLM with TurboQuant+ (weights + KV cache + MoE pruning)" \
      org.opencontainers.image.vendor="AIGateway" \
      aigateway.plugin="turboquant-plus" \
      aigateway.features="weight-quant,kv-compression,moe-pruning"

# Override the base image entrypoint with our wrapper
ENTRYPOINT ["python", "/opt/turboquant/entrypoint.py"]
