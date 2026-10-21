import type { Ticket } from '../../bindings/github.com/wondde/betty/internal/berry'

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
