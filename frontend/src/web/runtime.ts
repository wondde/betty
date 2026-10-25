// Stand-in for @wailsio/runtime in the web build: an in-page event bus with the same
// Events.On/Emit shape, and no-op window controls (a browser tab has no app windows).
type Handler = (e: { name: string; data: unknown }) => void
const handlers = new Map<string, Set<Handler>>()

export const Events = {
  On(name: string, cb: Handler): () => void {
    if (!handlers.has(name)) handlers.set(name, new Set())
    handlers.get(name)!.add(cb)
    return () => handlers.get(name)?.delete(cb)
  },
  Emit(name: string, data?: unknown) {
    for (const cb of handlers.get(name) ?? []) cb({ name, data })
  },
}

export const Window = {
  Hide: async () => {},
  Show: async () => {},
  Close: async () => {},
  SetSize: async (_w: number, _h: number) => {},
  Center: async () => {},
}
