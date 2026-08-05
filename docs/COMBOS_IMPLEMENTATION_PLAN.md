# Kế hoạch triển khai Combo cho Kiro-Go

**Trạng thái tài liệu:** thiết kế canonical, đối chiếu code ngày 2026-07-30.  
**Hiện trạng:** storage/registry/Admin API đã có trong working tree nhưng chưa commit; runtime routing, Fusion, UI và advertisement chưa có. Không gọi bất kỳ phase nào “hoàn tất” cho đến khi acceptance tests của phase đó xanh.

## 1. Mục tiêu và phạm vi đầy đủ

Combo là một model ID logic do client gửi; gateway resolve nó thành danh sách **model ID upstream** và áp dụng một trong ba strategy:

- `fallback`: bắt đầu từ model đầu tiên, chỉ chuyển model khi lỗi được classifier cho phép và chưa gửi public byte đầu tiên;
- `round-robin`: chọn **một primary model** theo lượt persistent/sticky; nếu primary thất bại trước first byte, các model còn lại theo thứ tự xoay là fallback candidates;
- `fusion`: gọi panel song song, sau đó trả trực tiếp một kết quả hoặc gọi judge để tổng hợp.

Phạm vi public bắt buộc:

- Claude Messages: `/v1/messages`, `/messages`, `/anthropic/v1/messages`;
- OpenAI Chat Completions: `/v1/chat/completions`, `/chat/completions`;
- OpenAI Responses: `/v1/responses`, `/responses`;
- Claude token counting: `/v1/messages/count_tokens`, `/messages/count_tokens` phải hiểu Combo để validate/count theo canonical request, nhưng **không reserve round-robin, không gọi panel/judge và không làm thay đổi rotation**;
- `/v1/models` và `/models` chỉ advertise Combo sau rollout gate.

Không mở rộng Combo sang image generation/edit trong scope này. Không cho Combo lồng Combo. Không lưu credential, prompt, tool arguments hay panel content trong cấu hình Combo.

## 2. Ground truth hiện tại

Code đã kiểm tra:

- `store/sqlite.go`: schema version 4; combo tables được tạo ở migration v3, ba cột Fusion được thêm ở v4;
- `store/combos.go`: CRUD, optimistic revision, reset và atomic rotation reservation;
- `proxy/admin_combos.go`: validation và Admin API;
- `proxy/handler.go`: registry hydrate/lazy load, admin routing, public protocol dispatch và lifecycle store;
- `proxy/model_fallback.go`, `proxy/account_failover.go`: fallback/account behavior hiện có cần được adapter hóa, không nhân bản;
- `proxy/responses_handler.go`, `proxy/responses_*`: Responses có state/history riêng phải được bảo toàn;
- frontend thật nằm ở `web/frontend/src`, không phải `web/*` chung chung.

Commit hiện tại là `dedb74d`; Combo implementation đang là thay đổi chưa commit. Không ghi “staged” vì trạng thái index có thể thay đổi. `ReserveComboRotation` chưa có caller trong proxy và không protocol public nào resolve Combo. Combo chưa xuất hiện trong `/v1/models`.

Các claim về 9router chỉ là input tham khảo, không phải acceptance đã kiểm chứng của Kiro-Go. Trước khi port một behavior, engineer phải khóa nó bằng source-linked characterization test trong checkout `9router/`; không dựa vào mô tả prose này cho retry/cooldown, grace/quorum, tool flatten hay Responses event conversion.

## 3. Danh tính model và invariant dữ liệu

- `requested_model`: model string nguyên bản từ client; giữ cho public log/response khi protocol cho phép.
- `combo`: immutable snapshot `{id,name,revision,strategy,...}` tại lúc request resolve.
- `candidate_model`: model ID lưu trong `combo_models.model`. **Không bắt buộc dạng `provider/model`**; current catalogs dùng cả model ID không namespace. Provider/account được account pool resolve sau đó.
- `effective_model`, `provider`, `account_id`: upstream thực tế cho từng attempt.
- `serving_model`: model tạo output public; Fusion có panel models và judge model.
- `attempt`: một upstream call; retry account/model là attempt mới nhưng vẫn cùng public `request_id`.

