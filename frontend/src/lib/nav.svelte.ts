// What the main window shows: sidebar scope, selected task, search and the done toggle.
// The sidebar has four places (내 할 일 · 팀 할 일 · 베티가 찾은 일 · 캘린더), projects and the
// team's areas; everything administrative lives in Settings (⌘,).
export type Area = 'engineering' | 'design' | 'marketing' | 'none'
export type Scope =
  | { kind: 'mine' }
  | { kind: 'team' }
  | { kind: 'inbox' }
  | { kind: 'calendar' }
  | { kind: 'area'; id: Area }
  | { kind: 'project'; id: string }
  | { kind: 'member'; id: string }

// project narrows team-wide lists ('' = all); unassigned shows only work nobody took;
// longTerm shows only long-term work;
// board switches 팀 할 일 between the list and status columns.
export const nav = $state<{ scope: Scope; selected: string | null; query: string; showDone: boolean; project: string; unassigned: boolean; longTerm: boolean; board: boolean }>({
  scope: { kind: 'mine' },
  selected: null,
  query: '',
  showDone: false,
  project: '',
  unassigned: false,
  longTerm: false,
  board: false,
})

export function go(scope: Scope) {
  nav.scope = scope
  nav.selected = null
}

export function sameScope(a: Scope, b: Scope): boolean {
  return a.kind === b.kind && ('id' in a ? a.id : '') === ('id' in b ? b.id : '')
}

export const areaName: Record<Area, string> = { engineering: '개발', design: '디자인', marketing: '마케팅', none: '영역 미정' }
export const memberArea = (m: { area?: string }): Area => (['engineering', 'design', 'marketing'].includes(m.area ?? '') ? (m.area as Area) : 'none')

// Team-wide task views that get the project chips.
export const byProject = (s: Scope) => ['team', 'member', 'area'].includes(s.kind)

// Views without a task list (no search, no add field, no detail pane).
export const isTaskList = (s: Scope) => !['inbox', 'calendar'].includes(s.kind)
