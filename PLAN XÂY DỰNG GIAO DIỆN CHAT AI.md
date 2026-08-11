# PLAN XÂY DỰNG GIAO DIỆN CHAT AI CHO KIRO-GO

## 1. Bối cảnh và mục tiêu

Kiro-Go đã có sẵn Go `net/http`, SQLite, React 19 + Vite + TypeScript + Tailwind/shadcn, React Query/Zustand, Admin session/CSRF, SSE, model routing/failover/Combo, multimodal translator và image generation. Vì vậy không đổi sang Next.js, Gin/Fiber, PostgreSQL, Redis hoặc S3 ở giai đoạn đầu.

Mục tiêu trước mắt là bổ sung một giao diện ChatGPT-like trong Admin, dùng được thực tế:

- Chat text streaming với nhiều provider/model và Combo.
- Lịch sử conversation lưu SQLite.
- Markdown, bảng và code block có highlight/copy.
- Stop, retry, regenerate và trạng thái lỗi rõ ràng.
- Upload/paste/drag-drop ảnh để gửi model vision.
- Tạo ảnh bằng router Codex/Grok/Antigravity hiện có.
- Responsive desktop/tablet/mobile và dark mode.
- Bảo mật, usage và observability đồng bộ với Admin hiện tại.

Tầm nhìn dài hạn vẫn là AI Workspace gồm Projects, Memory, Web Search, Tools, Agent, Code Workspace, Artifacts, Compare Models và Auto Router, nhưng triển khai theo version nhỏ để mỗi mốc đều chạy và kiểm thử được.

---

## 2. Kiến trúc được chọn

```text
/admin/chat (React/Vite)
       │
       │ Admin session + CSRF
       ▼
/admin/api/chat/* (Go net/http)
       │
       ├── SQLite: conversations/messages/attachment metadata
       ├── Runtime filesystem: image/file binary
       └── Internal gateway executor
              ├── OpenAI Chat / Responses
              ├── Combo / Fusion
              ├── Account pool + failover
              ├── Multimodal translators
              └── Codex/Grok/Antigravity image generation
```

Nguyên tắc:

- Browser không giữ và không gửi Gateway API key.
- Admin Chat gọi pipeline nội bộ dùng chung, không HTTP loopback tới `/v1`.
- Không xây provider router thứ hai.
- Không dùng Responses API store làm UI history vì lifecycle/TTL khác nhau.
- Binary/base64 không lưu trong SQLite.
- Chat dùng SQLite trong mô hình single-admin/single-instance hiện tại.
- Mỗi increment là một vertical slice DB → API → UI → test và rollback độc lập.

---

# 3. DELIVERABLE ĐẦU TIÊN — CHAT + IMAGES

Deliverable đầu tiên hoàn thành khi có:

1. Route `/admin/chat` và navigation trong Admin.
2. Conversation sidebar, New Chat, search title cơ bản và lịch sử theo ngày.
3. Conversation CRUD, pin/archive/rename/delete tối thiểu.
4. Model selector giữ đúng composite identity `provider + model`.
5. Chat text non-streaming trước, sau đó SSE streaming.
6. Stop bằng AbortController, retry không duplicate, partial response được giữ.
7. Markdown GFM an toàn và code block highlight/copy.
8. Upload/paste/drag-drop ảnh, preview và gửi vision.
9. Create Image, gallery, download và retry.
10. Lịch sử tồn tại sau browser refresh và server restart.
11. Session/CSRF, body/file limits, MIME/path/XSS protections và tests.
12. Usage liên kết request logs hiện tại, không tạo token ledger thứ hai.

Feature chưa hoàn chỉnh phải ẩn khỏi navigation bằng flag safe-default; chỉ bật mặc định khi Chat + Images đạt Definition of Done.

---

# 4. PHASE 0 — CONTRACT VÀ TEST HARNESS

Trước khi làm UI lớn:

