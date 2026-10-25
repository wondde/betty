// Where the shared UI runs: the Mac app, the Windows app, or a browser tab (web build).
export const isWeb = import.meta.env.MODE === 'web'
export const isMac = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.platform || navigator.userAgent)
// Shortcut labels for that platform.
export const mod = isMac ? '⌘' : 'Ctrl+'
// Only the Mac app draws its traffic-light buttons inside the window.
export const shell = isWeb ? 'web' : isMac ? 'mac' : 'desktop'
if (typeof document !== 'undefined') document.documentElement.dataset.shell = shell
