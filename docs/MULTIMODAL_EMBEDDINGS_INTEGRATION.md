# Multimodal Embeddings Integration Guide

## 🎯 Цель

Интеграция **nvidia/omni-embed-nemotron-3b** для мультимодальных embeddings (text + image + audio + video) с AIGateway Platform.

---

## ⚠️ Важно: Почему нельзя использовать Ollama напрямую?

1. **Ollama embeddings API** поддерживает только **text-only** модели
2. **llama.cpp** не поддерживает Qwen2.5-Omni архитектуру
3. **Multimodal processing** требует специальный AutoProcessor
4. **Audio/Video** не поддерживаются в llama.cpp

---

## 🏗️ Архитектура решения

### Option 1: Python Microservice (Рекомендуется)

```
┌─────────────────────────────────────────┐
│ AIGateway Platform (Go)                │
│ Port: 8080                              │
│                                         │
│ /v1/embeddings                          │
│   ├─ text-only → Ollama                 │
│   └─ multimodal → Python Service        │
└─────────────────────────────────────────┘
              ↓
┌─────────────────────────────────────────┐
│ Python Embedding Service                │
│ Port: 8000                              │
│                                         │
│ Framework: FastAPI                      │
│ Model: nvidia/omni-embed-nemotron-3b    │
└─────────────────────────────────────────┘
```

---

## 📦 Установка Python Service

### Шаг 1: Создайте Python окружение

```bash
mkdir multimodal-embeddings-service
cd multimodal-embeddings-service

python -m venv venv
source venv/bin/activate  # Windows: venv\Scripts\activate

pip install torch torchvision torchaudio --index-url https://download.pytorch.org/whl/cu121
pip install transformers==4.51.3
pip install fastapi uvicorn pillow librosa
```

### Шаг 2: Создайте сервис

```python
# service.py
from fastapi import FastAPI, File, UploadFile, Form
from fastapi.responses import JSONResponse
from transformers import AutoModel, AutoProcessor
import torch
import torch.nn.functional as F
from PIL import Image
import io
import librosa
import numpy as np
from typing import Optional

app = FastAPI()

# Load model at startup
model_name = "nvidia/omni-embed-nemotron-3b"
print(f"Loading model: {model_name}")

model = AutoModel.from_pretrained(
    model_name,
    torch_dtype=torch.bfloat16,
    attn_implementation="flash_attention_2",
    trust_remote_code=True,
)
model = model.to("cuda:0" if torch.cuda.is_available() else "cpu")
model.eval()

processor = AutoProcessor.from_pretrained(model_name, trust_remote_code=True)
print("Model loaded successfully!")


@app.get("/health")
async def health():
    return {"status": "healthy", "model": model_name}


@app.post("/v1/embeddings")
async def create_embeddings(
    text: Optional[str] = Form(None),
    image: Optional[UploadFile] = File(None),
    audio: Optional[UploadFile] = File(None),
    video: Optional[UploadFile] = File(None),
):
    """
    Create embeddings for multimodal inputs.
    
    OpenAI-compatible endpoint.
    """
    try:
        # Prepare content
        content = []
        
        if text:
            content.append({"type": "text", "text": f"passage: {text}"})
        
        if image:
            img_bytes = await image.read()
            img = Image.open(io.BytesIO(img_bytes))
            content.append({"type": "image", "image": img})
        
        if audio:
            audio_bytes = await audio.read()
            # Convert to waveform
            waveform, sr = librosa.load(io.BytesIO(audio_bytes), sr=16000)
            content.append({"type": "audio", "audio": waveform})
        
        if video:
            video_bytes = await video.read()
            # Save temp file for video processing
            temp_path = "/tmp/temp_video.mp4"
            with open(temp_path, "wb") as f:
                f.write(video_bytes)
            content.append({"type": "video", "video": temp_path})
        
        if not content:
            return JSONResponse(
                status_code=400,
                content={"error": "At least one input (text, image, audio, or video) is required"}
            )
        
        # Prepare input
        documents = [{"role": "user", "content": content}]
        
        # Process
        documents_texts = processor.apply_chat_template(
            documents, add_generation_prompt=False, tokenize=False
        )
        
        # Extract modalities
        from qwen_omni_utils import process_mm_info
        audio_data, images_data, videos_data = process_mm_info(documents, use_audio_in_video=False)
        
        # Batch process
        batch_dict = processor(
            text=documents_texts,
            images=images_data,
            videos=videos_data,
            audio=audio_data,
            return_tensors="pt",
            text_kwargs={"truncation": True, "padding": True, "max_length": 204800},
            videos_kwargs={"min_pixels": 32*14*14, "max_pixels": 64*28*28, "use_audio_in_video": False},
            audio_kwargs={"max_length": 2048000},
        )
        
        batch_dict = {k: v.to(model.device) for k, v in batch_dict.items()}
        
        # Get embeddings
        with torch.no_grad():
            last_hidden_states = model(**batch_dict, output_hidden_states=True).hidden_states[-1]
            
            # Average Pooling
            attention_mask = batch_dict["attention_mask"]
            last_hidden_states_masked = last_hidden_states.masked_fill(
                ~attention_mask[..., None].bool(), 0.0
            )
            embedding = last_hidden_states_masked.sum(dim=1) / attention_mask.sum(dim=1)[..., None]
            embedding = F.normalize(embedding, dim=-1)
        
        # Convert to OpenAI format
        embedding_list = embedding[0].cpu().tolist()
        
        response = {
            "object": "list",
            "data": [
                {
                    "object": "embedding",
                    "index": 0,
                    "embedding": embedding_list
                }
            ],
            "model": model_name,
            "usage": {
                "prompt_tokens": batch_dict["input_ids"].shape[1],
                "total_tokens": batch_dict["input_ids"].shape[1]
            }
        }
        
        return response
    
    except Exception as e:
        return JSONResponse(
            status_code=500,
            content={"error": str(e)}
        )


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
```

