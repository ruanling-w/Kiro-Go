// apiDocs.ts — static catalog for the API docs page: endpoint table rows +
// CLI/SDK/GUI snippet templates. Placeholders {{BASE}} {{KEY}} {{MODEL}}
// {{MODEL_OPUS}} {{MODEL_HAIKU}} are filled at render time by fillSnippet.
import {
  Terminal,
  Box,
  Code2,
  MonitorSmartphone,
  Compass,
  type LucideIcon,
} from 'lucide-react'

export interface ApiEndpointDoc {
  method: 'GET' | 'POST'
  path: string
  aliases?: string[]
  auth: boolean
  group: 'anthropic' | 'openai' | 'meta'
  descKey: string
  sampleBody?: string
}

export const API_ENDPOINTS: ApiEndpointDoc[] = [
  {
    method: 'POST',
    path: '/v1/messages',
    aliases: ['/messages', '/anthropic/v1/messages'],
    auth: true,
    group: 'anthropic',
    descKey: 'apiDocs.ep.messages',
    sampleBody: `{
  "model": "{{MODEL}}",
  "max_tokens": 1024,
  "messages": [{"role": "user", "content": "Hello"}]
}`,
  },
  {
    method: 'POST',
    path: '/v1/messages/count_tokens',
    aliases: ['/messages/count_tokens'],
    auth: true,
    group: 'anthropic',
    descKey: 'apiDocs.ep.countTokens',
  },
  {
    method: 'POST',
    path: '/v1/chat/completions',
    aliases: ['/chat/completions'],
    auth: true,
    group: 'openai',
    descKey: 'apiDocs.ep.chatCompletions',
    sampleBody: `{
  "model": "{{MODEL}}",
  "messages": [{"role": "user", "content": "Hello"}]
}`,
  },
  {
    method: 'POST',
    path: '/v1/responses',
    aliases: ['/responses'],
    auth: true,
    group: 'openai',
    descKey: 'apiDocs.ep.responses',
    sampleBody: `{
  "model": "{{MODEL}}",
  "input": "Hello"
}`,
  },
  {
    method: 'POST',
    path: '/v1/images/generations',
    aliases: ['/images/generations'],
    auth: true,
    group: 'openai',
    descKey: 'apiDocs.ep.imagesGen',
  },
  {
    method: 'POST',
    path: '/v1/images/edits',
    aliases: ['/images/edits'],
    auth: true,
    group: 'openai',
    descKey: 'apiDocs.ep.imagesEdit',
  },
  {
    method: 'POST',
    path: '/v1/embeddings',
    aliases: ['/embeddings'],
    auth: true,
    group: 'openai',
    descKey: 'apiDocs.ep.embeddings',
    sampleBody: `{
  "model": "voyage-4-large",
  "input": "Text to embed"
}`,
  },
  {
    method: 'POST',
    path: '/v1/rerank',
    aliases: ['/rerank'],
    auth: true,
    group: 'openai',
    descKey: 'apiDocs.ep.rerank',
    sampleBody: `{
  "model": "rerank-2.5",
  "query": "query string",
  "documents": ["doc 1", "doc 2"]
}`,
  },
  {
    method: 'GET',
    path: '/v1/models',
    aliases: ['/models'],
    auth: false,
    group: 'meta',
    descKey: 'apiDocs.ep.models',
  },
  {
    method: 'GET',
    path: '/v1/stats',
    auth: true,
    group: 'meta',
    descKey: 'apiDocs.ep.stats',
  },
  {
    method: 'GET',
    path: '/health',
    aliases: ['/'],
    auth: false,
    group: 'meta',
    descKey: 'apiDocs.ep.health',
  },
]

export interface DocBlock {
  titleKey: string
  /** Path label above the block, e.g. ~/.claude/settings.json */
  filename?: string
  lang: 'bash' | 'toml' | 'json' | 'python' | 'ts' | 'text'
  code: string
  noteKey?: string
}

export interface ToolGuide {
  id: 'claude-code' | 'codex' | 'curl' | 'sdk' | 'embeddings' | 'gui'
  labelKey: string
  icon: LucideIcon
  logo?: string
  blocks: DocBlock[]
}

