<script lang="ts">
  // 보드 (⌘9): the team's tasks in status columns. Drag a card to another column to change
  // its status; click it to open the detail pane. Finished work shows the last two weeks
  // unless "완료 포함" is on.
  import { Betty } from '../../bindings/github.com/wondde/betty'
  import type { Ticket } from '../../bindings/github.com/wondde/betty/internal/berry'
  import { app, message } from '../lib/view.svelte'
  import { nav } from '../lib/nav.svelte'
  import { dueOf, initial } from '../lib/tasks'

  let { tickets, onselect, onerror }: { tickets: Ticket[]; onselect: (t: Ticket) => void; onerror: (m: string) => void } = $props()

  let who = $state<'all' | 'mine' | 'unassigned'>('all')
  // Pointer-driven drag (HTML5 drag-and-drop is unreliable in the macOS webview).
  let dragging = $state<string | null>(null)
  let over = $state<string | null>(null)
  let ghost = $state<{ x: number; y: number; title: string } | null>(null)
  let press: { id: string; x: number; y: number; title: string } | null = null
  let dragged = false
  let moving = $state<Set<string>>(new Set())

  const me = $derived(app.view.member?.id)
  const columns = [
    { status: 'todo', label: '할 일' },
    { status: 'in_progress', label: '진행 중' },
    { status: 'blocked', label: '막힘' },
    { status: 'done', label: '완료' },
  ]
  const recent = new Date(Date.now() - 14 * 86400000).toISOString()

  const shown = $derived(
    tickets.filter(
      (t) =>
        (who === 'all' || (who === 'mine' ? t.assignee_id === me : !t.assignee_id)) &&
        (t.status !== 'done' || nav.showDone || t.updated_at >= recent),
    ),
  )
  const byStatus = $derived(
    Object.fromEntries(
      columns.map((c) => [
        c.status,
        shown
          .filter((t) => (t.status || 'todo') === c.status)
          .sort((a, b) => (c.status === 'done' ? b.updated_at.localeCompare(a.updated_at) : (a.due_date ?? '9999').localeCompare(b.due_date ?? '9999') || b.number - a.number)),
      ]),
    ) as Record<string, Ticket[]>,
  )
  const name = (id?: string | null) => app.view.members?.find((m) => m.id === id)?.name

  async function move(id: string, status: string) {
    const t = tickets.find((x) => x.id === id)
    if (!t || t.status === status) return
    moving = new Set([...moving, id])
    try {
      await Betty.SetStatus(id, status)
    } catch (e) {
      onerror(message(e))
    } finally {
      moving = new Set([...moving].filter((x) => x !== id))
    }
  }

  const columnAt = (x: number, y: number) => (document.elementFromPoint(x, y)?.closest('[data-status]') as HTMLElement | null)?.dataset.status ?? null

  function down(e: PointerEvent, t: Ticket) {
    if (e.button !== 0) return
    press = { id: t.id, x: e.clientX, y: e.clientY, title: t.title }
    dragged = false
  }
  function pointermove(e: PointerEvent) {
    if (!press) return
    if (!dragging && Math.hypot(e.clientX - press.x, e.clientY - press.y) < 5) return
    dragging = press.id
    dragged = true
    ghost = { x: e.clientX, y: e.clientY, title: press.title }
    over = columnAt(e.clientX, e.clientY)
  }
  function pointerup(e: PointerEvent) {
    const id = dragging, status = id ? columnAt(e.clientX, e.clientY) : null
    press = null
    dragging = null
    ghost = null
    over = null
    if (id && status) move(id, status)
  }
  function open(t: Ticket) {
    if (dragged) return void (dragged = false) // the click that ends a drag doesn't select
    onselect(t)
  }
</script>

<svelte:window onpointermove={pointermove} onpointerup={pointerup} onkeydown={(e) => e.key === 'Escape' && dragging && ((press = null), (dragging = null), (ghost = null), (over = null))} />