Registry lookup Combo là case-insensitive, nhất quán với DB `COLLATE NOCASE` và Admin validation. Sau lookup phải snapshot/deep-copy `Models` để update/delete đồng thời không đổi request đang chạy.

Không mutate parsed request/history dùng chung giữa attempts. Mỗi candidate/panel/judge nhận deep copy riêng; đặc biệt tools, content blocks, Responses input/history và multimodal byte slices/maps.

## 4. Storage và migration contract

Schema v4:

- `combos(id PRIMARY KEY, name COLLATE NOCASE UNIQUE, strategy, sticky_limit, fusion_quorum, fusion_timeout_ms, judge_model, revision, created_at, updated_at)`;
- `combo_models(combo_id FK ON DELETE CASCADE, position, model, PRIMARY KEY(combo_id,position))`;
- `combo_rotation(combo_id PRIMARY KEY FK ON DELETE CASCADE, revision, model_index, use_count)`.

SQLite dùng WAL, `synchronous=NORMAL`, `busy_timeout=5000`, foreign keys ở DSN và PRAGMA, một connection open/idle, lifetime một giờ, file 0600.

Migration facts và việc cần harden:

- fresh DB seed version 4 rồi vẫn chạy idempotent table creation;
- v2→v3 tạo tables đã chứa Fusion columns; v3→v4 `ALTER TABLE` ba columns với duplicate-column guard;
- hiện v4 `ALTER` statements và version bump không nằm cùng transaction: phải sửa hoặc có recovery test cho partial migration trước production;
- tests bắt buộc: fresh, v2→v4, v3→v4 thật, partial v4 từng column, newer-version rejection, data preservation.

Repository validation-neutral. Create revision 1; update thay toàn bộ model list, tăng revision và reset rotation; delete revision-bound và cascade; nil/closed store trả `ErrStorageUnavailable` nhất quán.

`ReserveComboRotation(id, revision)` là operation duy nhất cấp lượt. Nó trả ordered candidates bắt đầu ở reserved primary, tăng `use_count`, chuyển index sau đúng `sticky_limit` reservations, và reject stale revision. Reservation diễn ra trước network call; failed/cancelled request vẫn tiêu thụ lượt. Combo một model không cần ghi rotation state.

`ResetComboRotation` hiện delete trước existence check; behavior cuối vẫn đúng nhưng implementation nên check/delete trong một transaction. Admin reset phải nhận `{revision}` và repository reset phải revision-bound để request reset stale không xóa state của revision mới.

## 5. Registry, Admin API và concurrency

Registry `combosByID`/`combosByName` chỉ là read cache. Hydrate toàn bộ sau store open, lazy retry khi chưa loaded, publish sau DB commit. `GET /admin/api/combos` sort case-insensitive theo name rồi ID làm tie-breaker. Không giữ combo lock qua DB/network call.

Admin routes, tất cả qua auth/CSRF policy hiện hành:

- `GET /admin/api/combos` → `{data:[...]}`;
- `POST /admin/api/combos` → 201 + Combo;
- `GET /admin/api/combos/{id}`;
- `PUT /admin/api/combos/{id}`;
- `DELETE /admin/api/combos/{id}` body `{revision}` → 204, không body;
- `POST /admin/api/combos/{id}/reset-rotation` body `{revision}` → `{ok:true}`.

Decoder reject unknown fields, empty/trailing/multiple JSON values và oversized body (`http.MaxBytesReader`). Set `Content-Type` before status. Error envelope là `{error:string, fields?:{[field]:string}}`; 400 malformed/missing revision, 404 not found, 409 name/revision conflict, 422 validation, 503 unavailable/internal persistence error. Không classify SQLite conflict bằng substring `"unique"`; map typed repository errors.

Validation:

