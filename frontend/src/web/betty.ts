// The web build's Betty service: the same functions the Mac app's Go service exposes
// (betty.go), backed by the BerryTicket API with the browser's board session cookie.
// The app is the reference: View is assembled the same way, "new" tasks are tracked the
// same way (in localStorage instead of the app's prefs file), and the board is polled
// every 15 seconds. Things only a desktop app has (autostart, windows) are no-ops.
import type { View } from '../../bindings/github.com/wondde/betty/models'
import type * as berry from '../../bindings/github.com/wondde/betty/internal/berry/models'
import { Events } from './runtime'
import { BettyView } from './events'

const VERSION = __BETTY_VERSION__
const POLL_MS = 15_000

// ---------- what the Go side keeps in prefs ----------

type Board = {
  user: { role: string; id?: string; name: string; is_admin?: boolean }
  tickets: BoardTicket[]
  members: berry.Member[]
  projects: berry.Project[]
  proposals: berry.Proposal[]
  calendar: { meetings: berry.Meeting[]; proposals: berry.MeetingProposal[] }
  design_review: { questions: berry.DesignQuestion[] }
  github: berry.GitHub
  settings: Record<string, boolean>
  discord_channels: berry.DiscordChannel[]
  project_channels: berry.ProjectChannel[]
  progress: Record<string, berry.TaskProgress>
  design_links: Record<string, berry.DesignLink[]>
  trash: berry.TrashItem[]
  auto_assignments: Record<string, berry.AutoAssignment>
  long_term: Record<string, berry.LongTerm>
  weekly_review: berry.WeeklyReview
  revision: string
}
type BoardTicket = {
  id: string; number: number; project_id: string; title: string; description: string; status: string; priority: string
  assignee_id: string | null; due_date: string | null; source_url: string; area: string; created_at: string; updated_at: string
}
type Prefs = { member: string; seen: string[]; notified: string[]; order: string[]; hintDone: boolean; onboarded: boolean }

const PREFS = 'betty:prefs'
const blank = (): Prefs => ({ member: '', seen: [], notified: [], order: [], hintDone: false, onboarded: false })
function loadPrefs(): Prefs {
  try { return { ...blank(), ...JSON.parse(localStorage.getItem(PREFS) || '{}') } } catch { return blank() }
}
let prefs = loadPrefs()
const savePrefs = () => localStorage.setItem(PREFS, JSON.stringify(prefs))
const remember = (list: string[], ...ids: string[]) => {
  const out = [...list]
  for (const id of ids) if (!out.includes(id)) out.push(id)
  return out.slice(-1000)
}

let snapshot: Board | null = null
let signedIn = false
let offline = false
let syncedAt = ''
let started = false

// ---------- transport ----------

class APIError extends Error {
  constructor(public status: number, message: string) { super(message) }
}