- Chốt DTO conversation/message/attachment/model.
- Chốt SSE event names, terminal semantics và error codes.
- Chốt safe defaults cho JSON body, ảnh, concurrency và timeout.
- Tách interface giữa Admin Chat service và gateway executor.
- Thêm test helper cho Admin session + CSRF.
- Chốt capability mapping từ model catalog hiện có; không suy đoán Web/Tools nếu backend chưa hỗ trợ.

Acceptance:

- Contract tests chạy được.
- Không cần API key trong browser.
- Provider/model identity không bị mất ở bất kỳ tầng nào.

---

# 5. PHASE 1 — SQLITE CHAT PERSISTENCE (SCHEMA V8)

Nâng `store.schemaVersion` từ 7 lên 8, migration transaction/idempotent theo pattern reconciliation trong `store/sqlite.go`.

## 5.1 `chat_conversations`

```sql
CREATE TABLE chat_conversations (
  id          TEXT PRIMARY KEY,
  title       TEXT NOT NULL DEFAULT '',
  provider    TEXT NOT NULL DEFAULT '',
  model       TEXT NOT NULL DEFAULT '',
  mode        TEXT NOT NULL DEFAULT 'chat',
  status      TEXT NOT NULL DEFAULT 'active',
  pinned      INTEGER NOT NULL DEFAULT 0,
  project_id  TEXT,
  created_at  INTEGER NOT NULL,
  updated_at  INTEGER NOT NULL,
  archived_at INTEGER
);
```

Index phục vụ cursor list theo `status`, `pinned`, `updated_at`, `id`.

Chưa thêm `user_id` giả vì hiện chỉ có một Admin principal. Multi-user sẽ có migration owner riêng.

## 5.2 `chat_messages`

```sql
CREATE TABLE chat_messages (
  id                    TEXT PRIMARY KEY,
  conversation_id       TEXT NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
  parent_message_id     TEXT,
  client_request_id     TEXT NOT NULL DEFAULT '',
  role                  TEXT NOT NULL,
  content               TEXT NOT NULL DEFAULT '',
  provider              TEXT NOT NULL DEFAULT '',
  model                 TEXT NOT NULL DEFAULT '',
  status                TEXT NOT NULL DEFAULT 'complete',
  error_code            TEXT NOT NULL DEFAULT '',
  error_message         TEXT NOT NULL DEFAULT '',
  provider_response_id  TEXT NOT NULL DEFAULT '',
  request_id            TEXT NOT NULL DEFAULT '',
  input_tokens          INTEGER NOT NULL DEFAULT 0,
  output_tokens         INTEGER NOT NULL DEFAULT 0,
  cache_read_tokens     INTEGER NOT NULL DEFAULT 0,
  cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
  created_at            INTEGER NOT NULL,
  updated_at            INTEGER NOT NULL
);
```

Status: `pending | streaming | complete | stopped | error`.

Không update SQLite theo từng token. Tạo user message + assistant placeholder trong transaction, rồi finalize assistant một lần khi complete/stopped/error.

## 5.3 `chat_attachments`

```sql
CREATE TABLE chat_attachments (
  id              TEXT PRIMARY KEY,
  conversation_id TEXT NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
  message_id      TEXT REFERENCES chat_messages(id) ON DELETE CASCADE,
  kind            TEXT NOT NULL,
  name            TEXT NOT NULL DEFAULT '',
  mime_type       TEXT NOT NULL,
  size_bytes      INTEGER NOT NULL,
  storage_key     TEXT NOT NULL,
  width           INTEGER,
  height          INTEGER,
  created_at      INTEGER NOT NULL
);
```

Kind: `image_input | image_output | file`.

## 5.4 Store API

Tạo store module chuyên biệt:

- Create/List/Get/Update/Delete conversation.
- Cursor pagination và search title/preview.
- Create user turn + assistant placeholder trong transaction.
- Finalize assistant message.
- List message pages.
- Create/Get/Delete attachment metadata.
- Cascade delete và asset cleanup coordination.

Nếu SQLite không sẵn sàng, Chat trả `503 chat_store_unavailable`; không giả vờ lưu trong RAM.

Files chính:

- `store/sqlite.go`
- store chat mới
- focused migration/store tests

---

