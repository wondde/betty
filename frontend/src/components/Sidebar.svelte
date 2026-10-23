<script lang="ts">
  // Mac source-list sidebar, kept short: four places, the projects, and the team by area.
  // Everything administrative is in Settings (⌘,), reached from the gear at the bottom.
  import { app } from '../lib/view.svelte'
  import { nav, go, sameScope, areaName, memberArea, type Area, type Scope } from '../lib/nav.svelte'

  let { onsettings }: { onsettings: (tab?: string) => void } = $props()

  const v = $derived(app.view)
  const open = $derived((v.tickets ?? []).filter((t) => t.status !== 'done'))
  const me = $derived(v.member?.id)
  const found = $derived((v.proposals?.length ?? 0) + (v.meetingProposals?.length ?? 0) + (v.designQuestions?.length ?? 0))

  const places = $derived([
    { scope: { kind: 'mine' } as Scope, icon: '☑︎', label: '내 할 일', count: open.filter((t) => t.assignee_id === me).length, hint: '나에게 맡겨진 일' },
    { scope: { kind: 'team' } as Scope, icon: '◎', label: '팀 할 일', count: open.length, hint: '팀 전체 업무 · 목록과 보드' },
    { scope: { kind: 'inbox' } as Scope, icon: '✦', label: '베티가 찾은 일', count: found, badge: true, hint: '대화에서 찾은 할 일 후보 · 맞으면 넣기' },
    { scope: { kind: 'calendar' } as Scope, icon: '▦', label: '캘린더', count: 0, hint: '팀 회의 일정' },
  ])

  const saved = (key: string) => localStorage.getItem('sidebar:' + key) === '1'
  let folded = $state<Record<string, boolean>>({ projects: saved('projects'), engineering: saved('engineering'), design: saved('design'), marketing: saved('marketing'), none: saved('none') })
  let showIdle = $state(false)
  function fold(key: string) {
    folded[key] = !folded[key]
    localStorage.setItem('sidebar:' + key, folded[key] ? '1' : '0')
  }

  const projects = $derived((v.projects ?? []).map((p) => ({ ...p, count: open.filter((t) => t.project?.id === p.id).length })))
  const isHere = (id: string) => nav.scope.kind === 'project' && nav.scope.id === id
  const activeProjects = $derived(projects.filter((p) => p.count || isHere(p.id)))
  const idleProjects = $derived(projects.filter((p) => !p.count && !isHere(p.id)))

  const areas = $derived.by(() => {
    const members = (v.members ?? []).map((m) => ({ ...m, count: open.filter((t) => t.assignee_id === m.id).length }))
    return (['engineering', 'design', 'marketing', 'none'] as Area[])
      .map((id) => ({ id, members: members.filter((m) => memberArea(m) === id) }))
      .filter((a) => a.members.length)
      .map((a) => ({ ...a, count: a.members.reduce((n, m) => n + m.count, 0) }))
  })
  const anyArea = $derived(areas.some((a) => a.id !== 'none'))
</script>