### Шаг 3: Запустите сервис

```bash
python service.py
```

---

## 🔧 Интеграция с Go Proxy

### Добавьте multimodal handler

```go
// internal/api/handlers/multimodal_embeddings.go
package handlers

import (
    "bytes"
    "encoding/json"
    "io"
    "mime/multipart"
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/sirupsen/logrus"
    
    "aigateway/internal/config"
    "aigateway/internal/models"
)

type MultimodalEmbeddingsHandler struct {
    config       *config.Config
    logger       *logrus.Logger
    pythonClient *http.Client
}

func NewMultimodalEmbeddingsHandler(
    cfg *config.Config,
    logger *logrus.Logger,
) *MultimodalEmbeddingsHandler {
    return &MultimodalEmbeddingsHandler{
        config: cfg,
        logger: logger,
        pythonClient: &http.Client{
            Timeout: 60 * time.Second, // Multimodal может быть медленнее
        },
    }
}

// HandleMultimodalEmbeddings обрабатывает /v1/embeddings/multimodal
func (h *MultimodalEmbeddingsHandler) HandleMultimodalEmbeddings(c *gin.Context) {
    // Parse multipart form
    if err := c.Request.ParseMultipartForm(32 << 20); err != nil { // 32MB max
        h.logger.WithError(err).Error("Failed to parse multipart form")
        c.JSON(http.StatusBadRequest, models.ErrorResponse{
            Error: models.Error{
                Message: "Invalid multipart form",
                Type:    "invalid_request_error",
                Code:    "invalid_form",
            },
        })
        return
    }

    // Prepare request to Python service
    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)

    // Add text if present
    if text := c.PostForm("text"); text != "" {
        writer.WriteField("text", text)
    }

    // Add image if present
    if imageFile, _, err := c.Request.FormFile("image"); err == nil {
        defer imageFile.Close()
        
        part, _ := writer.CreateFormFile("image", "image.jpg")
        io.Copy(part, imageFile)
    }

    // Add audio if present
    if audioFile, _, err := c.Request.FormFile("audio"); err == nil {
        defer audioFile.Close()
        
        part, _ := writer.CreateFormFile("audio", "audio.wav")
        io.Copy(part, audioFile)
    }

    // Add video if present
    if videoFile, _, err := c.Request.FormFile("video"); err == nil {
        defer videoFile.Close()
        
        part, _ := writer.CreateFormFile("video", "video.mp4")
        io.Copy(part, videoFile)
    }

    writer.Close()

    // Forward to Python service
    pythonServiceURL := h.config.MultimodalEmbeddings.PythonServiceURL
    if pythonServiceURL == "" {
        pythonServiceURL = "http://localhost:8000/v1/embeddings"
    }

    req, err := http.NewRequest("POST", pythonServiceURL, body)
    if err != nil {
        h.logger.WithError(err).Error("Failed to create request to Python service")
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{
            Error: models.Error{
                Message: "Internal server error",
                Type:    "api_error",
                Code:    "internal_error",
            },
        })
        return
    }

    req.Header.Set("Content-Type", writer.FormDataContentType())

    // Execute request
    resp, err := h.pythonClient.Do(req)
    if err != nil {
        h.logger.WithError(err).Error("Python service request failed")
        c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
            Error: models.Error{
                Message: "Embedding service unavailable",
                Type:    "api_error",
                Code:    "service_unavailable",
            },
        })
        return
    }
    defer resp.Body.Close()

    // Read response
    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        h.logger.WithError(err).Error("Failed to read Python service response")
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{
            Error: models.Error{
                Message: "Failed to process response",
                Type:    "api_error",
                Code:    "response_error",
            },
        })
        return
    }

    // Return response
    var embeddingResp models.EmbeddingResponse
    if err := json.Unmarshal(respBody, &embeddingResp); err != nil {
        h.logger.WithError(err).Error("Failed to parse embedding response")
        c.JSON(http.StatusInternalServerError, models.ErrorResponse{
            Error: models.Error{
                Message: "Invalid response format",
                Type:    "api_error",
                Code:    "parse_error",
            },
        })
        return
    }

    h.logger.WithFields(logrus.Fields{
        "embeddings": len(embeddingResp.Data),
        "dimensions": len(embeddingResp.Data[0].Embedding),
    }).Info("Multimodal embeddings generated successfully")

    c.JSON(http.StatusOK, embeddingResp)
}
```

