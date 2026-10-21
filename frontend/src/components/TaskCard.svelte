<script lang="ts">
  // One task. Reminders-style circle to complete (with a short undo window),
  // ▶/❚❚ to start or pause, click to open details and actions.
  import { Betty } from '../../bindings/github.com/wondde/betty'
  import type { Ticket } from '../../bindings/github.com/wondde/betty/internal/berry'
  import { dueOf, statusLabel } from '../lib/tasks'
  import { message } from '../lib/view.svelte'

  let { ticket, isNew = false, onerror }: { ticket: Ticket; isNew?: boolean; onerror: (m: string) => void } = $props()

  let open = $state(false)
  let finishing = $state(false)
  let busy = $state(false)
  let timer: ReturnType<typeof setTimeout> | undefined

  const due = $derived(dueOf(ticket))
  const working = $derived(ticket.status === 'in_progress')

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
  function toggleDone() {
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

  function toggleOpen() {
    open = !open
    if (isNew) Betty.Acknowledge(ticket.id)
  }
</script>

<article class="card" class:open class:finishing class:isNew>
  <button class="check" class:on={finishing} onclick={toggleDone} aria-label={finishing ? '완료 취소' : `완료: ${ticket.title}`} title={finishing ? '되돌리기' : '완료'}>
    <svg viewBox="0 0 20 20" aria-hidden="true"><circle cx="10" cy="10" r="8.5" /><path d="M6 10.4l2.7 2.6L14.2 7.4" /></svg>
  </button>

  <div class="body" role="button" tabindex="0" onclick={toggleOpen} onkeydown={(e) => e.key === 'Enter' && toggleOpen()}>
    <div class="title">{ticket.title}</div>
    <div class="meta">
      <span class="num">BT-{ticket.number}</span>
      {#if isNew}<span class="new">NEW</span>{/if}
      {#if ticket.project}<span class="project"><i style:background={ticket.project.color}></i>{ticket.project.name}</span>{/if}
      {#if ticket.status === 'in_progress' || ticket.status === 'blocked'}<span class="status {ticket.status}">{statusLabel[ticket.status]}</span>{/if}
      {#if due}<span class="due {due.tone}">{due.label}</span>{/if}
    </div>
    {#if !open && ticket.description}<div class="preview">{ticket.description}</div>{/if}
  </div>

  {#if ticket.status !== 'blocked'}
    <button class="quick" class:working onclick={() => set(working ? 'todo' : 'in_progress')} disabled={busy} title={working ? '할 일로 되돌리기' : '시작하기'} aria-label={working ? '할 일로 되돌리기' : '시작하기'}>
      {#if working}
        <svg viewBox="0 0 12 12" aria-hidden="true"><rect x="2.5" y="2" width="2.4" height="8" rx="0.6" /><rect x="7.1" y="2" width="2.4" height="8" rx="0.6" /></svg>
      {:else}
        <svg viewBox="0 0 12 12" aria-hidden="true"><path d="M3.5 2.2v7.6L10 6z" /></svg>
      {/if}
    </button>
  {/if}

  {#if open}
    <div class="details">
      <p class="desc" class:muted={!ticket.description}>{ticket.description || '설명이 없어요'}</p>
      <div class="actions">
        {#if ticket.status === 'blocked'}
          <button onclick={() => set('in_progress')} disabled={busy}>다시 진행</button>
        {:else}
          <button onclick={() => set(working ? 'todo' : 'in_progress')} disabled={busy}>{working ? '할 일로' : '시작할게'}</button>
          <button onclick={() => set('blocked')} disabled={busy}>막혔어요</button>
        {/if}
        {#if ticket.source_url}<button onclick={() => Betty.OpenURL(ticket.source_url)}>원문 ↗</button>{/if}
        <button onclick={() => Betty.OpenBoard()}>보드 ↗</button>
      </div>
    </div>
  {/if}
</article>

<style>
  .card {
    display: grid; grid-template-columns: 22px 1fr auto; column-gap: 10px; align-items: start;
    padding: 10px 10px 10px 10px; border-radius: 12px; background: var(--card);
    box-shadow: 0 0 0 0.5px var(--line); transition: opacity 0.3s ease;
  }
  .card.isNew { box-shadow: 0 0 0 0.5px var(--line), inset 3px 0 0 var(--accent); }
  .card.finishing { opacity: 0.55; }
  .card.finishing .title { text-decoration: line-through; }
  .check { width: 20px; height: 20px; margin-top: 1px; }
  .check svg { width: 20px; height: 20px; fill: none; stroke: var(--fg3); stroke-width: 1.4; }
  .check path { stroke: transparent; stroke-width: 1.8; stroke-linecap: round; stroke-linejoin: round; }
  .check:hover circle { stroke: var(--accent); }
  .check.on circle { fill: var(--accent); stroke: var(--accent); }
  .check.on path { stroke: #fff; }
  .body { min-width: 0; outline: none; }
  .title { font-size: 13.5px; font-weight: 600; line-height: 1.35; word-break: break-word; }
  .meta { display: flex; flex-wrap: wrap; align-items: center; gap: 3px 8px; margin-top: 3px; font-size: 11px; color: var(--fg3); }
  .num { font-variant-numeric: tabular-nums; font-weight: 600; }
  .new { padding: 0 5px; border-radius: 4px; background: var(--accent); color: #fff; font-size: 9.5px; font-weight: 800; }
  .project { display: inline-flex; align-items: center; gap: 4px; color: var(--fg2); }
  .project i { width: 7px; height: 7px; border-radius: 50%; }
  .status { padding: 0 6px; border-radius: 5px; font-weight: 600; }
  .status.in_progress { color: var(--accent); background: color-mix(in srgb, var(--accent) 14%, transparent); }
  .status.blocked { color: var(--red); background: color-mix(in srgb, var(--red) 13%, transparent); }
  .due { font-weight: 600; }
  .due.overdue, .due.today { color: var(--red); }
  .due.soon { color: var(--orange); }
  .preview { margin-top: 4px; font-size: 12px; color: var(--fg2); line-height: 1.4; display: -webkit-box; -webkit-line-clamp: 2; line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
  .quick {
    width: 24px; height: 24px; border-radius: 50%; display: grid; place-items: center; align-self: center;
    background: var(--fill); opacity: 0; transition: opacity 0.15s ease;
  }
  .quick svg { width: 10px; height: 10px; fill: var(--fg2); }
  .card:hover .quick, .quick:focus-visible, .quick.working { opacity: 1; }
  .quick.working svg { fill: var(--accent); }
  .quick:hover { background: var(--fill-hover); }
  .details { grid-column: 2 / 4; margin-top: 8px; }
  .desc { margin: 0; padding: 8px 10px; border-radius: 8px; background: var(--fill); font-size: 12px; line-height: 1.5; white-space: pre-wrap; max-height: 150px; overflow-y: auto; user-select: text; -webkit-user-select: text; }
  .desc.muted { color: var(--fg3); }
  .actions { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 8px; }
  .actions button { padding: 3px 10px; border-radius: 6px; background: var(--fill); font-size: 11.5px; font-weight: 600; }
  .actions button:hover { background: var(--fill-hover); }
  button:disabled { opacity: 0.5; }
</style>
