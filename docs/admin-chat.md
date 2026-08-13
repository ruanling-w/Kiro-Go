# Admin Chat operations

Admin Chat is available at `/admin/chat`. It uses the existing Admin session; the browser never stores or sends a Gateway API key. Every mutating request uses the Admin HttpOnly session plus the `kiro_csrf` double-submit token in `X-CSRF-Token`. The legacy `X-Admin-Password` path is intended for non-browser compatibility only.

## API surface

- `GET|POST /admin/api/chat/conversations`
- `GET|PATCH|DELETE /admin/api/chat/conversations/{id}`
- `GET /admin/api/chat/conversations/{id}/messages`
- `POST /admin/api/chat/conversations/{id}/generate`
- `POST|GET /admin/api/chat/conversations/{id}/attachments`
- `GET|DELETE /admin/api/chat/conversations/{id}/attachments/{attachmentId}`
- `GET /admin/api/chat/conversations/{id}/attachments/{attachmentId}/content`
- `POST /admin/api/chat/conversations/{id}/images/generate`
- `GET /admin/api/chat/models`

Models have provider-qualified identity (`provider:model`). A retry creates a new `clientRequestId`; replaying the same ID with the same request hash is idempotent, while reusing it for different input is rejected.

Text streaming is a POST response with `text/event-stream`. Named events are `generation.created`, `response.delta`, `response.reasoning_summary.delta`, `response.completed`, `response.error`, and `done`. Stop aborts the browser request and persists a terminal stopped state where the provider request observes cancellation.

Stable errors include `csrf_mismatch`, `validation_failed`, `conversation_not_found`, `conversation_archived`, `generation_in_progress`, `concurrency_limit`, `model_capability_mismatch`, `invalid_image`, `invalid_image_output`, `image_generation_failed`, `generation_cancelled`, `provider_error`, and `chat_store_unavailable`.

## Images and storage

Uploads accept at most four PNG, JPEG, or WebP images per message. Each file is limited to 10 MiB and the multipart request is bounded to approximately 41 MiB. MIME is sniffed from bytes. SVG, HTML, GIF, malformed images, dimensions above 16,384, and images above 40 million pixels are rejected.

Binary and base64 image content is never stored in SQLite. SQLite stores metadata and a generated storage key. Asset directories use mode `0700`; files use `0600`. Temporary `.upload` files are atomically renamed only after validation. Path traversal and symlink escape are rejected. Asset responses send `X-Content-Type-Options: nosniff` and private/no-store caching.

Startup reconciliation removes expired unbound metadata, orphan files, and stale upload files. Deleting a conversation removes its validated asset subtree, including orphan files. Back up the SQLite database and chat asset root together to preserve attachment references. Prompts, file bytes, and base64 data must not be logged.

Generated image URLs are fetched only through the SSRF-safe retriever: HTTPS, public resolved addresses, DNS-pinned dialing, TLS hostname preservation, no redirects/proxy, strict timeouts, bounded bodies, and byte-level image validation.

## Capacity and operation

Default concurrent work limits are:

- Text generation: 8
- Attachment processing: 4
- Image generation: 2

Non-stream overload returns HTTP 429 with `concurrency_limit`. A stream already committed as HTTP 200 emits terminal `response.error` and `done`. Permits are released on success, provider failure, and cancellation.

Recommended health checks after deployment:

1. Create, rename, pin, archive, restore, export, and delete a conversation.
2. Stream text, stop it, refresh, and retry with a new request identity.
3. Upload valid images and verify malformed, oversized, fifth-file, and pixel-bomb rejection.
4. Verify multi-turn vision and provider/model capability gating.
5. Generate, stop, retry, and download an image; restart and verify persisted history/assets.
6. Correlate message request IDs and input/output/cache token counts with request logs.

## Rollout and rollback

Before rollout, back up SQLite and the asset root, run the full Go/race/frontend gates, and verify permissions for the runtime user. Roll out one instance first and monitor `concurrency_limit`, storage, provider, CSRF, and reconciliation errors.

Rollback the application binary/frontend together. Preserve SQLite and the asset root; do not restore only one side. If reverting across a schema change, restore the matched pre-rollout database and asset backup.

## Current scope

Implemented scope is Admin Chat text, Markdown/GFM, attachments and multi-turn vision, image generation, history, stop/retry, conversation productivity, export, and persisted gallery. Workspace, Web Search, Agent mode, Compare mode, and Multi-user collaboration are not implemented. Full provider-backed browser generation smoke tests require deterministic provider fixtures; backend integration tests cover generation contracts when such fixtures are unavailable.