# 6. PHASE 2 — ADMIN CHAT API

Mở rộng `handleAdminAPI()` trong `proxy/handler.go`, nhưng route parsing, DTO và validation nằm trong file Admin Chat riêng để tránh làm handler tiếp tục phình lớn.

## 6.1 Conversation API

```text
GET    /admin/api/chat/conversations?cursor=&limit=&q=&status=
POST   /admin/api/chat/conversations
GET    /admin/api/chat/conversations/{id}
PATCH  /admin/api/chat/conversations/{id}
DELETE /admin/api/chat/conversations/{id}
GET    /admin/api/chat/conversations/{id}/messages?cursor=&limit=
```

- Cursor dùng `(updated_at,id)`, không dùng offset cho danh sách dài.
- Phase đầu search title/preview; search message/FTS ở version sau.
- Delete phải xử lý DB metadata và asset cleanup.

## 6.2 Model catalog cho Chat

```text
GET /admin/api/chat/models
```

Model DTO:

```json
{
  "id": "provider:model",
  "provider": "provider-id",
  "model": "model-id",
  "displayName": "Model name",
  "availability": "available",
  "capabilities": {
    "vision": true,
    "imageGeneration": false,
    "reasoning": true,
    "tools": false,
    "web": false
  }
}
```

Yêu cầu:

- Giữ composite identity provider + model end-to-end.
- Combo vẫn là model chọn được.
- Availability ban đầu: `available | unavailable | unknown`.
- Không quảng cáo Web/Tools nếu backend chưa có contract thật.
- Không cho gửi tới model unavailable.

## 6.3 Auth và validation

- Mọi endpoint đi qua Admin session hiện có.
- POST/PATCH/DELETE và POST SSE dùng CSRF hiện có.
- JSON body có `http.MaxBytesReader` và strict validation.
- API trả error code ổn định, không chỉ text tự do.

Files/patterns:

- `proxy/handler.go`
- `proxy/admin_auth.go`
- `proxy/model_catalog_cache.go`
- `proxy/combo_capabilities.go`
- Admin Chat files mới

---

# 7. PHASE 3 — CHAT TEXT END-TO-END

## 7.1 Slice A — Non-streaming trước

- Tách internal chat executor từ OpenAI/Responses pipeline hiện có.
- Tái sử dụng account pool, provider-qualified routing, Combo/Fusion, failover, cache, usage và request logs.
- Không gọi loopback `/v1` và không nhân bản provider adapter.
- Persist user message + assistant placeholder atomically.
- Finalize assistant và update conversation timestamp/title sau response.
- Auto-title lần đầu bằng prompt truncate an toàn; AI-generated title để sau.
- `clientRequestId` làm idempotency key để retry mạng không tạo duplicate turn.

Acceptance:

- Tạo chat, gửi text, nhận answer và refresh vẫn thấy lịch sử.
- Provider/model thực tế được lưu.
- Error không làm mất user prompt.

## 7.2 Slice B — POST SSE streaming

```text
POST /admin/api/chat/conversations/{id}/generate
Accept: text/event-stream
```

Request:

```json
{
  "clientRequestId": "uuid",
  "provider": "provider-id",
  "model": "model-id",
  "content": "Nội dung",
  "attachmentIds": [],
  "reasoning": "balanced"
}
```

Frontend dùng `fetch` + `ReadableStream`; không dùng EventSource vì request cần POST body và CSRF.

SSE contract:

```text
event: generation.created
data: {"generationId":"...","userMessageId":"...","assistantMessageId":"..."}

event: response.delta
data: {"delta":"..."}

event: response.reasoning_summary.delta
data: {"delta":"..."}

event: response.completed
data: {"finishReason":"stop","provider":"...","model":"...","usage":{...}}

event: response.error
data: {"code":"model_unavailable","message":"...","retryable":true}

event: done
data: {}
```

- Chỉ gửi reasoning summary được upstream cho phép, không chain-of-thought nội bộ.
- Luôn có đúng một terminal outcome.
- Stop dùng AbortController; Go request context phải hủy upstream.
- Khi abort/disconnect, lưu partial response với status `stopped`.
- Chỉ thêm cancel endpoint riêng khi generation được tách khỏi HTTP request trong phase Agent/background.