async function api<T = unknown>(method: string, path: string, body?: unknown): Promise<T> {
  let res: Response
  try {
    res = await fetch(path, {
      method, credentials: 'same-origin',
      headers: body === undefined ? {} : { 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : JSON.stringify(body),
    })
  } catch {
    offline = true
    emit()
    throw new Error('서버에 연결하지 못했어요')
  }
  if (res.status === 401) {
    signedIn = false
    snapshot = null
    emit()
    throw new APIError(401, '로그인이 필요해요')
  }
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw new APIError(res.status, data.error || `HTTP ${res.status}`)
  return data as T
}

// ---------- view ----------

function colorProjects(projects: berry.Project[]) {
  const palette = ['#0a84ff', '#30a46c', '#f76b15', '#d6409f', '#12a594', '#e5484d', '#8e4ec6', '#c99a06', '#3e63dd', '#978365']
  const defaults = new Set(['#7564d9', '#6175ba'])
  const uses = new Map<string, number>()
  for (const p of projects) uses.set((p.color || '').toLowerCase(), (uses.get((p.color || '').toLowerCase()) ?? 0) + 1)
  const keep = (p: berry.Project) => { const c = (p.color || '').toLowerCase(); return c !== '' && uses.get(c) === 1 && !defaults.has(c) }
  const taken = new Set(projects.filter((p) => p.id !== 'berry-project-undecided' && keep(p)).map((p) => p.color.toLowerCase()))
  let next = 0
  for (const p of projects) {
    if (p.id === 'berry-project-undecided') { p.color = '#8b8d98'; continue }
    if (keep(p)) continue
    for (let n = 0; n < palette.length && taken.has(palette[next % palette.length]); n++) next++
    p.color = palette[next % palette.length]
    taken.add(p.color)
    next++
  }
}

const ticketOf = (t: BoardTicket, projects: berry.Project[]): berry.Ticket => ({
  id: t.id, number: t.number, title: t.title, description: t.description, status: t.status, priority: t.priority,
  due_date: t.due_date, source_url: t.source_url, updated_at: t.updated_at, project: projects.find((p) => p.id === t.project_id) ?? null,
  assignee_id: t.assignee_id, area: t.area, created_at: t.created_at,
})
const mine = (b: Board) => b.tickets.filter((t) => t.status !== 'done' && t.assignee_id === b.user.id)

function view(): View {
  const v: View = {
    signedIn, signingIn: false, member: null, isAdmin: false, tickets: [], members: [], projects: [], proposals: [], meetingProposals: [],
    designQuestions: [], meetings: [], github: { repos: [], items: [], suggestions: [], identities: [] }, settings: {}, discordChannels: [],
    projectChannels: [], progress: {}, designLinks: {}, trash: [], autoAssignments: {}, longTerm: {}, weeklyReview: { week: '', done_at: '', done_by: '', items: [] }, new: [], order: prefs.order, hintDone: prefs.hintDone,
    onboarded: prefs.onboarded, version: VERSION, offline, syncedAt, server: location.origin, update: null, updating: false,
  }
  const s = snapshot
  if (s && signedIn) {
    const me = s.members.find((m) => m.id === s.user.id)
    v.member = { id: s.user.id!, name: s.user.name, area: me?.area }
    v.isAdmin = !!s.user.is_admin
    v.tickets = s.tickets.map((t) => ticketOf(t, s.projects))
    v.members = s.members
    v.projects = s.projects
    v.proposals = s.proposals
    v.meetingProposals = s.calendar?.proposals ?? []
    v.meetings = s.calendar?.meetings ?? []
    v.designQuestions = s.design_review?.questions ?? []
    v.github = s.github ?? v.github
    v.settings = s.settings ?? {}
    v.discordChannels = s.discord_channels ?? []
    v.projectChannels = s.project_channels ?? []
    v.progress = s.progress ?? {}
    v.designLinks = s.design_links ?? {}
    v.trash = s.trash ?? []
    v.autoAssignments = s.auto_assignments ?? {}
    v.longTerm = s.long_term ?? {}
    v.weeklyReview = s.weekly_review ?? v.weeklyReview
    const seen = new Set(prefs.seen)
    v.new = mine(s).filter((t) => !seen.has(t.id)).map((t) => t.id)
  }
  return v
}

function emit() {
  Events.Emit(BettyView, view())
  const open = snapshot && signedIn ? mine(snapshot).length : 0
  document.title = open ? `(${open}) Betty` : 'Betty'
}

// ---------- sync ----------

function start() {
  if (started) return
  started = true
  let revision = ''
  const tick = async () => {
    if (!signedIn || document.hidden) return
    try {
      const r = await api<{ revision: string }>('GET', '/api/updates')
      offline = false
      if (r.revision !== (snapshot?.revision ?? revision)) { revision = r.revision; await Refresh() }
    } catch { /* offline or signed out: already reflected in the view */ }
  }
  setInterval(tick, POLL_MS)
  document.addEventListener('visibilitychange', () => { if (!document.hidden) tick() })
  // Desktop menu shortcuts that a browser lets a page keep.
  window.addEventListener('keydown', (e) => {
    const mod = e.metaKey || e.ctrlKey
    if (mod && !e.shiftKey && e.key.toLowerCase() === 'f') { e.preventDefault(); Events.Emit('betty:command', 'search') }
    else if (mod && e.key === ',') { e.preventDefault(); Events.Emit('betty:command', 'settings') }
    else if (mod && e.key.toLowerCase() === 'b' && !e.shiftKey) { e.preventDefault(); Events.Emit('betty:command', 'toggle-board') }
  })
}

export async function View(): Promise<View> {
  start()
  try { await Refresh() } catch { /* the view says signed out or offline */ }
  return view()
}

export async function Refresh(): Promise<void> {
  let board: Board
  try {
    board = await api<Board>('GET', '/api/state')
  } catch (e) {
    if (e instanceof APIError && e.status === 401) return
    throw e
  }
  if (!board.user?.id) {
    signedIn = false
    snapshot = null
    emit()
    throw new Error('팀원 계정으로 로그인해 주세요')
  }
  colorProjects(board.projects)
  signedIn = true
  offline = false
  syncedAt = new Date().toISOString()
  if (prefs.member !== board.user.id) prefs = { ...blank(), member: board.user.id, onboarded: prefs.onboarded, hintDone: prefs.hintDone }
  const open = mine(board).map((t) => t.id)
  const first = !snapshot && !prefs.seen.length
  let announce: BoardTicket[] = []
  if (first) {
    prefs.seen = remember(prefs.seen, ...open)
    prefs.notified = remember(prefs.notified, ...open)
  } else {
    const notified = new Set(prefs.notified)
    announce = mine(board).filter((t) => !notified.has(t.id))
    prefs.notified = remember(prefs.notified, ...announce.map((t) => t.id))
  }
  savePrefs()
  snapshot = board
  notify(announce)
  emit()
}

function notify(tickets: BoardTicket[]) {
  if (!tickets.length || typeof Notification === 'undefined' || Notification.permission !== 'granted') return
  const title = tickets.length > 1 ? `새 업무 ${tickets.length}건이 배정됐어요` : '새 업무가 배정됐어요'
  new Notification(title, { body: tickets.map((t) => `BT-${t.number} ${t.title}`).join('\n'), tag: 'betty-new-' + tickets[0].id })
}

// Calls that change the board and then reload it; a version conflict reloads first.
async function change(call: () => Promise<unknown>): Promise<void> {
  try {
    await call()
  } catch (e) {
    if (e instanceof APIError && e.status === 409) await Refresh().catch(() => {})
    throw e
  }
  await Refresh()
}

const boardTicket = (id: string) => {
  const t = snapshot?.tickets.find((x) => x.id === id)
  if (!t) throw new Error('업무를 찾지 못했어요. 새로고침해 주세요')
  return t
}
const enc = encodeURIComponent

// ---------- tasks ----------

export async function Update(id: string, changes: Record<string, unknown> | null): Promise<berry.Ticket> {
  const t = boardTicket(id)
  let updated: BoardTicket
  try {
    updated = await api<BoardTicket>('PATCH', `/api/tickets/${enc(id)}`, { ...changes, expected_updated_at: t.updated_at })
  } catch (e) {
    if (e instanceof APIError && e.status === 409) await Refresh().catch(() => {})
    throw e
  }
  prefs.seen = remember(prefs.seen, id)
  savePrefs()
  if (snapshot) snapshot.tickets = snapshot.tickets.map((x) => (x.id === id ? updated : x))
  emit()
  return ticketOf(updated, snapshot?.projects ?? [])
}

export const SetStatus = (id: string, status: string) => Update(id, { status })

export async function Claim(id: string): Promise<berry.Ticket> {
  if (!snapshot) throw new Error('새로고침한 뒤 다시 해 주세요')
  return Update(id, { assignee_id: snapshot.user.id })
}

export async function Create(input: berry.CreateInput): Promise<berry.Ticket> {
  const body = Object.fromEntries(Object.entries(input).filter(([, v]) => v !== undefined && v !== null && v !== ''))
  const created = await api<berry.Ticket>('POST', '/api/widget/tickets', body)
  prefs.seen = remember(prefs.seen, created.id)
  prefs.notified = remember(prefs.notified, created.id)
  savePrefs()
  Refresh().catch(() => {})
  return created
}

// ---------- review inbox, calendar ----------

const withProject = (projectID: string) => (projectID ? { project_id: projectID } : {})
export const ApproveProposal = (id: string, projectID: string) => change(() => api('POST', `/api/proposals/${enc(id)}/approve`, withProject(projectID)))
export const DismissProposal = (id: string) => change(() => api('POST', `/api/proposals/${enc(id)}/dismiss`, {}))
export const ApproveMeeting = (id: string, projectID: string) => change(() => api('POST', `/api/meeting-proposals/${enc(id)}/approve`, withProject(projectID)))
export const DismissMeeting = (id: string) => change(() => api('POST', `/api/meeting-proposals/${enc(id)}/dismiss`, {}))
export const DismissDesignQuestion = (id: string) => change(() => api('POST', `/api/design-review/questions/${enc(id)}/dismiss`, {}))

export function SaveMeeting(id: string, input: berry.MeetingInput): Promise<void> {
  const body = { ...input }
  if (id) body.expected_updated_at = snapshot?.calendar?.meetings?.find((m) => m.id === id)?.updated_at
  return change(() => (id ? api('PATCH', `/api/meetings/${enc(id)}`, body) : api('POST', '/api/meetings', body)))
}

// ---------- automation history ----------

async function history(cursor: string, status: string, ticketID: string): Promise<berry.History> {
  const q = new URLSearchParams({ limit: '30' })
  if (ticketID) q.set('ticket_id', ticketID)
  if (cursor) q.set('cursor', cursor)
  if (status) q.set('status', status)
  return api<berry.History>('GET', '/api/automation/history?' + q)
}
export const History = (cursor: string, status: string) => history(cursor, status, '')
export async function TaskHistory(ticketID: string): Promise<berry.Change[]> {
  return (await history('', 'all', ticketID)).items ?? []
}

export function Capture(content: string, projectID: string): Promise<void> {
  content = content.trim()
  if (!content) return Promise.reject(new Error('붙여 넣은 대화가 없어요'))
  if ([...content].length > 40000) return Promise.reject(new Error('대화가 너무 길어요. 4만 자 안으로 나눠 주세요'))
  return change(() => api('POST', '/api/sources', { content, project_id: projectID }))
}

export const Undo = (id: string) => change(() => api('POST', `/api/automation/history/${enc(id)}/undo`, {}))

// ---------- GitHub ----------

export function LinkGitHub(ticketID: string, repoID: string, number: number, allowShared: boolean, remove: boolean): Promise<void> {
  const t = boardTicket(ticketID)
  const itemVersion = snapshot?.github?.items?.find((i) => i.repo_id === repoID && i.number === number)?.updated_at ?? ''
  const body = { repo_id: repoID, number, expected_updated_at: t.updated_at, expected_github_updated_at: itemVersion, allow_shared: allowShared }
  return change(() => api(remove ? 'DELETE' : 'POST', `/api/tickets/${enc(ticketID)}/github-links`, body))
}
export function RejectGitHub(ticketID: string, repoID: string, number: number): Promise<void> {
  const t = boardTicket(ticketID)
  return change(() => api('POST', `/api/tickets/${enc(ticketID)}/github-links/reject`, { repo_id: repoID, number, expected_updated_at: t.updated_at }))
}
export const AddRepo = (fullName: string, projectID: string) => change(() => api('POST', '/api/github/repos', { full_name: fullName, project_id: projectID }))
export const RemoveRepo = (id: string) => change(() => api('DELETE', `/api/github/repos/${enc(id)}`, {}))

// ---------- team settings, trash, checklist ----------

export function SaveProject(id: string, name: string, channelIDs: string[] | null, color: string): Promise<void> {
  const body: Record<string, unknown> = { name, channel_ids: channelIDs ?? [] }
  if (color && id) body.color = color
  return change(() => (id ? api('PATCH', `/api/projects/${enc(id)}`, body) : api('POST', '/api/projects', body)))
}
export const SaveMember = (id: string, input: berry.MemberInput) => change(() => (id ? api('PATCH', `/api/members/${enc(id)}`, input) : api('POST', '/api/members', input)))
export const SetSetting = (key: string, on: boolean) => change(() => api('POST', '/api/settings', { [key]: on }))
export const Snooze = (id: string) => change(() => api('POST', `/api/tickets/${enc(id)}/snooze`, { days: 30 }))
export const FinishWeeklyReview = (decided: number) => change(() => api('POST', '/api/weekly-review/done', { decided }))
export const SetLongTerm = (id: string, long: boolean) => change(() => api('PATCH', `/api/tickets/${enc(id)}/horizon`, { horizon: long ? 'long' : 'quick' }))
export const SetAdmin = (memberID: string, on: boolean) => change(() => api('PATCH', `/api/members/${enc(memberID)}/admin`, { admin: on }))

export function Delete(id: string): Promise<void> {
  const t = boardTicket(id)
  return change(() => api('DELETE', `/api/tickets/${enc(id)}`, { expected_updated_at: t.updated_at }))
}
export const Restore = (id: string) => change(() => api('POST', `/api/tickets/${enc(id)}/restore`, {}))
export function EditChecklist(id: string, e: berry.ChecklistEdit): Promise<void> {
  const revision = snapshot?.progress?.[id]?.revision ?? ''
  return change(() => api('POST', `/api/tickets/${enc(id)}/checklist`, { ...e, expected_revision: revision }))
}

// ---------- local state ----------

export async function Acknowledge(id: string) { prefs.seen = remember(prefs.seen, id); savePrefs(); emit() }
export async function SaveOrder(ids: string[] | null) { prefs.order = ids ?? []; savePrefs() }
export async function DismissHint() { prefs.hintDone = true; savePrefs(); emit() }
export async function FinishOnboarding() { prefs.onboarded = true; prefs.hintDone = true; savePrefs(); emit() }

// ---------- account ----------

// Discord sign-in in this tab; the server sets the board session and comes back to "/".
export async function SignIn(): Promise<void> {
  const { enabled } = await api<{ enabled: boolean }>('GET', '/api/auth/discord')
  if (!enabled) throw new Error('Discord 로그인이 아직 준비되지 않았어요. 관리자에게 알려 주세요')
  location.href = '/auth/discord/start'
}
export async function CancelSignIn() {}
export async function SignOut(): Promise<void> {
  await api('POST', '/api/logout', {}).catch(() => {})
  signedIn = false
  snapshot = null
  emit()
}

// ---------- what only the desktop app has ----------

export async function Autostart() { return false }
export async function SetAutostart(_on: boolean) { throw new Error('브라우저에서는 쓸 수 없어요') }
export async function NotificationsAllowed() { return typeof Notification !== 'undefined' && Notification.permission === 'granted' }
export async function AllowNotifications() {
  if (typeof Notification === 'undefined') throw new Error('이 브라우저는 알림을 지원하지 않아요')
  return (await Notification.requestPermission()) === 'granted'
}
export async function OpenURL(link: string) {
  const u = new URL(link, location.href)
  if (!['http:', 'https:'].includes(u.protocol)) throw new Error('열 수 없는 주소예요')
  window.open(u.href, '_blank', 'noopener')
}
export async function OpenBoard() { window.open('/classic', '_blank', 'noopener') }
// The web is always the latest build; only the Mac app updates itself.
export async function CheckUpdate() { return null }
export async function InstallUpdate() {}
export async function ShowTasks() {}
export async function HideWindow(_name: string) {}
