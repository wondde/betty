<script lang="ts">
  // 지금 하고 있는 일: who has what in progress right now, one line per person, so the team
  // view answers "누가 뭐 하지?" before the long list. Clicking a task opens it; clicking a
  // name opens that person's list.
  import type { Ticket } from '../../bindings/github.com/wondde/betty/internal/berry'
  import { app } from '../lib/view.svelte'
  import { go } from '../lib/nav.svelte'

  // plain: no header or fold (the home screen titles it itself).
  let { tickets, onselect, plain = false }: { tickets: Ticket[]; onselect: (t: Ticket) => void; plain?: boolean } = $props()

  const people = $derived.by(() => {
    const by = new Map<string, Ticket[]>()
    for (const t of tickets) if (t.status === 'in_progress' && t.assignee_id) by.has(t.assignee_id) ? by.get(t.assignee_id)!.push(t) : by.set(t.assignee_id, [t])
    return [...by.entries()]
      .map(([id, items]) => ({ id, name: app.view.members?.find((m) => m.id === id)?.name ?? '팀원', items }))
      .sort((a, b) => b.items.length - a.items.length || a.name.localeCompare(b.name))
  })
  const nobody = $derived(tickets.filter((t) => t.status === 'in_progress' && !t.assignee_id).length)
  let open = $state(localStorage.getItem('nowdoing:closed') !== '1')
  function toggle() {
    open = !open
    localStorage.setItem('nowdoing:closed', open ? '0' : '1')
  }
</script>

{#if people.length || nobody}
  <section class="now">
    {#if !plain}
    <button class="head" onclick={toggle} aria-expanded={open}>
      <span class="chev" class:shut={!open}>›</span> 지금 하고 있는 일 <span class="n">{people.length}명</span>
      {#if nobody}<span class="warn" title="진행 중인데 맡은 사람이 없는 일이에요. 정리하기에서 담당을 정해 주세요">담당 없는 진행 중 {nobody}</span>{/if}
    </button>
    {/if}
    {#if open || plain}
      {#each people as p (p.id)}
        <div class="row">
          <button class="who" onclick={() => go({ kind: 'member', id: p.id })}><span class="avatar">{Array.from(p.name)[0]}</span>{p.name}</button>
          <div class="items">
            {#each p.items.slice(0, 3) as t (t.id)}
              <button class="item" style:--pc={t.project?.color} onclick={() => onselect(t)} title={t.title}><i></i>{t.title}</button>
            {/each}
            {#if p.items.length > 3}<span class="more">외 {p.items.length - 3}개</span>{/if}
          </div>
        </div>
      {/each}
    {/if}
  </section>
{/if}

<style>
  .now { margin: 4px 0 6px; padding: 6px 8px 8px; border-radius: 11px; background: var(--fill); }
  .head { display: flex; align-items: center; gap: 6px; width: 100%; padding: 2px 2px 4px; font-size: 12px; font-weight: 700; color: var(--fg2); }
  .n { font-weight: 500; color: var(--fg3); }
  .warn { margin-left: auto; padding: 0 7px; border-radius: 9px; font-size: 11px; font-weight: 600; color: var(--orange); background: color-mix(in srgb, var(--orange) 14%, transparent); }
  .chev { display: inline-block; transform: rotate(90deg); transition: transform 0.12s ease; color: var(--fg3); }
  .chev.shut { transform: rotate(0deg); }
  .row { display: flex; align-items: center; gap: 8px; padding: 3px 0; min-width: 0; }
  .who { flex: none; width: 76px; display: flex; align-items: center; gap: 6px; font-size: 12px; font-weight: 600; text-align: left; overflow: hidden; white-space: nowrap; }
  .who:hover { color: var(--accent); }
  .avatar { width: 20px; height: 20px; flex: none; border-radius: 50%; background: var(--fill-hover); display: grid; place-items: center; font-size: 10px; font-weight: 700; color: var(--fg2); }
  .items { flex: 1; min-width: 0; display: flex; gap: 5px; overflow: hidden; }
  .item { flex: 0 1 auto; min-width: 0; max-width: 240px; display: inline-flex; align-items: center; gap: 5px; height: 22px; padding: 0 8px; border-radius: 6px; font-size: 11.5px; background: var(--card-solid, #fff); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  @media (prefers-color-scheme: dark) { .item { --card-solid: #3a3a3c; } }
  .item i { width: 6px; height: 6px; flex: none; border-radius: 50%; background: var(--pc, var(--fg3)); }
  .item:hover { box-shadow: inset 0 0 0 1px var(--accent); }
  .more { flex: none; font-size: 11px; color: var(--fg3); align-self: center; }
</style>
