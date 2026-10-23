<script lang="ts">
  // One task row. Reminders-style circle to complete (short undo window), ▶/❚❚ to start or
  // pause; clicking the row selects it and opens the detail pane.
  import { Betty } from '../../bindings/github.com/wondde/betty'
  import type { Ticket } from '../../bindings/github.com/wondde/betty/internal/berry'
  import { dueOf, statusLabel, initial } from '../lib/tasks'
  import { app, message } from '../lib/view.svelte'

  let {
    ticket,
    isNew = false,
    selected = false,
    showAssignee = false,
    showProject = true,
    showStatus = true,
    onselect,
    onerror,
  }: { ticket: Ticket; isNew?: boolean; selected?: boolean; showAssignee?: boolean; showProject?: boolean; showStatus?: boolean; onselect: () => void; onerror: (m: string) => void } = $props()

  let finishing = $state(false)
  let busy = $state(false)
  let timer: ReturnType<typeof setTimeout> | undefined

  const due = $derived(dueOf(ticket))
  const done = $derived(ticket.status === 'done')
  const working = $derived(ticket.status === 'in_progress')
  const assignee = $derived(app.view.members?.find((m) => m.id === ticket.assignee_id))
  const guesses = $derived((app.view.github?.suggestions ?? []).filter((s) => s.ticket_id === ticket.id).length)
  const checklist = $derived.by(() => {
    const p = app.view.progress?.[ticket.id]
    const done = p?.finished?.length ?? 0
    return { done, total: done + (p?.remaining?.length ?? 0) }
  })

  async function set(status: string) {
    busy = true
    try {
      await Betty.SetStatus(ticket.id, status)
    } catch (e) {
      onerror(message(e))
    } finally {
      busy = false
    }
  }

  // Completing waits 1.6s so a mis-click can be undone by clicking the circle again.
  function toggleDone(e: MouseEvent) {
    e.stopPropagation()
    if (done) return void set('todo')
    if (finishing) {
      clearTimeout(timer)
      finishing = false
      return
    }
    finishing = true
    timer = setTimeout(async () => {
      await set('done')
      finishing = false
    }, 1600)
  }
</script>

