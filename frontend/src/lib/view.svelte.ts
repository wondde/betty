// One shared, reactive copy of the Go-side View. The Betty service pushes "betty:view"
// whenever anything changes, so the menu bar window and quick add stay in sync.
import { Events } from '@wailsio/runtime'
import { Betty, type View } from '../../bindings/github.com/wondde/betty'
import { BettyView } from '../../bindings/events'

const empty: View = { signedIn: false, signingIn: false, member: null, isAdmin: false, tickets: [], members: [], projects: [], proposals: [], meetingProposals: [], designQuestions: [], meetings: [], github: { repos: [], items: [], suggestions: [], identities: [] }, settings: {}, discordChannels: [], projectChannels: [], progress: {}, designLinks: {}, trash: [], autoAssignments: {}, longTerm: {}, new: [], order: [], hintDone: false, onboarded: false, version: '', offline: false, syncedAt: '', server: '' }
export const app = $state<{ view: View; ready: boolean }>({ view: empty, ready: false })

export function connect() {
  Betty.View().then((v) => {
    app.view = v
    app.ready = true
  })
  return Events.On(BettyView, (e) => {
    app.view = e.data as View
    app.ready = true
  })
}

// Readable message from a rejected binding call.
export function message(err: unknown): string {
  if (err && typeof err === 'object' && 'message' in err) return String((err as { message: unknown }).message)
  return String(err)
}

// Admins can switch to the screen everyone else sees (kept on this device). The server still
// decides what an admin may change; this only hides team setup while looking as a member.
export const viewMode = $state({ admin: localStorage.getItem('view:member') !== '1' })
export const asAdmin = () => app.view.isAdmin && viewMode.admin
export function setAdminView(on: boolean) {
  viewMode.admin = on
  localStorage.setItem('view:member', on ? '0' : '1')
}