Tái sử dụng:

- `proxy/openai_sse_writer.go`
- `proxy/responses_handler.go`
- Combo stream handlers
- account failover
- token/request-log helpers

---

# 8. PHASE 4 — FRONTEND CHATGPT-LIKE

## 8.1 Routing và layout

- Thêm lazy route `/admin/chat`.
- Thêm Chat vào `NAV_ITEMS`.
- Giữ `AuthGuard` và `AppShell`.
- Route Chat dùng full-height content, không bị padding/scroll kép.
- Desktop: conversation sidebar + chat.
- Tablet/mobile: history sidebar chuyển thành `Sheet`, transcript full width, composer sticky và safe-area aware.

## 8.2 Feature structure

```text
features/chat/
  ChatPage.tsx
  components/
    ChatSidebar.tsx
    ConversationList.tsx
    ConversationView.tsx
    MessageList.tsx
    MessageBubble.tsx
    MarkdownMessage.tsx
    ChatComposer.tsx
    ModelSelector.tsx
    AttachmentTray.tsx
    ImageGenerationPanel.tsx
    GenerationError.tsx
  hooks/
    useChatStream.ts
    useConversationActions.ts
    useImagePaste.ts
  services/
    chat.service.ts
    chatStream.ts
  chatUIStore.ts
  types.ts
```

## 8.3 State ownership

React Query quản lý:

- Conversation list/detail.
- Message pages.
- Model catalog.
- Attachment metadata.

Zustand/local reducer chỉ quản lý:

- Draft theo conversation.
- Selected provider/model và mode.
- Active generation/AbortController.
- Pending attachment.
- Mobile drawer.

Không duplicate toàn bộ message history trong Zustand.

## 8.4 Composer và message UX

- Textarea auto-grow.
- Enter gửi; Shift+Enter xuống dòng.
- Send/Stop.
- Upload/paste/drag-drop.
- Model/reasoning/chat-image mode controls.
- Optimistic user message + assistant placeholder.
- Complete reconcile server IDs; error giữ prompt + Retry; abort giữ partial.
- Auto-scroll chỉ khi user đang gần cuối transcript.
- Ctrl/Cmd+K search; Ctrl/Cmd+Shift+O New Chat; Esc Stop.

JSON API dùng Admin `httpClient`; stream dùng fetch helper riêng đọc CSRF cookie và normalize 401/403 giống axios client.

Files chính:

- `web/frontend/src/router.tsx`
- `web/frontend/src/config/nav.ts`
- `web/frontend/src/components/layout/AppShell.tsx`
- `web/frontend/src/services/httpClient.ts`
- `web/frontend/src/config/queryKeys.ts`
- feature Chat mới
- locale trong `web/frontend/src/lib/i18n.ts`

---

# 9. PHASE 5 — MARKDOWN VÀ CODE

Bổ sung:

- `react-markdown`
- `remark-gfm`
- `rehype-sanitize`
- syntax highlighter lazy-load, ưu tiên Shiki nếu bundle phù hợp

Yêu cầu:

- Heading, bold, italic, list, table, quote, link, inline/fenced code.
- Không bật raw HTML trong MVP.
- Sanitize URL scheme; chặn `javascript:` và data URI không tin cậy.
- External link dùng `target=_blank` và `noopener noreferrer`.
- Mở rộng shared `CodeBlock.tsx` để giữ language label/copy hiện có và thêm syntax highlighting.
- Throttle/batch streaming render khoảng 30–50ms thay vì set state mỗi token.
- Chưa có Run, Apply Patch hoặc Artifact iframe trong deliverable đầu.

---

# 10. PHASE 6 — MULTIMODAL IMAGE INPUT

## 10.1 Upload API

```text
POST /admin/api/chat/conversations/{id}/attachments
Content-Type: multipart/form-data
```

Content/download endpoint cũng phải Admin-authenticated.

Storage:

