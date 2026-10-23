// What the main window shows: sidebar scope, selected task, search and the done toggle.
export type Scope =
  | { kind: 'mine' }
  | { kind: 'team' }
  | { kind: 'board' }
  | { kind: 'unassigned' }
  | { kind: 'inbox' }
  | { kind: 'calendar' }
  | { kind: 'teamStatus' }
  | { kind: 'history' }
  | { kind: 'github' }
  | { kind: 'admin' }
  | { kind: 'trash' }
  | { kind: 'project'; id: string }
  | { kind: 'member'; id: string }

// project narrows the team-wide lists and the board ('' = all projects).
export const nav = $state<{ scope: Scope; selected: string | null; query: string; showDone: boolean; project: string }>({
  scope: { kind: 'mine' },
  selected: null,
  query: '',
  showDone: false,
  project: '',
})

export function go(scope: Scope) {
  nav.scope = scope
  nav.selected = null
}

export function sameScope(a: Scope, b: Scope): boolean {
  return a.kind === b.kind && ('id' in a ? a.id : '') === ('id' in b ? b.id : '')
}

// Team-wide task views that get the project chips.
export const byProject = (s: Scope) => ['team', 'unassigned', 'member', 'board'].includes(s.kind)

// Views without a task list (no search, no add field, no detail pane).
export const isTaskList = (s: Scope) => !['inbox', 'calendar', 'teamStatus', 'history', 'github', 'admin', 'trash'].includes(s.kind)