- name trim, regex `^[A-Za-z0-9_.-]+$`, 1–128 bytes, case-insensitive unique;
- reserved aliases hiện có: `auto`, `gpt-4`, `gpt-4o`; đồng thời reject collision với mọi direct model ID đã biết để Combo không shadow model thật;
- strategy enum; sticky 1–10000; models 1–8; mỗi model trim, bounded length, case-insensitive unique, known direct model, không phải Combo;
- Fusion: quorum 1..N, timeout 100..300000 ms, judge là known direct non-Combo model;
- non-Fusion payload phải reject hoặc normalize Fusion-only fields theo một contract cố định; chọn **normalize về zero/empty khi persist và response**;
- model-known check dùng hợp nhất static provider catalogs, fresh cache và account pool. Không gọi network trong validation request; cache stale/empty phải cho lỗi field rõ ràng, không âm thầm coi một fallback Anthropic list là toàn bộ provider catalog.

## 6. Routing core và composition với account fallback

Tạo một core dùng chung, không ba engine riêng:

1. protocol handler parse/auth/validate shape, lưu `requested_model`;
2. resolve Combo case-insensitive; direct model đi nguyên path hiện có;
3. deep-copy Combo và canonical request;
4. detect hard modalities từ **trailing current user turn** và protocol metadata;
5. tạo candidate order: fallback giữ config order; round-robin gọi reservation một lần; capability stable-partition các model đủ capability lên trước;
6. với từng candidate, chạy account selector và current account failover/model mapping;
7. classifier quyết định next account, next candidate, bounded wait hoặc terminal;
8. chỉ commit public headers/body khi candidate đã có response hợp lệ; ghi attempt usage/log chính xác.

Account fallback được exhaust trong cùng candidate trước khi chuyển Combo candidate, trừ policy hiện có đánh dấu account-independent/non-retryable. Direct-model path không còn cross-model fallback (removed; use Combos); multi-model chỉ qua Combos. Trong Combo, mỗi candidate chỉ exhaust account của model đó (GetNextForModelExcluding), không nhảy model ngoài candidate set.

Retryable mặc định: transport trước first byte, 408, 429, 500, 502, 503, 504, quota/capacity typed. Terminal: malformed/unsupported request, deterministic auth/config error, policy refusal, context overflow không thể sửa bằng candidate capability, client cancellation. 401/403 chỉ next-account khi classifier xác nhận credential/account-specific; không tự động next-model.

## 7. Typed retry và first-byte invariant

Dùng type chung có ít nhất `Class`, `Status`, provider code, `RetryAfter`, `BeforeFirstByte`, `Usage`, `Cause`. Không parse error message rải rác trong protocol handlers.

`Retry-After` hỗ trợ delta-seconds và HTTP-date, clamp theo request deadline và max wait cấu hình. Không sleep nếu client context done. Test negative/huge/malformed values.

Sau public first byte không được đổi account/model, không trộn streams. Cần response writer wrapper ghi nhận header/body flush thực tế. Upstream SSE headers thành công chưa đồng nghĩa public first byte; core có thể buffer đến event đầu hợp lệ với cap/time limit. Mid-stream error emit protocol-correct terminal error khi có thể, mark partial usage, rồi dừng.

Round-robin có primary duy nhất nhưng **được fallback trước first byte** theo ordered candidates; điều này thay thế claim mâu thuẫn “một candidate duy nhất”.

## 8. Canonical IR và protocol adapters

IR phải đủ lossless cho feature hiện có, không chỉ “messages + tools” tối thiểu:

- model/requested model, stream, system/developer instructions;
- ordered content blocks/items gồm text, image, document/PDF, audio/video nếu protocol nhận;
- tool definitions, tool choice, parallel tool calls, tool call/result IDs và error state;
- thinking/reasoning fields, response format/JSON schema, sampling/token limits và metadata;
- Responses fields cần cho `previous_response_id`, stored history and item/event IDs;
- cancellation/deadline và auth/request metadata.

Claude adapter giữ block order, `tool_use/tool_result`, thinking/signature fields và stop reason. Chat adapter giữ roles, developer/system semantics, multi-tool ordering, finish reason và deltas. Responses adapter giữ input item types, stateful-history semantics, response IDs/status, `response.output_*` event ordering và usage. Unknown fields phải theo compatibility policy hiện hành, không bị Combo tự ý drop.