```text
{runtime-dir}/chat-assets/{conversation-id}/{server-generated-id}
```

- Directory/file permission 0700/0600.
- `storage_key` tương đối, server-generated.
- Filename người dùng chỉ là metadata.
- Không lưu base64 trong SQLite.

## 10.2 Safe defaults

- Tối đa 4 ảnh/message.
- Tối đa 10 MiB/ảnh và giới hạn tổng request.
- Allowlist PNG/JPEG/WebP.
- GIF chỉ bật khi decode policy xác nhận hỗ trợ.
- Reject SVG/HTML.
- Dùng `http.MaxBytesReader`.
- Sniff MIME từ bytes, không tin extension/header.
- Decode dimensions và giới hạn pixel để chống decompression bomb.
- Chặn path traversal.
- Content response có `nosniff`, private/no-store.

## 10.3 Frontend UX

- Click upload.
- Drag/drop.
- Paste image/file.
- Nhiều preview.
- Remove-before-send.
- Progress và lỗi từng file.

Backend đọc attachment đã xác thực và dựng content parts theo shape mà `responses_input.go` và translator hiện có hiểu. Không gửi local URL cho upstream provider.

Delete conversation/message phải cleanup asset; thêm reconciliation cho orphan file sau crash. Không log prompt/base64/file content.

---

# 11. PHASE 7 — IMAGE GENERATION

```text
POST /admin/api/chat/conversations/{id}/images/generate
```

Request gồm provider/model identity, prompt và options model thật sự hỗ trợ.

Implementation:

- Refactor `handleImageGenerations()` thành reusable executor.
- Tái sử dụng Codex/Grok/Antigravity adapters, classifier, account pool và failover.
- Không HTTP loopback.
- Validate base64/MIME output.
- Lưu ảnh thành `image_output` attachment.
- Tạo assistant message liên kết asset.

UI:

- Chat/Create Image mode.
- Generating state và cancellation theo request context.
- Gallery card.
- Download.
- Copy prompt.
- Retry.

Edit/upscale/variation để phase sau khi provider contract hỗ trợ thật.

---

# 12. VERSION 0.2 — CONVERSATION PRODUCTIVITY VÀ FILE TEXT

- Rename, archive, pin, delete confirmation, duplicate.
- Export Markdown/JSON.
- Search title + message content; chỉ thêm SQLite FTS5 sau khi đo dữ liệu.
- Edit prompt/regenerate tạo branch bằng `parent_message_id`, không overwrite lịch sử cũ.
- TXT/Markdown/CSV/source upload có charset normalization, extraction/token budget và provenance.
- PDF/Word/Excel cần extractor giới hạn tài nguyên.
- ZIP chỉ bật sau malware scanning, archive-bomb protection và sandbox.

---

# 13. VERSION 0.3 — WEB SEARCH, PROJECTS, MEMORY, PROMPTS

## Web Search

Không chỉ thêm toggle UI. Cần backend search/tool contract và structured citations:

```text
search.started
search.result
search.completed
citation
```

- SSRF protection: chặn loopback/private IP, redirect bất thường, response quá lớn.
- Timeout/quota/attribution rõ ràng.
- UI render source từ dữ liệu cấu trúc, không parse URL từ markdown model.

## Projects

Project gom:

- Conversations.
- Files.
- Instructions.
- Knowledge references.

Project ở phase này là data scope, chưa phải agent sandbox.

## Memory

- Scope `global | project`.
- Admin xem, sửa, xóa, disable.
- Có injection budget.
- AI chỉ đề xuất memory; Admin xác nhận trước khi lưu.

## Prompt Library

- Slash command.
- Tags.
- Version nhẹ.
- Prompt templates theo user/project.

---

# 14. VERSION 0.4 — TOOLS VÀ AGENT MODE

Tool registry server-side có:

- Versioned input/output schema.
- Permission.
- Timeout.
- Input/output size limit.
- Audit log.
- Cancellation.

Tool có side effect phải có confirmation gate.

Agent loop có:

- Max steps.
- Max duration.
- Token/cost budget.
- Cancellation.
- Recovery/error state.

