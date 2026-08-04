// Add-account flow registry. Each entry maps a method id to its flow component +
// display metadata. AddAccountDialog is just a picker over this list, so adding a
// provider later = one entry here + its flow file, no dialog edits.
import type { ComponentType } from 'react'
import {
  Boxes,
  Sparkles,
  Bot,
  Terminal,
  Cloud,
  KeyRound,
  Fingerprint,
  ShieldCheck,
  FileKey,
  ClipboardPaste,
  HardDrive,
  Cookie,
  type LucideIcon,
} from 'lucide-react'
import type { FlowComponentProps } from './types'
import { GrokFlow } from './GrokFlow'
import { AntigravityFlow } from './AntigravityFlow'
import { CodexFlow } from './CodexFlow'
import { BuilderIdFlow } from './BuilderIdFlow'
import { IamSsoFlow } from './IamSsoFlow'
import { KiroSsoFlow } from './KiroSsoFlow'
import { LocalCacheFlow } from './LocalCacheFlow'
import { WebCookieFlow } from './WebCookieFlow'
import {
  KiroApiKeyFlow,
  RemoteKiroFlow,
  SsoTokenFlow,
  CredentialsFlow,
  CodexImportFlow,
} from './ImportFlows'

export interface FlowEntry {
  id: string
  labelKey: string
  icon: LucideIcon
  /** Which provider bucket this flow belongs under (for grouping in the picker). */
  group: 'kiro' | 'antigravity' | 'grok' | 'codex' | 'remotekiro'
  Component: ComponentType<FlowComponentProps>
}

export const FLOW_ENTRIES: FlowEntry[] = [
  // Kiro
  { id: 'kiro-sso', labelKey: 'addAccount.kiroSso', icon: ShieldCheck, group: 'kiro', Component: KiroSsoFlow },
  { id: 'builderid', labelKey: 'addAccount.builderId', icon: Fingerprint, group: 'kiro', Component: BuilderIdFlow },
  { id: 'iam-sso', labelKey: 'addAccount.iamSso', icon: ShieldCheck, group: 'kiro', Component: IamSsoFlow },
  { id: 'kiro-apikey', labelKey: 'addAccount.kiroApiKey', icon: KeyRound, group: 'kiro', Component: KiroApiKeyFlow },
  { id: 'sso-token', labelKey: 'addAccount.ssoToken', icon: FileKey, group: 'kiro', Component: SsoTokenFlow },
  { id: 'credentials', labelKey: 'addAccount.credentials', icon: ClipboardPaste, group: 'kiro', Component: CredentialsFlow },
  { id: 'local', labelKey: 'modal.localTitle', icon: HardDrive, group: 'kiro', Component: LocalCacheFlow },
  { id: 'cookie', labelKey: 'modal.cookieTitle', icon: Cookie, group: 'kiro', Component: WebCookieFlow },
  // Antigravity
  { id: 'antigravity', labelKey: 'addAccount.antigravity', icon: Sparkles, group: 'antigravity', Component: AntigravityFlow },
  // Grok
  { id: 'grok', labelKey: 'addAccount.grok', icon: Bot, group: 'grok', Component: GrokFlow },
  // Codex
  { id: 'codex', labelKey: 'addAccount.codex', icon: Terminal, group: 'codex', Component: CodexFlow },
  { id: 'codex-import', labelKey: 'addAccount.codexImport', icon: FileKey, group: 'codex', Component: CodexImportFlow },
  // Remote Kiro
  { id: 'remote-kiro', labelKey: 'addAccount.remoteKiro', icon: Cloud, group: 'remotekiro', Component: RemoteKiroFlow },
]

export const GROUP_ICON: Record<FlowEntry['group'], LucideIcon> = {
  kiro: Boxes,
  antigravity: Sparkles,
  grok: Bot,
  codex: Terminal,
  remotekiro: Cloud,
}