<div class="filters">
  <div class="chips" role="radiogroup" aria-label="담당">
    {#each [['all', '전체'], ['mine', '내 일'], ['unassigned', '담당자 없음']] as [value, label]}
      <button role="radio" aria-checked={who === value} class:on={who === value} onclick={() => (who = value as typeof who)}>{label}</button>
    {/each}
  </div>
</div>

<div class="board" class:lifting={dragging}>
  {#each columns as c (c.status)}
    <section
      class="column {c.status}"
      class:over={over === c.status && dragging}
      aria-label={c.label}
      data-status={c.status}
    >
      <h2>{c.label} <span>{byStatus[c.status].length}</span></h2>
      <div class="cards">
        {#each byStatus[c.status] as t (t.id)}
          {@const due = dueOf(t)}
          <button
            class="card"
            class:selected={nav.selected === t.id}
            class:dragging={dragging === t.id}
            class:moving={moving.has(t.id)}
            data-task={t.id}
            onpointerdown={(e) => down(e, t)}
            onclick={() => open(t)}
          >
            <span class="title">{t.title}</span>
            <span class="meta">
              {#if t.project}<i class="dot" style:background={t.project.color} aria-hidden="true"></i><span class="project">{t.project.name}</span>{/if}
              {#if due && t.status !== 'done'}<span class="due {due.tone}">{due.label}</span>{/if}
              <span class="spacer"></span>
              <span class="num">BT-{t.number}</span>
              {#if t.assignee_id}<span class="avatar" title={name(t.assignee_id)}>{initial(name(t.assignee_id))}</span>{/if}
            </span>
          </button>
        {:else}
          <p class="empty">{c.status === 'done' ? '최근 2주 완료 없음' : '없음'}</p>
        {/each}
      </div>
    </section>
  {/each}
</div>

{#if ghost}<div class="ghost" style:left="{ghost.x + 8}px" style:top="{ghost.y + 8}px" aria-hidden="true">{ghost.title}</div>{/if}

<style>
  .filters { display: flex; align-items: center; gap: 8px; padding: 2px 8px 8px; }
  .chips { display: flex; gap: 2px; padding: 2px; border-radius: 8px; background: var(--fill); }
  .chips button { height: 24px; padding: 0 10px; border-radius: 6px; font-size: 12px; font-weight: 600; color: var(--fg2); }
  .chips button.on { background: var(--card-solid, #fff); color: var(--fg); box-shadow: 0 1px 2px rgba(0,0,0,0.12); }
  @media (prefers-color-scheme: dark) { .chips button.on { --card-solid: #48484a; } }
  .board { display: grid; grid-template-columns: repeat(4, minmax(190px, 1fr)); gap: 8px; min-height: calc(100% - 44px); overflow-x: auto; padding: 0 2px 4px; }
  .column { display: flex; flex-direction: column; min-width: 0; border-radius: 10px; background: var(--fill); transition: box-shadow 0.12s ease; }
  .column.over { box-shadow: inset 0 0 0 2px var(--accent); }
  h2 { margin: 0; padding: 9px 10px 6px; font-size: 11.5px; font-weight: 700; color: var(--fg2); }
  h2 span { font-weight: 500; color: var(--fg3); margin-left: 2px; }
  .in_progress h2 { color: var(--accent); }
  .blocked h2 { color: var(--red); }
  .done h2 { color: var(--green); }
  .cards { display: flex; flex-direction: column; gap: 6px; padding: 0 6px 8px; }
  .card {
    display: flex; flex-direction: column; gap: 6px; padding: 8px 9px; border-radius: 8px; text-align: left; cursor: default;
    background: var(--card-solid, #fff); box-shadow: 0 0.5px 1.5px rgba(0,0,0,0.14);
  }
  @media (prefers-color-scheme: dark) { .card { --card-solid: #3a3a3c; } }
  .card:hover { box-shadow: 0 1px 4px rgba(0,0,0,0.18); }
  .card.selected { box-shadow: 0 0 0 2px var(--accent); }
  .card.dragging { opacity: 0.4; }
  .card.moving { opacity: 0.6; }
  .done .card .title { color: var(--fg3); text-decoration: line-through; }
  .title { font-size: 12.5px; font-weight: 600; line-height: 1.4; display: -webkit-box; -webkit-line-clamp: 3; line-clamp: 3; -webkit-box-orient: vertical; overflow: hidden; }
  .meta { display: flex; align-items: center; gap: 5px; min-width: 0; font-size: 11px; color: var(--fg3); }
  .dot { width: 7px; height: 7px; border-radius: 50%; flex: none; }
  .project { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .due { flex: none; font-weight: 600; }
  .due.overdue, .due.today { color: var(--red); }
  .due.soon { color: var(--orange); }
  .spacer { flex: 1; }
  .num { flex: none; font-variant-numeric: tabular-nums; }
  .avatar { flex: none; width: 18px; height: 18px; border-radius: 50%; background: var(--fill-hover); display: grid; place-items: center; font-size: 10px; font-weight: 700; color: var(--fg2); }
  .board.lifting, .board.lifting * { cursor: grabbing; }
  .ghost { position: fixed; z-index: 20; max-width: 220px; padding: 7px 10px; border-radius: 8px; pointer-events: none; font-size: 12.5px; font-weight: 600; background: var(--card-solid, #fff); box-shadow: 0 8px 24px rgba(0,0,0,0.25); transform: rotate(-1.5deg); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  @media (prefers-color-scheme: dark) { .ghost { --card-solid: #3a3a3c; } }
  .empty { margin: 4px 4px 0; font-size: 11.5px; color: var(--fg3); }
</style>
