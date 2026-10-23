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
  | { kind: 'project'; id: string }
  | { kind: 'member'; id: string }

export const nav = $state<{ scope: Scope; selected: string | null; query: string; showDone: boolean }>({
  scope: { kind: 'mine' },
  selected: null,
  query: '',
  showDone: false,
})

export function go(scope: Scope) {
  nav.scope = scope
  nav.selected = null
}

export function sameScope(a: Scope, b: Scope): boolean {
  return a.kind === b.kind && ('id' in a ? a.id : '') === ('id' in b ? b.id : '')
}

// Views without a task list (no search, no add field, no detail pane).
export const isTaskList = (s: Scope) => !['inbox', 'calendar', 'teamStatus', 'history', 'github', 'admin'].includes(s.kind)
