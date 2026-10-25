// Shortcut labels for the platform the shared UI runs on (Mac app, Windows app, browser).
export const isMac = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.platform || navigator.userAgent)
export const mod = isMac ? '⌘' : 'Ctrl+'
