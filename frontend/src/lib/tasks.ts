import type { GitHubItem, Ticket } from '../../bindings/github.com/wondde/betty/internal/berry'

export const statusLabel: Record<string, string> = { todo: '할 일', in_progress: '진행 중', blocked: '막힘', done: '완료' }

export function daysUntil(date?: string | null): number | null {
  if (!date) return null
  const [y, m, d] = date.split('-').map(Number)
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  return Math.round((new Date(y, m - 1, d).getTime() - today.getTime()) / 86400000)
}

export type Due = { label: string; tone: 'overdue' | 'today' | 'soon' | 'later' }
export function dueOf(t: Ticket): Due | null {
  const days = daysUntil(t.due_date)
  if (days === null || t.status === 'done') return null
  if (days < 0) return { label: `${-days}일 지남`, tone: 'overdue' }
  if (days === 0) return { label: '오늘 마감', tone: 'today' }
  if (days === 1) return { label: '내일 마감', tone: 'soon' }
  const [, m, d] = t.due_date!.split('-')
  return { label: `~${Number(m)}/${Number(d)}`, tone: 'later' }
}

// Urgent first: overdue, today, tomorrow, newly assigned, in progress, the rest.
// Within a level the user's dragged order wins, then nearest deadline, then newest.
export function sortTasks(tickets: Ticket[], isNew: Set<string>, order: string[]): Ticket[] {
  const level = (t: Ticket) => {
    const days = daysUntil(t.due_date)
    if (days !== null && days < 0) return 0
    if (days === 0) return 1
    if (days === 1) return 2
    if (isNew.has(t.id)) return 2.5
    if (t.status === 'in_progress') return 3
    if (t.status === 'blocked') return 3.5
    return 4
  }
  const rank = (id: string) => order.indexOf(id)
  const due = (t: Ticket) => daysUntil(t.due_date) ?? 9999
  return tickets.slice().sort((a, b) => level(a) - level(b) || rank(a.id) - rank(b.id) || due(a) - due(b) || b.number - a.number)
}

export function isUrgent(t: Ticket): boolean {
  const days = daysUntil(t.due_date)
  return days !== null && days <= 1
}

export function todayLabel(): string {
  const d = new Date()
  return `${d.getMonth() + 1}월 ${d.getDate()}일 ${'일월화수목금토'[d.getDay()]}요일`
}

export function dayGroup(iso: string): string {
  const d = new Date(iso)
  const start = (x: Date) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime()
  const diff = Math.round((start(new Date()) - start(d)) / 86400000)
  if (diff === 0) return '오늘'
  if (diff === 1) return '어제'
  return `${d.getMonth() + 1}월 ${d.getDate()}일`
}

export const priorityLabel: Record<string, string> = { low: '낮음', normal: '보통', high: '높음', urgent: '긴급' }
export const areaLabel: Record<string, string> = { '': '정하지 않음', engineering: '개발', design: '디자인', marketing: '마케팅' }

type Named = { id: string; name: string }

// Search across BT number, title, description, assignee and project names.
export function matches(t: Ticket, query: string, members: Named[]): boolean {
  const q = query.trim().toLowerCase().replace(/\s+/g, ' ')
  if (!q) return true
  const who = members.find((m) => m.id === t.assignee_id)?.name ?? ''
  const hay = `bt-${t.number} bt${t.number} ${t.title} ${t.description} ${who} ${t.project?.name ?? ''}`.toLowerCase()
  return q.split(' ').every((word) => hay.includes(word))
}

const statusOrder: Record<string, number> = { in_progress: 0, blocked: 1, todo: 2, done: 3 }
const statusTitle: Record<string, string> = { in_progress: '진행 중', blocked: '막힘', todo: '할 일', done: '완료' }

// Team-style grouping: by status, nearest deadline first inside each group.
export function groupByStatus(tickets: Ticket[]): { title: string; items: Ticket[] }[] {
  const sorted = tickets.slice().sort((a, b) =>
    statusOrder[a.status] - statusOrder[b.status] || (daysUntil(a.due_date) ?? 9999) - (daysUntil(b.due_date) ?? 9999) || b.number - a.number)
  const out: { title: string; items: Ticket[] }[] = []
  for (const t of sorted) {
    const title = statusTitle[t.status] ?? t.status
    if (out.at(-1)?.title !== title) out.push({ title, items: [] })
    out.at(-1)!.items.push(t)
  }
  return out
}

// Initial for the little avatar dot next to a task.
export function initial(name?: string): string {
  return name ? Array.from(name)[0] : ''
}

// GitHub item stage in words, plus a tone for the badge colour.
export function stageLabel(i: GitHubItem): [string, string] {
  if (i.kind === 'pr' && i.stage === 'merged') return ['병합됨', 'merged']
  if (i.kind === 'pr' && i.stage === 'closed') return ['병합 없이 닫힘', 'closed']
  if (i.kind === 'issue' && i.state === 'closed') return i.state_reason === 'completed' ? ['완료됨', 'merged'] : ['하지 않기로 함', 'closed']
  return ({ draft: ['준비 중', 'draft'], review_requested: ['리뷰 요청', 'review'] } as Record<string, [string, string]>)[i.stage] ?? ['진행 중', 'open']
}