<article class="row" class:selected class:finishing class:done class:isNew>
  <button class="check" class:on={finishing || done} onclick={toggleDone} aria-label={done ? '다시 할 일로' : finishing ? '완료 취소' : `완료: ${ticket.title}`} title={done ? '다시 할 일로' : finishing ? '되돌리기' : '완료'}>
    <svg viewBox="0 0 20 20" aria-hidden="true"><circle cx="10" cy="10" r="8.5" /><path d="M6 10.4l2.7 2.6L14.2 7.4" /></svg>
  </button>

  <div class="body" role="button" tabindex="-1" onclick={onselect} onkeydown={(e) => e.key === 'Enter' && onselect()}>
    <div class="title">{ticket.title}</div>
    <div class="meta">
      <span class="num">BT-{ticket.number}</span>
      {#if isNew}<span class="new">NEW</span>{/if}
      {#if showProject && ticket.project}<span class="project" style:--pc={ticket.project.color}>{ticket.project.name}</span>{/if}
      {#if showStatus && (working || ticket.status === 'blocked')}<span class="status {ticket.status}">{statusLabel[ticket.status]}</span>{/if}
      {#if ticket.priority === 'urgent' || ticket.priority === 'high'}<span class="prio {ticket.priority}">{ticket.priority === 'urgent' ? '긴급' : '중요'}</span>{/if}
      {#if due}<span class="due {due.tone}">{due.label}</span>{/if}
      {#if guesses}<span class="guess" title="베티가 같은 업무 같다고 본 GitHub 항목이 있어요. 상세에서 확인해 주세요">PR 확인 {guesses}</span>{/if}
      {#if checklist.total}<span class="checks" class:all={checklist.done === checklist.total}>☑ {checklist.done}/{checklist.total}</span>{/if}
    </div>
  </div>

  {#if showAssignee}
    <span class="who" title={assignee?.name ?? '담당자 없음'} class:none={!assignee}>{assignee ? initial(assignee.name) : ''}</span>
  {/if}
  {#if !done && ticket.status !== 'blocked'}
    <button class="quick" class:working onclick={(e) => (e.stopPropagation(), set(working ? 'todo' : 'in_progress'))} disabled={busy} title={working ? '할 일로 되돌리기' : '시작하기'} aria-label={working ? '할 일로 되돌리기' : '시작하기'}>
      {#if working}
        <svg viewBox="0 0 12 12" aria-hidden="true"><rect x="2.5" y="2" width="2.4" height="8" rx="0.6" /><rect x="7.1" y="2" width="2.4" height="8" rx="0.6" /></svg>
      {:else}
        <svg viewBox="0 0 12 12" aria-hidden="true"><path d="M3.5 2.2v7.6L10 6z" /></svg>
      {/if}
    </button>
  {/if}
</article>

<style>
  .row { display: flex; align-items: center; gap: 10px; padding: 8px 10px; border-radius: 9px; }
  .row:hover { background: var(--fill); }
  .row.selected { background: color-mix(in srgb, var(--accent) 16%, transparent); }
  .row.isNew .title::before { content: ''; display: inline-block; width: 7px; height: 7px; margin: 0 6px 1px 0; border-radius: 50%; background: var(--accent); }
  .row.finishing, .row.done { opacity: 0.55; }
  .row.finishing .title, .row.done .title { text-decoration: line-through; }
  .check { width: 20px; height: 20px; flex: none; }
  .check svg { width: 20px; height: 20px; fill: none; stroke: var(--fg3); stroke-width: 1.4; }
  .check path { stroke: transparent; stroke-width: 1.8; stroke-linecap: round; stroke-linejoin: round; }
  .check:hover circle { stroke: var(--accent); }
  .check.on circle { fill: var(--accent); stroke: var(--accent); }
  .check.on path { stroke: #fff; }
  .body { flex: 1; min-width: 0; outline: none; cursor: default; }
  .title { font-size: 13px; font-weight: 500; line-height: 1.35; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .meta { display: flex; flex-wrap: wrap; align-items: center; gap: 2px 8px; margin-top: 2px; font-size: 11px; color: var(--fg3); }
  .num { font-variant-numeric: tabular-nums; font-weight: 600; }
  .new { padding: 0 5px; border-radius: 4px; background: var(--accent); color: #fff; font-size: 9.5px; font-weight: 800; }
  .project { padding: 0 6px; border-radius: 5px; font-weight: 600; color: color-mix(in srgb, var(--pc) 80%, var(--fg)); background: color-mix(in srgb, var(--pc) 16%, transparent); }
  .status { padding: 0 6px; border-radius: 5px; font-weight: 600; }
  .status.in_progress { color: var(--accent); background: color-mix(in srgb, var(--accent) 14%, transparent); }
  .status.blocked { color: var(--red); background: color-mix(in srgb, var(--red) 13%, transparent); }
  .prio { font-weight: 700; }
  .prio.urgent { color: var(--red); }
  .prio.high { color: var(--orange); }
  .due { font-weight: 600; }
  .checks { font-variant-numeric: tabular-nums; }
  .guess { font-weight: 600; color: var(--orange); }
  .checks.all { color: var(--green); }
  .due.overdue, .due.today { color: var(--red); }
  .due.soon { color: var(--orange); }
  .who { width: 22px; height: 22px; flex: none; border-radius: 50%; display: grid; place-items: center; font-size: 10.5px; font-weight: 700; background: var(--fill-hover); color: var(--fg2); }
  .who.none { background: transparent; }
  .quick { width: 24px; height: 24px; flex: none; border-radius: 50%; display: grid; place-items: center; background: var(--fill); opacity: 0; }
  .quick svg { width: 10px; height: 10px; fill: var(--fg2); }
  .row:hover .quick, .quick:focus-visible { opacity: 1; }
  .quick.working svg { fill: var(--accent); }
  .quick:hover { background: var(--fill-hover); }
  button:disabled { opacity: 0.5; }
</style>