`count_tokens` resolve Combo nhưng không chọn account theo rotation; count canonical payload theo deterministic first eligible candidate hoặc provider-neutral estimator hiện hành, và document rằng kết quả là estimate nếu candidates tokenize khác nhau.

## 9. Capability policy

Capability source priority: static provider registry → fresh account/model metadata → conservative unknown. Định nghĩa typed capability set và provenance/age; đừng suy capability từ substring model name.

Hard: image/vision, PDF/document, audio input, video input. Candidate thiếu hard capability không được nhận payload; filter nó trước call. Nếu không candidate phù hợp, trả protocol-specific 400/422 trước first byte. Soft: tools/search/structured output; ưu tiên hoặc trả unsupported rõ ràng, không strip âm thầm.

Detection chỉ xét trailing current user turn cho modality routing, nhưng adapter vẫn phải preserve media trong history. Test content dạng string/array, empty trailing blocks, tool-result turn sau user media, Responses item variants, remote URLs/base64/file IDs và malformed blocks.

## 10. Fusion contract

- fan-out tối đa N≤8 với global semaphore và per-request bounded concurrency;
- mỗi panel dùng cloned request, `stream=false`, không tools/tool_choice; tool history được serialize thành **data-delimited prose** theo protocol-aware function, không mutate original;
- hard timeout lấy `fusionTimeoutMs`; quorum lấy persisted value; grace mặc định 8s nhưng phải là constant/config được test;
- khi quorum đạt, đợi grace hoặc tất cả panel; cancel stragglers sau quyết định;
- panel outputs có byte/token cap; không log content mặc định.

Outcomes:

- 0 success → protocol-specific 503;
- đúng 1 success → direct answer, không judge;
- ≥2 successes và đạt quorum → judge available results;
- timeout/all-finished nhưng success count >0 và <quorum: **degrade trực tiếp chỉ khi đúng 1; nếu >1 trả 503 quorum-not-met**. Không judge dưới quorum;
- judge retryable failure dùng normal account policy tối đa budget; không rerun panels;
- judge terminal failure trả error, không giả fused;
- client cancel hủy panel/judge ngay.

Judge model phải đi qua direct-model executor với Combo resolution disabled để chống recursion. Judge prompt đặt panel text trong delimiters, coi là untrusted data, anonymize Source N, yêu cầu không lộ model/panel. Judge giữ client stream preference. Việc judge dùng client tools cần một tool-loop contract rõ: gateway hiện không tự thực thi arbitrary client tools, vì vậy judge có thể emit tool calls cho client nhưng continuation request phải không rerun panel vô điều kiện; lưu/associate Fusion state hoặc cấm judge tools trong phase đầu. Chọn phase đầu: **judge tools bị tắt**, thêm judge tool continuation ở phase hardening riêng.

Direct panel result phải được chuyển từ canonical result sang client protocol với fresh public IDs/model metadata; không forward raw provider envelope của protocol khác.

## 11. Logging, usage, billing và bảo mật

Mở rộng log schema/API, không chỉ struct RAM, để mỗi public request và attempts biểu diễn được: request ID, requested/effective model, Combo ID/revision/strategy, candidate index, provider/account, fusion role, status/retry class, TTFT, latency, usage/cache usage, partial flag. Migration mới phải tăng schema version; không nhét field mới vào v4 prose.

Billing dùng usage thực của mọi panel/judge/retry, effective model cho price lookup, requested model cho display. Nếu upstream fail sau khi tiêu token vẫn ghi usage. Không double count qua SSE→JSON/Responses conversion. Quy định aggregate public usage: tổng billable attempts cho admin/billing; protocol response usage là usage của serving result/judge theo compatibility contract, kèm internal aggregate riêng.

Redact credential, raw API key, prompt, tool args, panel content. Cap request/model/name/prompt sizes. Model ID validation không tự nó chống SSRF; executor/provider selection vẫn phải dùng allowlisted provider implementations và existing remote-Kiro SSRF protections.

