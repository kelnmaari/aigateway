#!/usr/bin/env python3
"""
TurboQuant+ entrypoint wrapper for vLLM.

Applies Python-API patches from turboquant-plus-vllm (varjoranta fork)
based on environment variables, then hands off to `vllm serve`.

Environment variables:
    TQ_WEIGHT_BITS          int   — enable TQ3 weight compression (e.g. "3")
    TQ_KV_ENABLED           bool  — enable KV cache compression ("1", "true")
    TQ_K_BITS               int   — bits per key element (2-8, default 4)
    TQ_V_BITS               int   — bits per value element (2-8, default 3)
    TQ_NORM_CORRECTION      bool  — norm correction ("1", default on)
    TQ_REAP_PRUNE_FRACTION  float — MoE expert prune fraction (0.0-0.5)

After applying patches, this script execs `vllm serve <passed-args>`.
"""
from __future__ import annotations

import os
import sys
import logging

logging.basicConfig(
    format="[turboquant-plus-entrypoint] %(levelname)s: %(message)s",
    level=logging.INFO,
)
log = logging.getLogger("entrypoint")


def _parse_bool(value: str | None) -> bool:
    if value is None:
        return False
    return value.strip().lower() in ("1", "true", "yes", "on")


def _parse_int(value: str | None) -> int | None:
    if not value:
        return None
    try:
        return int(value)
    except ValueError:
        log.warning("invalid int value %r — ignoring", value)
        return None


def _parse_float(value: str | None) -> float | None:
    if not value:
        return None
    try:
        return float(value)
    except ValueError:
        log.warning("invalid float value %r — ignoring", value)
        return None


def apply_patches() -> None:
    """Apply turboquant-plus-vllm patches based on environment variables."""

    weight_bits = _parse_int(os.environ.get("TQ_WEIGHT_BITS"))
    kv_enabled = _parse_bool(os.environ.get("TQ_KV_ENABLED"))
    k_bits = _parse_int(os.environ.get("TQ_K_BITS")) or 4
    v_bits = _parse_int(os.environ.get("TQ_V_BITS")) or 3
    norm_correction = _parse_bool(os.environ.get("TQ_NORM_CORRECTION"))
    reap_fraction = _parse_float(os.environ.get("TQ_REAP_PRUNE_FRACTION"))

    if not (weight_bits or kv_enabled or reap_fraction):
        log.info("no TurboQuant+ features enabled — running vanilla vLLM")
        return

    # Import lazily so import errors are caught and surfaced clearly
    try:
        import turboquant_vllm
    except ImportError as exc:
        log.error("turboquant-plus-vllm not installed: %s", exc)
        sys.exit(1)

    # 1) Weight compression (applied to the model as it loads)
    if weight_bits:
        log.info("enabling weight quantization: bits=%d", weight_bits)
        try:
            turboquant_vllm.enable_weight_quantization(bits=weight_bits)
        except Exception as exc:  # noqa: BLE001
            log.error("enable_weight_quantization failed: %s", exc)
            sys.exit(2)

    # 2) KV cache compression (patches attention kernels)
    if kv_enabled:
        log.info(
            "patching vLLM attention: k_bits=%d v_bits=%d norm_correction=%s",
            k_bits,
            v_bits,
            norm_correction,
        )
        try:
            turboquant_vllm.patch_vllm_attention(
                k_bits=k_bits,
                v_bits=v_bits,
                norm_correction=norm_correction,
            )
        except Exception as exc:  # noqa: BLE001
            log.error("patch_vllm_attention failed: %s", exc)
            sys.exit(3)

    # 3) MoE expert pruning (REAP)
    if reap_fraction is not None and reap_fraction > 0:
        log.info("enabling REAP expert pruning: fraction=%.2f", reap_fraction)
        # REAP requires the model and tokenizer, which vLLM loads internally.
        # The plugin exposes a hook that runs during model load.
        try:
            turboquant_vllm.enable_reap_pruning(prune_fraction=reap_fraction)
        except AttributeError:
            log.warning(
                "enable_reap_pruning not available in this turboquant-plus-vllm "
                "version — skipping MoE pruning"
            )
        except Exception as exc:  # noqa: BLE001
            log.error("enable_reap_pruning failed: %s", exc)
            sys.exit(4)

    log.info("TurboQuant+ patches applied successfully")


def main() -> None:
    apply_patches()

    # Hand off to `vllm serve <args>` using os.execvp so signals (SIGTERM from
    # Docker stop) are delivered directly to vLLM without PID-1 indirection.
    argv = sys.argv[1:]
    if not argv:
        log.error("no arguments passed — expected vllm serve arguments")
        sys.exit(64)

    # If the first arg is already "serve" or starts with "-", assume the caller
    # wants `vllm <args>`. Otherwise prepend "serve".
    if argv[0] == "serve" or argv[0].startswith("-"):
        cmd = ["vllm"] + argv
    else:
        # Common case: `docker run ... image --model X` — the base image's
        # previous entrypoint turned this into `vllm serve --model X`. Keep
        # the same contract.
        cmd = ["vllm", "serve"] + argv

    log.info("exec: %s", " ".join(cmd))
    os.execvp(cmd[0], cmd)


if __name__ == "__main__":
    main()
