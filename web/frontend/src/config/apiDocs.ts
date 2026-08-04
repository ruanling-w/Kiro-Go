// apiDocs.ts — static catalog for the API docs page: endpoint table rows +
// CLI/SDK/GUI snippet templates. Placeholders {{BASE}} {{KEY}} {{MODEL}}
// {{MODEL_OPUS}} {{MODEL_HAIKU}} are filled at render time by fillSnippet.
import {
  Terminal,
  Box,
  Code2,
  MonitorSmartphone,
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
  id: 'claude-code' | 'codex' | 'curl' | 'sdk' | 'gui'
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
        code: `# Kiro-Go Configuration for Codex CLI
model = "{{MODEL}}"
model_provider = "kiro-go"

[model_providers.kiro-go]
name = "Kiro-Go"
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
    ],
  },
  {
    id: 'sdk',
    labelKey: 'apiDocs.tools.sdk',
    icon: Code2,
    blocks: [
      {
        titleKey: 'apiDocs.tools.sdkPython',
        filename: 'python',
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