## 12. Lifecycle và consistency

`runtimeStoreMu.RLock` bao từng store operation; `Close` lấy exclusive lock sau khi flusher dừng. Hiện `Close()` chỉ đóng `stopRuntime`; `backgroundRefresh` và `backgroundStatsSaver` vẫn chạy, và startup sleep 10s không interruptible. Đưa cả hai vào wait group, close `stopRefresh`/`stopStatsSaver`, thay sleep bằng timer select, làm Close idempotent và race-test.

Admin mutation commit DB rồi publish registry tạo cửa sổ nhỏ DB/cache lệch; chấp nhận eventual micro-window nhưng test concurrent reads. Nếu publish logic panic/fail, invalidate `combosLoaded` để lazy reload. Reset rotation không cần registry mutation.

## 13. Frontend scope thực tế

Implement dưới `web/frontend/src`:

- `features/combos`, `services/combos.service.ts`, query/mutation hooks, types, query keys, route/nav;
- list/create/edit/delete/reset với revision-aware conflict refresh;
- ordered model picker (keyboard accessible), strategy controls, Fusion quorum/timeout/judge, capability badges;
- Fusion N+1 warning, validation/error/loading/empty states, mobile layout, copy-full-ID;
- locale keys theo i18n mechanism hiện tại; không hardcode text;
- tests cho service, form validation, reorder, 409 rollback/refresh, focus and keyboard behavior.

Không sửa generated `web/dist` thủ công; build frontend theo pipeline repo rồi cập nhật artifacts chỉ khi project convention yêu cầu.

## 14. `/v1/models` và rollout

Trước routing green, không advertise. Sau đó advertisement phải áp dụng cho cả `/v1/models` và `/models`, auth visibility nhất quán, ID là Combo name, `owned_by:"kiro"`, metadata strategy/capabilities không giả provider. Capability advertised là conservative intersection/declared support, không union gây false promise.

Có feature gates riêng: resolver, fallback, round-robin, Fusion, advertisement. Unknown/direct model behavior giữ nguyên khi resolver off. In-flight request giữ snapshot khi config đổi. Rollback tắt resolver/advertisement nhưng giữ DB. Canary đo latency/TTFT/retry/usage/quorum/cost.

## 15. Test matrix bắt buộc

### Storage/migration

Fresh/v2/v3/partial-v4/newer DB; CRUD/case/revision/cascade; Fusion fields; reset stale revision; sticky 1/N; concurrent reservation exact distribution; restart/update reset; close-during-operation; nil/closed store; race tests.

### Admin

Session/legacy auth and CSRF behavior; every exact route/method; body size, empty/unknown/trailing/multiple JSON; all boundaries; case collision/direct-model collision/nested Combo; stale cache; typed 404/409/422/503; publish-after-commit; update/delete/reset races; 204 empty body; deterministic list.

### Routing

Direct model unchanged; case-insensitive Combo; immutable snapshot; all aliases; count_tokens no reservation; fallback/account ordering; classifier status/provider-code matrix; `Retry-After`; round-robin sticky/restart/concurrency/failure consumption; capability filtering; no switch after first byte; header/flush/mid-SSE errors; cancel/deadline.

### Protocol adapters

Mỗi protocol: stream/non-stream, multimodal mixed blocks, system/developer, thinking/reasoning, multiple tools/results/errors, structured output, usage/cache tokens, cancellation, translation error. Verify IDs, model metadata, stop/finish reasons and exact SSE event order. Responses thêm `previous_response_id`, stored/not-stored, parallel items và continuation.

### Fusion

0/1/<quorum/quorum/all panels; grace/hard timeout; semaphore saturation; panel panic/HTTP/oversize; deterministic Source ordering; prompt-injection delimiters; judge retry/failure; judge tools disabled; cancellation/leak; hard modality filtering; no request mutation; canonical direct-result conversion; all-attempt accounting.

### Regression commands

- `go test ./store ./proxy`
- `go test -race ./store ./proxy`
- frontend lint/typecheck/test/build theo scripts trong `web/frontend/package.json`

