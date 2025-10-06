# Ollama‑OpenAI Proxy

A lightweight, production‑ready gateway that exposes a fully‑compatible OpenAI API
surface over a local Ollama server.  
It supports:

- **Full `/v1/chat/completions` endpoint** with streaming (SSE) support.
- **Real‑time model mapping** – use the exact Ollama model names.
- **API key authentication** with bcrypt/argon2 hashing.
- **Circuit breakers** and rate limiting for fault tolerance.
- **Terminal UI** (Bubbletea) for monitoring, key management, and metrics.
- **Zero‑config** – just point to your Ollama instance and start.

Ideal for developers who want OpenAI‑compatible tooling without the cloud
dependency, or for enterprises needing on‑prem LLM inference with a familiar API.