UI chỉ hiển thị action/status/result tóm tắt, không hiển thị chain-of-thought.

Code execution, terminal và browser bắt buộc chạy trong sandbox riêng, tuyệt đối không chạy trực tiếp trong process Kiro-Go.

---

# 15. VERSION 0.5 — CODE WORKSPACE VÀ ARTIFACTS

- Monaco Editor lazy-loaded.
- File tree.
- Diff.
- Patch preview → explicit approval → apply.
- Undo.
- Run/terminal/preview trong container sandbox.
- Workspace root do server cấp; chặn traversal và symlink escape.
- Artifact HTML/React dùng sandboxed iframe, CSP nghiêm ngặt và origin cách ly nếu có thể.

---

# 16. VERSION 0.6 — COMPARE MODEL VÀ AUTO ROUTER

## Compare Model

- Gửi một prompt tới 2–4 provider/model candidates.
- Stream event luôn có candidate identity.
- Cancel từng candidate hoặc tất cả.
- Persist compare run và từng answer.
- Vote, copy, continue từ answer được chọn.
- Merge chỉ khi có judge semantics và cost visibility rõ.

Tận dụng Combo/Fusion hiện có nhưng tạo UI contract riêng.

## Auto Router

Policy theo thứ tự có thể giải thích:

```text
Capability → Availability → Health → Latency/Cost
```

UI luôn hiển thị provider/model thực tế đã xử lý request.

---

# 17. VERSION 0.7–1.0 — VOICE, USAGE, MULTI-USER, SCALE

## Voice

- Speech-to-Text và Text-to-Speech qua provider contract thật.
- Consent và microphone permission rõ ràng.

## Usage

- Join chat generation với request logs bằng `request_id`.
- Hiển thị input/output/cache/cost theo semantics metrics hiện có.
- Không tạo token/cost ledger thứ hai.

## Scale gates

Chỉ chuyển PostgreSQL/object storage/worker queue hoặc split services khi xuất hiện một hoặc nhiều tín hiệu:

- Multi-user/tenant.
- Nhiều Go instances.
- Attachment lớn hoặc dung lượng cao.
- Background agent dài hạn.
- Collaboration.
- High availability.
- SQLite write contention được đo.

---

# 18. SECURITY VÀ LIFECYCLE

Bắt buộc xuyên suốt:

- HTTPS ở deployment edge.
- Same-origin Admin session.
- CSRF cho mọi mutation, kể cả streaming POST.
- Không lộ API key hoặc Admin password xuống browser.
- JSON/multipart body limits.
- Method/content-type validation.
- Context cancellation và upstream timeout.
- Parameterized SQL và FK/cascade.
- Storage path containment.
- MIME/pixel/file/count quotas.
- Markdown sanitize; không raw HTML/SVG/data URI tùy ý.
- Không log conversation/file content theo mặc định.
- Retention, export và delete rõ ràng.
- DB + asset cleanup và orphan reconciliation.
- Share link ở phase sau dùng opaque revocable token có expiry.
- Rate/concurrency limit riêng cho stream, upload và image generation.

---

# 19. ERROR CONTRACT

Các error code tối thiểu:

```text
chat_store_unavailable
conversation_not_found
model_unavailable
model_capability_mismatch
invalid_attachment
attachment_too_large
unsupported_media_type
generation_timeout
generation_cancelled
upstream_rate_limited
upstream_unavailable
stream_interrupted
```

UI giữ prompt và partial response khi có thể, hiển thị Retry nếu lỗi retryable và không biến lỗi giữa stream thành answer kết thúc bình thường.

---

# 20. PERFORMANCE

- SSE cho chat đơn giản; chưa cần WebSocket.
- React Query pagination.
- Lazy-load Chat page, Markdown highlighter và Monaco.
- Batch stream delta khoảng 30–50ms.
- Auto-scroll có điều kiện.
- Virtualized message list chỉ thêm sau khi đo conversation dài; tránh complexity sớm.
- SQLite single writer: không ghi mỗi token.
- Asset binary nằm ngoài DB.
- Concurrency budget tránh một tab chiếm toàn bộ account pool.

