# AGENT.md — Quy tắc codebase FE Kiro-Go Admin

Tài liệu này là luật bắt buộc cho mọi code trong `web/`. Đọc trước khi viết bất kỳ component/hook/service nào. Mục tiêu: clean code, tách layer rõ, tái sử dụng tối đa, bám sát backend contract.

---

## 1. Tech stack (không tự ý thêm lib ngoài danh sách)

| Nhóm | Công nghệ | Ghi chú |
|---|---|---|
| Build | Vite + React 18 + TypeScript (strict) | build ra `web/dist/`, `base: '/admin/'` |
| Data/server state | **TanStack Query** | mọi call API đi qua hook query/mutation, KHÔNG fetch trực tiếp trong component |
| UI base | **Tailwind CSS + shadcn/ui** (Radix) | copy-source vào `components/ui/`, không import runtime |
| UI động | **animate-ui** + **Motion** | copy-source vào `components/ui/animate/`; dùng cho tabs, counter, dialog enter/exit, shimmer, view transition |
| Client state | **Zustand** | CHỈ UI state (theme, lang, filter, selection, privacy) |
| i18n | **react-i18next** | dùng `locales/{vi,zh,en}.json`, default `vi`, fallback `zh` |
| Toast | **sonner** | giữ variant + `duration:0` sticky + `onClick` |
| Routing | **React Router** | 7 view + login guard |

Quy tắc: không thêm dependency mới nếu shadcn/animate-ui đã có component tương đương. Ưu tiên copy-source (shadcn/animate-ui) hơn cài package đóng gói.

---

## 2. Kiến trúc phân tầng (BẮT BUỘC theo chiều phụ thuộc)

```
component  →  hook (TanStack Query)  →  service (axios)  →  httpClient (axios instance)
   ↑ dùng         ↑ bọc                     ↑ không React
shared components / ui / animate
```

- **`services/`**: gọi API qua `http.*` (axios) trong `httpClient.ts`, KHÔNG import React, KHÔNG import store. Nhận tham số, trả `Promise<T>` đã typed. Đây là nơi DUY NHẤT biết đường dẫn endpoint.
- **`hooks/queries` + `hooks/mutations`**: bọc service bằng `useQuery`/`useMutation`. Mutation phải `invalidateQueries` đúng key. Component KHÔNG được gọi service trực tiếp.
- **`components/` (feature)**: chỉ render + gọi hook. Không chứa logic fetch, không hardcode URL.
- **`components/shared/`, `components/ui/`, `components/ui/animate/`**: tái sử dụng, không phụ thuộc feature nào.
- **`stores/`**: Zustand, chỉ UI state. KHÔNG cache dữ liệu server ở đây (đó là việc của TanStack Query).

Cấm phụ thuộc ngược: service không import hook/component; shared không import feature.

---

## 3. Quy tắc tái sử dụng

- Component xuất hiện ở **>1 feature** → nâng lên `components/shared/`. Ví dụ có sẵn: `UsageBar`, `StatCard`, `StatusBadge`, `DataTable`, `ConfirmDialog`, `CopyButton`, `RegionSelect`, `PasswordInput`, `ProviderIcon`, `EmptyState`, `HamsterLoader`.
- Mọi OAuth flow dùng CHUNG hook `hooks/useOAuthFlow.ts` (quản lý `sessionId` + polling + cleanup). KHÔNG copy logic poll cho từng provider.
- Bảng có sort/pagination client-side dùng CHUNG `shared/DataTable` (ApiKeys, Usage, Logs).
- Format số/ngày, mask email → `lib/format.ts`, `lib/mask.ts`. KHÔNG viết lại inline.
- Ba dòng lặp thì chấp nhận; đừng tạo abstraction non khi chưa có lần dùng thứ 2.

---

## 4. Backend contract (bám CHÍNH XÁC, dễ sai)