Không chấp nhận baseline “known failing” bằng prose: ghi issue/test cụ thể, sửa hoặc quarantine có owner và expiry. Không hardcode line numbers của tests vì stale nhanh.

## 16. File map triển khai

Đã tồn tại/đang thay đổi:

- `store/sqlite.go`, `store/combos.go`, `store/combos_test.go`;
- `proxy/admin_combos.go`, `proxy/admin_combos_test.go`, `proxy/handler.go`;
- integration lifecycle ở `main.go`, `proxy/admin_apikeys.go`, `proxy/check_key.go`.

Dự kiến, tên có thể gộp nhưng trách nhiệm không được thiếu:

- `proxy/combo_router.go`: resolve/snapshot/strategy/core;
- `proxy/combo_retry.go`: typed errors/classifier/Retry-After;
- `proxy/combo_capabilities.go`: metadata và detection;
- `proxy/combo_ir.go` + protocol adapter files;
- `proxy/combo_fusion.go`: fan-out/quorum/judge;
- tests tương ứng và protocol contract tests;
- modify `proxy/handler.go` public dispatch, `proxy/responses_handler.go`, current Claude/OpenAI handlers, `proxy/model_fallback.go`, `proxy/account_failover.go`;
- logging migration/store structs cho observability;
- frontend paths nêu ở §13;
- model handler trong `proxy/handler.go` (hiện không có `proxy/models.go`) chỉ ở phase advertisement.

## 17. Phases, dependency và acceptance

1. **Correctness of existing persistence/Admin:** fix migration atomicity, typed conflicts, reset revision, body caps, normalization/collision rules, deterministic list và lifecycle tests. Tất cả current Combo tests + store/proxy suites xanh.
2. **IR + fallback/round-robin routing:** cả ba protocol và count_tokens; direct regression; typed retry; account composition; first-byte invariant; persistent rotation.
3. **Fusion:** bounded panels, strict quorum/grace/timeout, protocol conversion, judge without tools, cancellation, security and accounting.
4. **Observability/lifecycle hardening:** schema migration cho attempt logs, billing aggregation, all goroutines stopped, race/no-leak/load tests.
5. **Admin frontend:** full CRUD/reset/revision UX, ordered models, Fusion controls, i18n/a11y/mobile and frontend tests.
6. **Advertisement/rollout:** feature gates, both model-list aliases, authorization/capability metadata, canary and rollback tests.
7. **Optional judge-tool continuation:** chỉ sau khi có persisted/request-scoped Fusion continuation design; không block base Combo release.

Mỗi phase là một commit/PR độc lập sau khi acceptance xanh; tài liệu này không yêu cầu commit.

## 18. Fixed decisions và open decisions phải chốt trước code

Fixed:

1. Combo lookup case-insensitive; direct model không bị Combo shadow.
2. Reservation trước network, persistent và revision-bound.
3. Round-robin primary có fallback trước first byte; không fallback sau first byte.
4. Hard capability filter, không strip modality.
5. Fusion panel non-streaming và không tools; quorum là điều kiện bắt buộc để judge.
6. 0 success → 503; 1 success → direct; >1 nhưng dưới quorum → quorum error.
7. Judge direct/non-Combo, panel data untrusted, judge tools tắt trong base release.
8. Billing tính mọi usage thực; protocol usage và internal aggregate tách rõ.
9. Không advertise trước routing contract green.
10. Chỉ sửa tài liệu này trong task review hiện tại; không commit.

Phải chốt bằng ADR/test trong phase tương ứng, không để engineer tự đoán:

- exact max body/model/panel-output sizes và global Fusion semaphore size;
- grace window có config hay constant 8s;
- public response `model` giữ Combo name ở từng protocol hay effective serving model (khuyến nghị Combo name để client identity ổn định);
- count-token estimator khi candidates khác tokenizer;
- retention/schema shape cho per-attempt logs;
- behavior khi registry unavailable nhưng direct model request vẫn hợp lệ (khuyến nghị direct fail-open, Combo cannot resolve).
