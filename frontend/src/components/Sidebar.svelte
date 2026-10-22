<script lang="ts">
  // Mac source-list sidebar: smart lists, review inbox, projects and people with open counts.
  import { app } from '../lib/view.svelte'
  import { nav, go, sameScope, type Scope } from '../lib/nav.svelte'

  let { onsettings }: { onsettings: () => void } = $props()

  const v = $derived(app.view)
  const open = $derived((v.tickets ?? []).filter((t) => t.status !== 'done'))
  const me = $derived(v.member?.id)
  const inboxCount = $derived((v.proposals?.length ?? 0) + (v.meetingProposals?.length ?? 0) + (v.designQuestions?.length ?? 0))

  const smart = $derived([
    { scope: { kind: 'mine' } as Scope, icon: '☑︎', label: '내 할 일', count: open.filter((t) => t.assignee_id === me).length },
    { scope: { kind: 'team' } as Scope, icon: '◎', label: '팀 전체', count: open.length },
    { scope: { kind: 'unassigned' } as Scope, icon: '○', label: '담당자 없음', count: open.filter((t) => !t.assignee_id).length },
    { scope: { kind: 'inbox' } as Scope, icon: '✉︎', label: '확인해 주세요', count: inboxCount, badge: true },
  ])
  const soonMeetings = $derived.by(() => {
    const today = new Date().toISOString().slice(0, 10), week = new Date(Date.now() + 7 * 86400000).toISOString().slice(0, 10)
    return (v.meetings ?? []).filter((m) => m.status !== 'cancelled' && m.date >= today && m.date <= week).length
  })
  const views = $derived([
    { scope: { kind: 'calendar' } as Scope, icon: '▦', label: '캘린더', count: soonMeetings },
    { scope: { kind: 'teamStatus' } as Scope, icon: '▤', label: '팀 현황', count: 0 },
    { scope: { kind: 'history' } as Scope, icon: '↺', label: '자동 정리 기록', count: 0 },
    { scope: { kind: 'github' } as Scope, icon: '⇄', label: '개발 현황', count: v.github?.suggestions?.length ?? 0 },
    { scope: { kind: 'admin' } as Scope, icon: '⚙︎', label: '팀·연결 설정', count: 0 },
  ])
  const projects = $derived((v.projects ?? []).map((p) => ({ ...p, count: open.filter((t) => t.project?.id === p.id).length })))
  const members = $derived((v.members ?? []).map((m) => ({ ...m, count: open.filter((t) => t.assignee_id === m.id).length })))
</script>

<aside class="sidebar">
  <nav aria-label="목록">
    {#each smart as item}
      <button class="row" class:active={sameScope(nav.scope, item.scope)} onclick={() => go(item.scope)}>
        <span class="icon" aria-hidden="true">{item.icon}</span>
        <span class="label">{item.label}</span>
        {#if item.count}<span class="count" class:badge={item.badge}>{item.count}</span>{/if}
      </button>
    {/each}

    <h3>보기</h3>
    {#each views as item}
      <button class="row" class:active={sameScope(nav.scope, item.scope)} onclick={() => go(item.scope)}>
        <span class="icon" aria-hidden="true">{item.icon}</span>
        <span class="label">{item.label}</span>
        {#if item.count}<span class="count">{item.count}</span>{/if}
      </button>
    {/each}

    <h3>프로젝트</h3>
    {#each projects as p (p.id)}
      <button class="row" class:active={sameScope(nav.scope, { kind: 'project', id: p.id })} onclick={() => go({ kind: 'project', id: p.id })}>
        <i class="dot" style:background={p.color} aria-hidden="true"></i>
        <span class="label">{p.name}</span>
        {#if p.count}<span class="count">{p.count}</span>{/if}
      </button>
    {/each}

    <h3>팀원</h3>
    {#each members as m (m.id)}
      <button class="row" class:active={sameScope(nav.scope, { kind: 'member', id: m.id })} onclick={() => go({ kind: 'member', id: m.id })}>
        <span class="avatar" aria-hidden="true">{Array.from(m.name)[0]}</span>
        <span class="label">{m.name}{m.id === me ? ' (나)' : ''}</span>
        {#if m.count}<span class="count">{m.count}</span>{/if}
      </button>
    {/each}
  </nav>

  <footer>
    <span class="who">{v.member?.name}</span>
    <button class="gear" onclick={onsettings} title="설정 (⌘,)" aria-label="설정">
      <svg viewBox="0 0 16 16" aria-hidden="true"><circle cx="8" cy="8" r="2.2" /><path d="M8 1.5v2M8 12.5v2M1.5 8h2M12.5 8h2M3.4 3.4l1.4 1.4M11.2 11.2l1.4 1.4M3.4 12.6l1.4-1.4M11.2 4.8l1.4-1.4" /></svg>
    </button>
  </footer>
</aside>

<style>
  .sidebar { width: 210px; flex: none; display: flex; flex-direction: column; padding: 46px 8px 8px; border-right: 0.5px solid var(--line); background: var(--sidebar, transparent); }
  nav { flex: 1; overflow-y: auto; display: flex; flex-direction: column; gap: 1px; }
  h3 { margin: 14px 10px 4px; font-size: 11px; font-weight: 700; color: var(--fg3); }
  .row { display: flex; align-items: center; gap: 8px; height: 28px; padding: 0 8px; border-radius: 7px; text-align: left; }
  .row:hover { background: var(--fill); }
  .row.active { background: var(--fill-hover); font-weight: 600; }
  .icon { width: 16px; text-align: center; color: var(--accent); }
  .dot { width: 9px; height: 9px; margin: 0 3px 0 4px; border-radius: 50%; flex: none; }
  .avatar { width: 18px; height: 18px; border-radius: 50%; background: var(--fill-hover); display: grid; place-items: center; font-size: 10px; font-weight: 700; color: var(--fg2); flex: none; }
  .label { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .count { font-size: 11px; color: var(--fg3); font-variant-numeric: tabular-nums; }
  .count.badge { padding: 0 6px; border-radius: 9px; background: var(--red); color: #fff; font-weight: 700; }
  footer { display: flex; align-items: center; justify-content: space-between; padding: 8px 6px 2px; border-top: 0.5px solid var(--line); }
  .who { font-size: 12px; color: var(--fg2); }
  .gear { width: 24px; height: 24px; border-radius: 6px; display: grid; place-items: center; }
  .gear:hover { background: var(--fill-hover); }
  .gear svg { width: 15px; height: 15px; fill: none; stroke: var(--fg2); stroke-width: 1.4; stroke-linecap: round; }
</style>