- Auth = **cookie-session** (không gửi password mỗi request). Login `POST /login` → backend set cookie session HttpOnly + cookie CSRF đọc được (`kiro_csrf`). `httpClient` (axios) tự gắn `X-CSRF-Token` từ cookie đó cho request mutation + `withCredentials`. Logout `POST /logout`. 401 ở bất kỳ đâu → interceptor gọi `forceLogout()` đá về login. Header `X-Admin-Password` chỉ còn là fallback tương thích ngược ở backend, FE KHÔNG dùng.
- **Envelope KHÔNG đồng nhất**: `GET /accounts` và `/accounts/{id}/full` trả bare array/object; đa số mutation wrap `{success:true,...}`; vài config GET trả bare field map. → Typing **per-endpoint** trong `types/`, KHÔNG giả định envelope chung.
- **Timestamp**: account/status/logs = **unix giây**; `/export` = **mili giây**. Format phải phân biệt rõ (`lib/format.ts` có hàm riêng cho từng đơn vị).
- Non-admin call (`/v1/models`, `/v1/stats`, `/version`, `/check/api/lookup`) không đi qua session/CSRF — gọi bằng axios trần hoặc service riêng.
- Lỗi non-2xx: `httpClient` ném `ApiError { status, error }`; hook trả `error` cho UI hiển thị qua toast.

---

## 5. Clean code

**Đặt tên**
- Component: `PascalCase.tsx`. Hook: `useXxx.ts`. Service: `xxx.service.ts`. Type file: `xxx.ts` trong `types/`.
- Boolean: `isLoading`, `hasToken`, `canRefresh`. Handler: `handleClick`, `onSubmit`.
- Query key tập trung ở `config/queryKeys.ts` — không rải chuỗi magic.

**Component**
- Một component một việc. File >200 dòng là tín hiệu cần tách.
- Props typed rõ ràng, không `any`. Không truyền `state` object khổng lồ; truyền đúng thứ cần.
- Không side-effect trong render; dùng `useEffect`/event handler. Cleanup timer/subscription trong `useEffect` return.
- Ưu tiên component có sẵn của shadcn/animate-ui hơn tự dựng (Select, Dialog, Switch, Tabs, Table).

**TypeScript**
- `strict: true`. Cấm `any` (dùng `unknown` + narrow). Cấm `@ts-ignore` trừ khi có comment lý do.
- Type dữ liệu API sống trong `types/`, không khai báo inline rải rác.

**Style**
- Chỉ Tailwind + token của design system. KHÔNG viết CSS file riêng cho từng component (trừ global trong `index.css`).
- Dark/light qua class `.dark` + CSS var; component phải hoạt động cả 2 theme.
- Dùng `lib/cn.ts` (clsx + tailwind-merge) để ghép class có điều kiện.

**Comment**
- Mặc định KHÔNG comment. Chỉ comment khi WHY không hiển nhiên (ràng buộc ẩn, workaround, đơn vị timestamp). Không comment mô tả WHAT.

---

## 6. i18n

- Mọi chuỗi hiển thị đi qua `t('key')`. KHÔNG hardcode text tiếng Việt/Anh/Trung trong JSX.
- Thêm key mới phải thêm đủ cả 3 file `vi/zh/en.json`.
- Giữ nguyên format key dotted phẳng + arg vị trí `{0}` như bản cũ để tái dùng 708 key hiện có.

---

## 7. Animation (animate-ui)

- Dùng animate-ui cho: chuyển view (layout/transition), animated tabs & segmented control (Grok/Codex dual-mode, settings sub-nav), counter cho KPI overview, dialog/sheet enter-exit, shimmer/skeleton loading, animated switch, micro-interaction account card.
- Animation phải tôn trọng `prefers-reduced-motion`.
- Không animation nào chặn tương tác hay kéo dài >300ms cho micro-interaction.

---

## 8. Definition of Done (mỗi PR/feature)

- [ ] `pnpm typecheck` + `pnpm lint` sạch.
- [ ] Không fetch trực tiếp trong component (đi qua hook → service).
- [ ] Không hardcode URL ngoài `services/`, không hardcode chuỗi UI ngoài i18n.
- [ ] Component tái dùng được đặt đúng `shared/`.
- [ ] Hoạt động cả light/dark, cả 3 ngôn ngữ.
- [ ] Test golden path trên browser với backend Go thật (`pnpm dev` + Vite proxy).