export interface SnippetVars {
  base: string
  key: string
  model: string
  modelOpus?: string
  modelHaiku?: string
}

export function fillSnippet(code: string, v: SnippetVars): string {
  const base = (v.base || 'http://localhost:8080').replace(/\/+$/, '')
  const key = v.key || 'sk-your-key'
  const model = v.model || 'claude-sonnet-4.5'
  const modelOpus = v.modelOpus || model
  const modelHaiku = v.modelHaiku || model
  return code
    .replaceAll('{{BASE}}', base)
    .replaceAll('{{KEY}}', key)
    .replaceAll('{{MODEL_OPUS}}', modelOpus)
    .replaceAll('{{MODEL_HAIKU}}', modelHaiku)
    .replaceAll('{{MODEL}}', model)
}

export const TOOL_GUIDES: ToolGuide[] = [
  {
    id: 'embeddings',
    labelKey: 'apiDocs.tools.embeddings',
    icon: Compass,
    blocks: [
      {
        titleKey: 'apiDocs.tools.embCurl',
        lang: 'bash',
        code: `# 1. Tạo Embeddings (Voyage AI / OpenAI / Gemini)
curl -sS "{{BASE}}/v1/embeddings" \\
  -H "content-type: application/json" \\
  -H "authorization: Bearer {{KEY}}" \\
  -d '{
    "model": "voyage-4-large",
    "input": ["Tìm kiếm ngữ nghĩa", "Semantic search with Voyage AI"]
  }'

# 2. Xếp hạng lại kết quả (Voyage AI Reranker)
curl -sS "{{BASE}}/v1/rerank" \\
  -H "content-type: application/json" \\
  -H "authorization: Bearer {{KEY}}" \\
  -d '{
    "model": "rerank-2.5",
    "query": "Thủ đô của Việt Nam là gì?",
    "documents": [
      "Hà Nội là thủ đô của Việt Nam.",
      "Thành phố Hồ Chí Minh là trung tâm kinh tế lớn nhất.",
      "Đà Nẵng là thành phố đáng sống."
    ],
    "top_k": 2
  }'`,
        noteKey: 'apiDocs.tools.embCurlNote',
      },
      {
        titleKey: 'apiDocs.tools.embPython',
        filename: 'python',
        lang: 'python',
        code: `from openai import OpenAI
import requests

client = OpenAI(
    base_url="{{BASE}}/v1",
    api_key="{{KEY}}",
)

# --- 1. Tạo Embeddings (Hỗ trợ voyage-4-large, text-embedding-3-small, text-embedding-004...) ---
emb_res = client.embeddings.create(
    model="voyage-4-large",
    input=[
        "Khám phá trí tuệ nhân tạo",
        "Machine learning and vector embeddings",
    ],
)
for item in emb_res.data:
    print(f"Index {item.index}: Vector chiều dài = {len(item.embedding)}")

# --- 2. Reranking (Hỗ trợ rerank-2.5, rerank-2.5-lite, rerank-2...) ---
rerank_res = requests.post(
    "{{BASE}}/v1/rerank",
    headers={
        "Authorization": "Bearer {{KEY}}",
        "Content-Type": "application/json",
    },
    json={
        "model": "rerank-2.5",
        "query": "Thủ đô của Việt Nam là gì?",
        "documents": [
            "Hà Nội là thủ đô của Việt Nam.",
            "Hồ Chí Minh là thành phố đông dân nhất.",
            "Đà Lạt nổi tiếng với khí hậu mát mẻ quanh năm."
        ],
        "top_k": 2,
    },
)

results = rerank_res.json()
print("\\nKết quả xếp hạng Rerank:")
for r in results.get("results", []):
    print(f"Vị trí {r['index']} (Điểm: {r['relevance_score']:.4f}): {r.get('document')}")`,
        noteKey: 'apiDocs.tools.embPythonNote',
      },
      {
        titleKey: 'apiDocs.tools.embNode',
        filename: 'node / typescript',
        lang: 'ts',
        code: `import OpenAI from "openai";

const client = new OpenAI({
  baseURL: "{{BASE}}/v1",
  apiKey: "{{KEY}}",
});

// 1. Tạo Embeddings
const embeddings = await client.embeddings.create({
  model: "voyage-4-large",
  input: ["Vector search integration", "Học máy hiện đại"],
});
console.log("Vector length:", embeddings.data[0].embedding.length);

// 2. Rerank via HTTP POST /v1/rerank
const rerankResponse = await fetch("{{BASE}}/v1/rerank", {
  method: "POST",
  headers: {
    "Authorization": "Bearer {{KEY}}",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    model: "rerank-2.5",
    query: "What is Kiro Proxy?",
    documents: [
      "Kiro Proxy is a high-performance AI API gateway.",
      "The sky is blue and sunny today."
    ],
    top_k: 1,
  }),
});
const rerankData = await rerankResponse.json();
console.log("Top result:", rerankData.results?.[0]);`,
      },
    ],
  },
  {
    id: 'claude-code',
    labelKey: 'apiDocs.tools.claudeCode',
    icon: Terminal,
    blocks: [
      {
        titleKey: 'apiDocs.tools.claudeSettings',
        filename: '~/.claude/settings.json',
        lang: 'json',
        code: `{
  "env": {
    "ANTHROPIC_BASE_URL": "{{BASE}}",
    "ANTHROPIC_AUTH_TOKEN": "{{KEY}}",
    "ANTHROPIC_API_KEY": "{{KEY}}",
    "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
    "API_TIMEOUT_MS": "600000",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "{{MODEL}}",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "{{MODEL_OPUS}}",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "{{MODEL_HAIKU}}"
  },
  "model": "sonnet",
  "hasCompletedOnboarding": true
}`,
        noteKey: 'apiDocs.tools.claudeNote',
      },
      {
        titleKey: 'apiDocs.tools.claudeEnv',
        filename: 'shell (optional)',
        lang: 'bash',
        code: `export ANTHROPIC_BASE_URL="{{BASE}}"
export ANTHROPIC_AUTH_TOKEN="{{KEY}}"
export ANTHROPIC_API_KEY="{{KEY}}"
export CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC="1"
export API_TIMEOUT_MS="600000"
export ANTHROPIC_DEFAULT_SONNET_MODEL="{{MODEL}}"
export ANTHROPIC_DEFAULT_OPUS_MODEL="{{MODEL_OPUS}}"
export ANTHROPIC_DEFAULT_HAIKU_MODEL="{{MODEL_HAIKU}}"
claude`,
      },
    ],
  },
  {
    id: 'codex',
    labelKey: 'apiDocs.tools.codex',
    icon: Box,
    logo: '/admin/codex-color.svg',
    blocks: [
      {
        titleKey: 'apiDocs.tools.codexConfig',
        filename: '~/.codex/config.toml',
        lang: 'toml',
        code: `# Kiro Proxy Configuration for Codex CLI
model = "{{MODEL}}"
model_provider = "kiro-go"

[model_providers.kiro-go]
name = "Kiro Proxy"
base_url = "{{BASE}}/v1"
wire_api = "responses"

[agents.subagent]
model = "{{MODEL}}"`,
        noteKey: 'apiDocs.tools.codexNote',
      },
      {
        titleKey: 'apiDocs.tools.codexAuth',
        filename: '~/.codex/auth.json',
        lang: 'json',
        code: `{
  "auth_mode": "apikey",
  "OPENAI_API_KEY": "{{KEY}}"
}`,
      },
    ],
  },
  {
    id: 'curl',
    labelKey: 'apiDocs.tools.curl',
    icon: Terminal,
    blocks: [
      {
        titleKey: 'apiDocs.tools.curlAnthropic',
        lang: 'bash',
        code: `curl -sS "{{BASE}}/v1/messages" \\
  -H "content-type: application/json" \\
  -H "anthropic-version: 2023-06-01" \\
  -H "x-api-key: {{KEY}}" \\
  -H "authorization: Bearer {{KEY}}" \\
  -d '{
    "model": "{{MODEL}}",
    "max_tokens": 1024,
    "messages": [{"role": "user", "content": "Hello"}]
  }'`,
      },
      {
        titleKey: 'apiDocs.tools.curlOpenAI',
        lang: 'bash',
        code: `curl -sS "{{BASE}}/v1/chat/completions" \\
  -H "content-type: application/json" \\
  -H "authorization: Bearer {{KEY}}" \\
  -d '{
    "model": "{{MODEL}}",
    "messages": [{"role": "user", "content": "Hello"}]
  }'`,
      },
      {
        titleKey: 'apiDocs.tools.curlResponses',
        lang: 'bash',
        code: `curl -sS "{{BASE}}/v1/responses" \\
  -H "content-type: application/json" \\
  -H "authorization: Bearer {{KEY}}" \\
  -d '{
    "model": "{{MODEL}}",
    "input": "Hello"
  }'`,
      },
      {
        titleKey: 'apiDocs.tools.curlEmbeddings',
        lang: 'bash',
        code: `curl -sS "{{BASE}}/v1/embeddings" \\
  -H "content-type: application/json" \\
  -H "authorization: Bearer {{KEY}}" \\
  -d '{
    "model": "voyage-4-large",
    "input": "Text to vectorize"
  }'`,
      },
      {
        titleKey: 'apiDocs.tools.curlRerank',
        lang: 'bash',
        code: `curl -sS "{{BASE}}/v1/rerank" \\
  -H "content-type: application/json" \\
  -H "authorization: Bearer {{KEY}}" \\
  -d '{
    "model": "rerank-2.5",
    "query": "query question",
    "documents": ["doc 1", "doc 2", "doc 3"]
  }'`,
      },
    ],
  },
  {
    id: 'sdk',
    labelKey: 'apiDocs.tools.sdk',
    icon: Code2,
    blocks: [
      {
        titleKey: 'apiDocs.tools.sdkEmbeddings',
        filename: 'python (voyage & openai)',
        lang: 'python',
        code: `import openai

client = openai.OpenAI(
    base_url="{{BASE}}/v1",
    api_key="{{KEY}}",
)

# 1. Embeddings (Voyage AI / OpenAI / Gemini)
emb = client.embeddings.create(
    model="voyage-4-large",
    input=["Hello world", "Artificial intelligence"],
)
print("Vector length:", len(emb.data[0].embedding))

# 2. Reranking (Voyage AI) via HTTP POST /v1/rerank
import requests
resp = requests.post(
    "{{BASE}}/v1/rerank",
    headers={"Authorization": "Bearer {{KEY}}"},
    json={
        "model": "rerank-2.5",
        "query": "What is AI?",
        "documents": ["AI is machine intelligence.", "Apples are fruits."],
    },
)
print("Rerank results:", resp.json())`,
      },
      {
        titleKey: 'apiDocs.tools.sdkPython',
        filename: 'python (anthropic)',
        lang: 'python',
        code: `from anthropic import Anthropic

client = Anthropic(
    base_url="{{BASE}}",
    api_key="{{KEY}}",
)

msg = client.messages.create(
    model="{{MODEL}}",
    max_tokens=1024,
    messages=[{"role": "user", "content": "Hello"}],
)
print(msg.content)`,
      },
      {
        titleKey: 'apiDocs.tools.sdkNode',
        filename: 'node',
        lang: 'ts',
        code: `import OpenAI from "openai";

const client = new OpenAI({
  baseURL: "{{BASE}}/v1",
  apiKey: "{{KEY}}",
});

const res = await client.chat.completions.create({
  model: "{{MODEL}}",
  messages: [{ role: "user", content: "Hello" }],
});
console.log(res.choices[0]?.message);`,
      },
    ],
  },
  {
    id: 'gui',
    labelKey: 'apiDocs.tools.gui',
    icon: MonitorSmartphone,
    blocks: [
      {
        titleKey: 'apiDocs.tools.guiTable',
        lang: 'text',
        code: `Cherry Studio / Chatbox / OpenWebUI (OpenAI-compatible)
  API Host / Base URL : {{BASE}}/v1
  API Key             : {{KEY}}
  Model               : {{MODEL}}  (or any id from GET /v1/models)

Cherry Studio (Anthropic provider)
  API Host            : {{BASE}}
  API Key             : {{KEY}}
  Model               : {{MODEL}}`,
        noteKey: 'apiDocs.tools.guiNote',
      },
    ],
  },
]