---

# 21. TEST PLAN

## Backend/store

- DB mới và migration v7 → v8.
- Migration chạy lại idempotent.
- FK/cascade/index/cursor/search.
- CRUD conversation/message/attachment.
- Transaction placeholder/finalize.
- Idempotent retry.
- Complete/stopped/error persistence.
- Nil-store trả 503.
- Admin auth và CSRF đúng/sai/hết hạn.
- Method/body/content-type validation.
- SSE event ordering và đúng một terminal event.
- Malformed/midstream errors.
- Client disconnect hủy upstream và persist partial.
- Provider/model identity và Combo routing.
- MIME spoof, oversize, pixel bomb, traversal, unauthorized content.
- Asset cleanup/orphan reconciliation.
- Vision translator fixtures.
- Image routing/provider failure/invalid base64/cancel.
- Usage/request ID khớp request logs và không double count.

## Frontend

Bổ sung Vitest + React Testing Library, mocked fetch/SSE và Playwright smoke.

- Create/open/rename/archive/delete conversation.
- Hydrate sau refresh.
- SSE parser chịu network chunk cắt ở mọi ranh giới.
- Malformed SSE, 401/403, abort/retry.
- Markdown XSS và unsafe URL.
- Code copy/highlight.
- Long content và auto-scroll.
- Model availability/capability gating.
- Paste/drop/upload/remove ảnh.
- Partial upload failure.
- Vision send.
- Image generation success/failure/download/retry.
- Desktop/tablet/mobile, dark/light, keyboard/accessibility.

## Quality gates

```text
gofmt modified Go files
go test ./store ./proxy
go test -race ./store ./proxy
go test ./...
pnpm --dir web/frontend typecheck
pnpm --dir web/frontend lint
pnpm --dir web/frontend build
frontend unit tests
Playwright smoke tests
```

Nếu full Go suite vẫn gặp lỗi proxy-environment có sẵn thì báo riêng, không che giấu và không quy lỗi cho Chat khi focused suites pass.

Manual end-to-end:

```text
Login Admin
→ Create Chat
→ Select provider/model or Combo
→ Stream
→ Stop
→ Retry
→ Refresh history
→ Upload image and ask vision model
→ Generate image
→ Download generated image
→ Compare Logs/Dashboard usage
```

---

# 22. INCREMENT VÀ COMMIT ORDER

1. `docs(chat): align AI workspace plan with Kiro-Go`
2. `feat(chat): persist conversations and messages`
3. `feat(chat): add admin conversation APIs`
4. `feat(chat): add text chat page`
5. `feat(chat): stream and cancel generations`
6. `feat(chat): render markdown and code`
7. `feat(chat): support image attachments`
8. `feat(chat): generate and persist images`
9. `test(chat): cover admin chat end to end`

Sau mỗi increment:

1. Implement phần nhỏ nhất hoàn chỉnh.
2. Chạy focused tests.
3. Verify build/UI behavior.
4. Commit riêng.
5. Chuyển sang slice tiếp theo.

---

# 23. DEFINITION OF DONE — DELIVERABLE ĐẦU TIÊN

- Admin vào `/admin/chat` mà không cần API key phía browser.
- Conversation/messages tồn tại sau refresh và restart.
- Chọn đúng provider/model/Combo; unavailable model không gửi được.
- Text stream ổn định.
- Stop hủy upstream và partial response lưu đúng.
- Retry không tạo duplicate user message.
- Markdown/code render an toàn, highlight và copy được.
- Upload/paste/drop ảnh được validate, lưu ngoài SQLite, preview và gửi qua vision pipeline.
- Image generation dùng Codex/Grok/Antigravity router hiện có; output persist, xem lại và download được.
- Session/CSRF/body/path/MIME/XSS protections có tests.
- Usage liên kết request logs hiện có; không có bộ đếm thứ hai.
- Migration v7 → v8 pass.
- Focused Go tests, frontend tests/typecheck/lint/build đạt.
- Mọi lỗi suite có sẵn được báo trung thực.
