# Phase 11: Audio + Images

**Phase:** 11  
**Goal:** Implement `/v1/audio/*` and `/v1/images/*` with provider support.  
**Requirements:** OPENAI-07..08  
**Estimated duration:** 4–5 days  
**Wave:** 4 — Advanced API Surface

---

## Why

Multimodal APIs are required for full OpenAI compatibility. This phase extends the provider interface to audio and images.

---

## Scope

### In scope
- `internal/providers/openai/audio.go` — speech, transcription.
- `internal/providers/openai/images.go` — generations, edits, variations.
- `internal/providers/gemini/audio.go` — Gemini TTS/STT.
- `internal/api/audio.go` — `/v1/audio/speech`, `/transcriptions`, `/translations`.
- `internal/api/images.go` — `/v1/images/generations`, `/edits`, `/variations`.
- Streaming TTS via SSE.

### Out of scope
- Video generation.
- Advanced image editing pipelines.

---

## Verification

### Tests
1. TTS returns valid audio bytes.
2. Transcription returns valid JSON.
3. Image generation returns URL or b64 data.
4. Streaming TTS emits audio chunks via SSE.
5. Gemini audio converters forward correct parameters.

### Manual verification
1. Run TTS and play resulting audio.
2. Run image generation and view image.

---

## Tasks

1. Extend provider interface for audio and images.
2. Implement OpenAI audio/image converters.
3. Implement Gemini audio converter.
4. Implement API handlers with multipart support where needed.
5. Write fixture tests.
6. Verify gates.

---

## Risks

| Risk | Mitigation |
|------|------------|
| Multipart form parsing is tricky | Use explicit boundary parsing tests. |
| Audio codec mismatches | Test with common formats (MP3, WAV). |