<aside class="sidebar">
  <nav aria-label="목록">
    {#each places as item}
      <button class="row" class:active={sameScope(nav.scope, item.scope)} onclick={() => go(item.scope)} title={item.hint}>
        <span class="icon" aria-hidden="true">{item.icon}</span>
        <span class="label">{item.label}</span>
        {#if item.count}<span class="count" class:badge={item.badge}>{item.count}</span>{/if}
      </button>
    {/each}

    <h3><button onclick={() => fold('projects')} aria-expanded={!folded.projects}>프로젝트 <span class="chev" class:shut={folded.projects}>›</span></button></h3>
    {#if !folded.projects}
      {#each [...activeProjects, ...(showIdle ? idleProjects : [])] as p (p.id)}
        <button class="row" class:active={sameScope(nav.scope, { kind: 'project', id: p.id })} onclick={() => go({ kind: 'project', id: p.id })}>
          <i class="dot" style:background={p.color} aria-hidden="true"></i>
          <span class="label">{p.name}</span>
          {#if p.count}<span class="count">{p.count}</span>{/if}
        </button>
      {/each}
      {#if idleProjects.length}<button class="more" onclick={() => (showIdle = !showIdle)}>{showIdle ? '접기' : `할 일 없는 프로젝트 ${idleProjects.length}개`}</button>{/if}
    {/if}

    <h3 class="team">팀</h3>
    {#each areas as a (a.id)}
      <div class="area">
        <button class="row" class:active={sameScope(nav.scope, { kind: 'area', id: a.id })} onclick={() => go({ kind: 'area', id: a.id })}>
          <span class="fold" role="button" tabindex="-1" aria-label={folded[a.id] ? '펼치기' : '접기'} onclick={(e) => (e.stopPropagation(), fold(a.id))} onkeydown={(e) => e.key === 'Enter' && fold(a.id)}>
            <span class="chev" class:shut={folded[a.id]}>›</span>
          </span>
          <span class="label">{areaName[a.id]} <small>{a.members.length}명</small></span>
          {#if a.count}<span class="count">{a.count}</span>{/if}
        </button>
        {#if !folded[a.id]}
          {#each a.members as m (m.id)}
            <button class="row person" class:active={sameScope(nav.scope, { kind: 'member', id: m.id })} onclick={() => go({ kind: 'member', id: m.id })}>
              <span class="avatar" aria-hidden="true">{Array.from(m.name)[0]}</span>
              <span class="label">{m.name}{m.id === me ? ' (나)' : ''}</span>
              {#if m.count}<span class="count">{m.count}</span>{/if}
            </button>
          {/each}
        {/if}
      </div>
    {/each}
    {#if !anyArea}<button class="more hint" onclick={() => onsettings('team')}>팀원을 개발·디자인·마케팅으로 나누려면 설정에서 영역을 정해 주세요</button>{/if}
  </nav>

  <footer>
    <span class="who">{v.member?.name}</span>
    <button class="gear" onclick={() => onsettings()} title="설정 (⌘,) · 팀원·프로젝트, 자동화, GitHub, 휴지통" aria-label="설정">
      <svg viewBox="0 0 16 16" aria-hidden="true"><circle cx="8" cy="8" r="2.2" /><path d="M8 1.5v2M8 12.5v2M1.5 8h2M12.5 8h2M3.4 3.4l1.4 1.4M11.2 11.2l1.4 1.4M3.4 12.6l1.4-1.4M11.2 4.8l1.4-1.4" /></svg>
    </button>
  </footer>
</aside>

<style>
  .sidebar { width: 210px; flex: none; display: flex; flex-direction: column; padding: 46px 8px 8px; border-right: 0.5px solid var(--line); background: var(--sidebar, transparent); }
  nav { flex: 1; overflow-y: auto; display: flex; flex-direction: column; gap: 1px; }
  h3 { margin: 16px 10px 4px; font-size: 11px; font-weight: 700; color: var(--fg3); }
  h3 button { display: flex; align-items: center; gap: 4px; width: 100%; font: inherit; color: inherit; }
  h3 button:hover { color: var(--fg2); }
  .row { display: flex; align-items: center; gap: 8px; height: 28px; padding: 0 8px; border-radius: 7px; text-align: left; }
  .row:hover { background: var(--fill); }
  .row.active { background: var(--fill-hover); font-weight: 600; }
  .icon { width: 16px; text-align: center; color: var(--accent); }
  .dot { width: 9px; height: 9px; margin: 0 3px 0 4px; border-radius: 50%; flex: none; }
  .fold { width: 16px; display: grid; place-items: center; color: var(--fg3); }
  .person { padding-left: 26px; }
  .avatar { width: 18px; height: 18px; border-radius: 50%; background: var(--fill-hover); display: grid; place-items: center; font-size: 10px; font-weight: 700; color: var(--fg2); flex: none; }
  .label { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .label small { font-size: 11px; font-weight: 500; color: var(--fg3); margin-left: 2px; }
  .count { font-size: 11px; color: var(--fg3); font-variant-numeric: tabular-nums; }
  .count.badge { padding: 0 6px; border-radius: 9px; background: var(--red); color: #fff; font-weight: 700; }
  .chev { display: inline-block; transform: rotate(90deg); transition: transform 0.12s ease; }
  h3 .chev { opacity: 0; }
  h3:hover .chev, h3 .chev.shut { opacity: 1; }
  .chev.shut { transform: rotate(0deg); }
  .more { min-height: 24px; padding: 2px 10px 2px 33px; text-align: left; font-size: 11.5px; color: var(--fg3); border-radius: 7px; }
  .more:hover { background: var(--fill); color: var(--fg2); }
  .more.hint { padding-left: 10px; line-height: 1.4; }
  footer { display: flex; align-items: center; justify-content: space-between; padding: 8px 6px 2px; border-top: 0.5px solid var(--line); }
  .who { font-size: 12px; color: var(--fg2); }
  .gear { width: 24px; height: 24px; border-radius: 6px; display: grid; place-items: center; }
  .gear:hover { background: var(--fill-hover); }
  .gear svg { width: 15px; height: 15px; fill: none; stroke: var(--fg2); stroke-width: 1.4; stroke-linecap: round; }
</style>
