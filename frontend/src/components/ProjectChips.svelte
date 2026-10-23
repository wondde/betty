<script lang="ts">
  // One row of project chips above team-wide lists: "전체" plus each project that has open
  // work here, in that project's colour. Picking one narrows the list and the board.
  import type { Ticket } from '../../bindings/github.com/wondde/betty/internal/berry'
  import { app } from '../lib/view.svelte'
  import { nav } from '../lib/nav.svelte'

  let { tickets, all = true }: { tickets: Ticket[]; all?: boolean } = $props()

  const counts = $derived.by(() => {
    const n = new Map<string, number>()
    for (const t of tickets) if (t.status !== 'done' && t.project) n.set(t.project.id, (n.get(t.project.id) ?? 0) + 1)
    return n
  })
  const projects = $derived((app.view.projects ?? []).filter((p) => counts.get(p.id) || nav.project === p.id))
  const total = $derived([...counts.values()].reduce((a, b) => a + b, 0))
</script>

<div class="chips" role="radiogroup" aria-label="프로젝트">
  {#if all}
    <button role="radio" aria-checked={!nav.project} class:on={!nav.project} onclick={() => (nav.project = '')}>전체 <span>{total}</span></button>
  {/if}
  {#each projects as p (p.id)}
    <button role="radio" aria-checked={nav.project === p.id} class:on={nav.project === p.id} style:--pc={p.color} onclick={() => (nav.project = nav.project === p.id && all ? '' : p.id)}>
      <i aria-hidden="true"></i>{p.name} <span>{counts.get(p.id) ?? 0}</span>
    </button>
  {/each}
</div>

<style>
  .chips { display: flex; gap: 6px; padding: 0 18px 8px; overflow-x: auto; scrollbar-width: none; flex: none; }
  .chips::-webkit-scrollbar { display: none; }
  button { flex: none; display: inline-flex; align-items: center; gap: 5px; height: 26px; padding: 0 10px; border-radius: 13px; font-size: 12px; font-weight: 600; color: var(--fg2); background: var(--fill); }
  button:hover { background: var(--fill-hover); }
  button span { font-weight: 500; color: var(--fg3); font-variant-numeric: tabular-nums; }
  i { width: 8px; height: 8px; border-radius: 50%; background: var(--pc); }
  button.on { color: #fff; background: var(--pc, var(--accent)); }
  button.on span { color: rgba(255, 255, 255, 0.8); }
  button.on i { background: #fff; }
</style>