### Добавьте конфигурацию

```yaml
# configs/dev.yaml
multimodal_embeddings:
  enabled: true
  python_service_url: "http://localhost:8000/v1/embeddings"
  timeout: 60s
```

### Добавьте роут

```go
// internal/api/router/router.go
if r.config.MultimodalEmbeddings.Enabled {
    v1.POST("/embeddings/multimodal", r.multimodalEmbeddingsHandler.HandleMultimodalEmbeddings)
}
```

---

## 🧪 Тестирование

### Text + Image embeddings

```bash
curl -X POST http://localhost:8080/v1/embeddings/multimodal \
  -H "Authorization: Bearer YOUR_KEY" \
  -F "text=A beautiful sunset over the ocean" \
  -F "image=@sunset.jpg"
```

### Text + Audio embeddings

```bash
curl -X POST http://localhost:8080/v1/embeddings/multimodal \
  -H "Authorization: Bearer YOUR_KEY" \
  -F "text=Music description" \
  -F "audio=@music.mp3"
```

### All modalities

```bash
curl -X POST http://localhost:8080/v1/embeddings/multimodal \
  -H "Authorization: Bearer YOUR_KEY" \
  -F "text=Description" \
  -F "image=@image.jpg" \
  -F "audio=@audio.wav" \
  -F "video=@video.mp4"
```

---

## 📊 Performance

**Ожидаемая latency:**
- Text-only: ~50ms
- Text + Image: ~200ms
- Text + Audio: ~300ms
- Text + Video: ~1-2s (зависит от длины)
- All modalities: ~2-3s

**GPU Requirements:**
- Минимум: RTX 3090 (24GB)
- Рекомендуется: A100 (40GB) или H100 (80GB)

---

## 🚀 Production Deployment

### Docker Compose

```yaml
# docker-compose.multimodal.yml
version: '3.8'

services:
  ollama-proxy:
    build: .
    ports:
      - "8080:8080"
    environment:
      - MULTIMODAL_EMBEDDINGS_ENABLED=true
      - MULTIMODAL_EMBEDDINGS_PYTHON_SERVICE_URL=http://python-embeddings:8000/v1/embeddings
    depends_on:
      - python-embeddings
  
  python-embeddings:
    build: ./multimodal-embeddings-service
    ports:
      - "8000:8000"
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [gpu]
```

---

## ⚠️ Важные замечания

1. **GPU Memory:** Модель требует ~10GB VRAM
2. **Latency:** Multimodal медленнее чем text-only (2-3s vs 50ms)
3. **Dimensions:** 2048 (не совместимо с 768/1024 моделями!)
4. **Batching:** Поддерживается, но требует больше памяти

---

## 📚 Альтернативы

Если Omni-Embed слишком тяжелая:

1. **CLIP** - text + image (OpenAI)
2. **ImageBind** - text + image + audio (Meta)
3. **BridgeTower** - text + image (Intel)

Все они **легче конвертируются** в GGUF для Ollama.

---

**Рекомендация:** Используйте Python microservice подход для максимальной гибкости!


