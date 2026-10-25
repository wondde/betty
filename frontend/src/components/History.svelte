<script lang="ts">
  // What the assistant did on its own (GitHub links, progress, completion), newest first,
  // with the evidence it used. A wrong change can be undone from here. Several links made in
  // one decision (same task, same reason, within a minute) show as one card.
  import { onMount } from 'svelte'
  import { Betty } from '../../bindings/github.com/wondde/betty'
  import type { Change } from '../../bindings/github.com/wondde/betty/internal/berry'
  import { app, message } from '../lib/view.svelte'
  import { statusLabel } from '../lib/tasks'

  let { onerror, onsaved }: { onerror: (m: string) => void; onsaved: (m: string) => void } = $props()

  let items = $state<Change[]>([])
  let total = $state(0)
  let cursor = $state<string | null>(null)
  let filter = $state<'all' | 'active' | 'undone'>('all')
  let loading = $state(false)
  let busy = $state<string | null>(null)

  const kindLabel: Record<string, string> = { linked: 'GitHub 연결', progressed: '진행 반영', completed: '완료 처리' }
  const projectName = (id: string) => app.view.projects?.find((p) => p.id === id)?.name ?? ''
  const when = (iso: string) => new Date(iso).toLocaleString('ko-KR', { month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit' })

  async function load(reset = true) {
    loading = true
    try {
      const page = await Betty.History(reset ? '' : cursor ?? '', filter)
      items = reset ? page.items ?? [] : [...items, ...(page.items ?? [])]
      total = page.total
      cursor = page.next_cursor ?? null
    } catch (e) {
      onerror(message(e))
    } finally {
      loading = false
    }
  }
  onMount(() => void load())

  type Card = Change & { ids: string[]; undoable: string[] }
  const cards = $derived.by(() => {
    const out: Card[] = []
    for (const c of items) {
      const prev = out.at(-1)
      const same = prev && prev.kind === c.kind && prev.ticket_id === c.ticket_id && prev.reason === c.reason &&
        Math.abs(Date.parse(prev.created_at) - Date.parse(c.created_at)) < 60000 && !!prev.undone_at === !!c.undone_at
      if (same) {
        prev.ids.push(c.id)
        if (c.can_undo) prev.undoable.push(c.id)
        prev.links = [...(prev.links ?? []), ...(c.links ?? [])]
        prev.evidence = [...(prev.evidence ?? []), ...(c.evidence ?? []).filter((e) => !prev.evidence?.some((x) => x.quote === e.quote && x.url === e.url))]
      } else out.push({ ...c, links: [...(c.links ?? [])], evidence: [...(c.evidence ?? [])], ids: [c.id], undoable: c.can_undo ? [c.id] : [] })
    }
    return out
  })

  async function undo(c: Card) {
    busy = c.id
    try {
      for (const id of c.undoable) await Betty.Undo(id)
      onsaved(`BT-${c.ticket.number} 자동 처리를 되돌렸어요`)
      await load()
    } catch (e) {
      onerror(message(e))
    } finally {
      busy = null
    }
  }
</script>

<div class="history">
  <div class="filters" role="radiogroup" aria-label="기록 보기">
    {#each [['all', '전체'], ['active', '적용 중'], ['undone', '되돌림']] as [value, label]}
      <button role="radio" aria-checked={filter === value} class:on={filter === value} onclick={() => ((filter = value as typeof filter), load())}>{label}</button>
    {/each}
    <span class="total">{total}건</span>
  </div>

  {#each cards as c (c.id)}
    <article class="item" class:undone={!!c.undone_at}>
      <div class="top">
        <span class="kind {c.kind}">{kindLabel[c.kind] ?? c.kind}{c.ids.length > 1 ? ` ${c.ids.length}개` : ''}</span>
        <span class="time">{when(c.created_at)}</span>
      </div>
      <h3>BT-{c.ticket.number} {c.ticket.title}</h3>
      <p class="meta">
        {projectName(c.ticket.project_id)}{c.before_status && c.after_status && c.before_status !== c.after_status ? ` · ${statusLabel[c.before_status] ?? c.before_status} → ${statusLabel[c.after_status] ?? c.after_status}` : ''}
      </p>
      {#if c.reason}<p class="reason">{c.reason}</p>{/if}
      {#if c.evidence?.length || c.links?.length}
        <details>
          <summary>근거 {c.evidence?.length ?? 0}개{c.links?.length ? ` · 연결 ${c.links.length}개` : ''}</summary>
          {#each c.evidence ?? [] as e}
            <blockquote><b>{e.label}</b> {e.quote}{#if e.url} <button class="link" onclick={() => Betty.OpenURL(e.url)}>↗</button>{/if}</blockquote>
          {/each}
          {#each c.links ?? [] as l}
            {#if l.url}<button class="link" onclick={() => Betty.OpenURL(l.url)}>{l.label || `${l.repository} #${l.number}`} ↗</button>{/if}
          {/each}
        </details>
      {/if}
      <div class="actions">
        {#if c.undone_at}
          <span class="done">{c.undone_by ? `${c.undone_by}님이 ` : ''}되돌렸어요</span>
        {:else if c.undoable.length}
          <button disabled={busy === c.id} onclick={() => undo(c)}>되돌리기</button>
        {:else if c.undo_reason}
          <span class="muted">{c.undo_reason}</span>
        {/if}
      </div>
    </article>
  {:else}
    {#if !loading}<p class="empty">아직 자동으로 처리한 기록이 없어요.</p>{/if}
  {/each}

  {#if cursor}<button class="more" disabled={loading} onclick={() => load(false)}>{loading ? '불러오는 중…' : '더 보기'}</button>{/if}
</div>

<style>
  .history { display: flex; flex-direction: column; gap: 8px; padding: 0 16px 20px; }
  .filters { display: flex; align-items: center; gap: 4px; margin-bottom: 4px; }
  .filters button { height: 26px; padding: 0 12px; border-radius: 7px; background: var(--fill); font-size: 12px; font-weight: 600; color: var(--fg2); }
  .filters button.on { background: var(--accent); color: #fff; }
  .total { margin-left: auto; font-size: 11.5px; color: var(--fg3); }
  .item { padding: 10px 14px; border-radius: 11px; background: var(--card); box-shadow: 0 0 0 0.5px var(--line); }
  .item.undone { opacity: 0.6; }
  .top { display: flex; justify-content: space-between; }
  .kind { padding: 1px 7px; border-radius: 5px; font-size: 11px; font-weight: 700; background: var(--fill); }
  .kind.completed { color: var(--green); }
  .kind.progressed { color: var(--accent); }
  .kind.linked { color: var(--fg2); }
  .time { font-size: 11px; color: var(--fg3); }
  h3 { margin: 6px 0 2px; font-size: 13.5px; }
  .meta { margin: 0; font-size: 11.5px; color: var(--fg3); }
  .reason { margin: 6px 0; font-size: 12.5px; line-height: 1.5; color: var(--fg2); white-space: pre-wrap; }
  details { font-size: 12px; color: var(--fg2); }
  summary { color: var(--fg3); cursor: default; }
  blockquote { margin: 6px 0; padding: 6px 10px; border-left: 3px solid var(--line); background: var(--fill); border-radius: 0 6px 6px 0; white-space: pre-wrap; user-select: text; -webkit-user-select: text; }
  .link { color: var(--accent); font-size: 12px; padding: 0 4px; }
  .actions { display: flex; justify-content: flex-end; margin-top: 6px; font-size: 12px; }
  .actions button { height: 26px; padding: 0 12px; border-radius: 7px; background: var(--fill); font-weight: 600; }
  .actions button:hover { background: var(--fill-hover); }
  .done { color: var(--fg3); }
  .muted { color: var(--fg3); font-size: 11.5px; }
  .empty { text-align: center; color: var(--fg3); padding: 60px 0; }
  .more { align-self: center; height: 28px; padding: 0 16px; border-radius: 8px; background: var(--fill); font-weight: 600; }
  button:disabled { opacity: 0.5; }
</style>
