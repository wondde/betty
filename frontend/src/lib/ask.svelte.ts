// In-app confirmation. The macOS webview has no window.confirm (it answers "cancel"
// without showing anything), so questions go through this dialog instead.
type Question = { message: string; ok: string; danger: boolean; resolve: (yes: boolean) => void }
export const asking = $state<{ current: Question | null }>({ current: null })

export function ask(message: string, ok = '확인', danger = false): Promise<boolean> {
  asking.current?.resolve(false)
  return new Promise((resolve) => (asking.current = { message, ok, danger, resolve }))
}

export function answer(yes: boolean) {
  const q = asking.current
  asking.current = null
  q?.resolve(yes)
}
