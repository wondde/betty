<script lang="ts">
  // Sidebar, kept short: four places, my team (each person), the other teams (one row each),
  // then projects. Team setup lives in Settings, which admins reach from the gear at the bottom.
  import { app, asAdmin, setAdminView } from '../lib/view.svelte'
  import { nav, go, sameScope, areaName, memberArea, type Area, type Scope } from '../lib/nav.svelte'

  let { onsettings }: { onsettings: (tab?: string) => void } = $props()

  const v = $derived(app.view)
  const open = $derived((v.tickets ?? []).filter((t) => t.status !== 'done'))
  const me = $derived(v.member?.id)
  const found = $derived((v.proposals?.length ?? 0) + (v.meetingProposals?.length ?? 0) + (v.designQuestions?.length ?? 0))
  // Red only when something there waits on me (admins look after all of it).
  const foundForMe = $derived(asAdmin() || (v.proposals ?? []).some((p) => p.payload?.assignee_id === me))

  const places = $derived([
    { scope: { kind: 'mine' } as Scope, icon: '☀︎', label: '오늘', count: open.filter((t) => t.assignee_id === me).length, hint: '내가 할 일과 확인할 것' },
    { scope: { kind: 'inbox' } as Scope, icon: '✦', label: '베티가 찾은 일', count: found, badge: foundForMe, hint: '대화에서 찾은 할 일 후보 · 맞으면 넣기' },
    { scope: { kind: 'team' } as Scope, icon: '◎', label: '모든 업무', count: 0, hint: '팀 전체 업무 · 목록과 보드' },
    { scope: { kind: 'calendar' } as Scope, icon: '▦', label: '캘린더', count: 0, hint: '팀 회의 일정' },
  ])

  const saved = (key: string, fallback = false) => { const x = localStorage.getItem('sidebar:' + key); return x === null ? fallback : x === '1' }
  let folded = $state<Record<string, boolean>>({ projects: saved('projects', true) })
  let showIdle = $state(false)
  function fold(key: string) {
    folded[key] = !folded[key]
    localStorage.setItem('sidebar:' + key, folded[key] ? '1' : '0')
  }

  const projects = $derived((v.projects ?? []).map((p) => ({ ...p, count: open.filter((t) => t.project?.id === p.id).length })))
  const isHere = (id: string) => nav.scope.kind === 'project' && nav.scope.id === id
  const activeProjects = $derived(projects.filter((p) => p.count || isHere(p.id)))
  const idleProjects = $derived(projects.filter((p) => !p.count && !isHere(p.id)))

  const members = $derived((v.members ?? []).map((m) => ({ ...m, count: open.filter((t) => t.assignee_id === m.id).length })))
  const myArea = $derived(memberArea(v.member ?? {}))
  // Without an area of my own, everyone counts as my team.
  const myTeam = $derived(members.filter((m) => myArea === 'none' || memberArea(m) === myArea).sort((a, b) => (a.id === me ? -1 : b.id === me ? 1 : 0)))
  const otherTeams = $derived(
    myArea === 'none' ? [] : (['engineering', 'design', 'marketing', 'none'] as Area[])
      .filter((a) => a !== myArea)
      .map((id) => ({ id, members: members.filter((m) => memberArea(m) === id) }))
      .filter((a) => a.members.length)
      .map((a) => ({ ...a, count: a.members.reduce((n, m) => n + m.count, 0) })),
  )
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

    <h3>
      {#if myArea === 'none'}팀{:else}<button onclick={() => go({ kind: 'area', id: myArea })}>우리 팀 · {areaName[myArea]}</button>{/if}
    </h3>
    {#each myTeam as m (m.id)}
      <button class="row" class:active={sameScope(nav.scope, { kind: 'member', id: m.id })} onclick={() => go(m.id === me ? { kind: 'mine' } : { kind: 'member', id: m.id })}>
        <span class="avatar" aria-hidden="true">{Array.from(m.name)[0]}</span>
        <span class="label">{m.name}{m.id === me ? ' (나)' : ''}</span>
        {#if m.count}<span class="count">{m.count}</span>{/if}
      </button>
    {/each}

    {#if otherTeams.length}
      <h3>다른 팀</h3>
      {#each otherTeams as a (a.id)}
        <button class="row" class:active={sameScope(nav.scope, { kind: 'area', id: a.id })} onclick={() => go({ kind: 'area', id: a.id })}>
          <span class="icon muted" aria-hidden="true">●</span>
          <span class="label">{areaName[a.id]} <small>{a.members.length}명</small></span>
          {#if a.count}<span class="count">{a.count}</span>{/if}
        </button>
      {/each}
    {/if}
    {#if myArea === 'none' && asAdmin()}<button class="more hint" onclick={() => onsettings('team')}>팀원마다 개발·디자인·마케팅 영역을 정하면 자기 팀 일을 따로 볼 수 있어요</button>{/if}

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
  </nav>

  <footer>
    <span class="who">{v.member?.name}</span>
    {#if v.isAdmin}
      <!-- Admins switch between their own screen and the one everyone else sees. -->
      <div class="seg" role="radiogroup" aria-label="화면">
        <button role="radio" aria-checked={asAdmin()} class:on={asAdmin()} onclick={() => setAdminView(true)} title="팀원·프로젝트, GitHub, 자동화 설정까지 보여요">관리자</button>
        <button role="radio" aria-checked={!asAdmin()} class:on={!asAdmin()} onclick={() => setAdminView(false)} title="다른 팀원이 보는 화면이에요">팀원</button>
      </div>
    {/if}
    <button class="gear" onclick={() => onsettings()} title="설정" aria-label="설정">
      <svg viewBox="0 0 16 16" aria-hidden="true"><circle cx="8" cy="8" r="2.2" /><path d="M8 1.5v2M8 12.5v2M1.5 8h2M12.5 8h2M3.4 3.4l1.4 1.4M11.2 11.2l1.4 1.4M3.4 12.6l1.4-1.4M11.2 4.8l1.4-1.4" /></svg>
    </button>
  </footer>
</aside>

<style>
  .sidebar { width: 210px; flex: none; display: flex; flex-direction: column; padding: var(--top) 8px 8px; border-right: 0.5px solid var(--line); background: var(--sidebar, transparent); }
  nav { flex: 1; overflow-y: auto; display: flex; flex-direction: column; gap: 1px; }
  nav > * { flex-shrink: 0; } /* the list scrolls; rows never squash */
  h3 { margin: 16px 10px 4px; font-size: 11px; font-weight: 700; color: var(--fg3); }
  h3 button { display: flex; align-items: center; gap: 4px; width: 100%; font: inherit; color: inherit; }
  h3 button:hover { color: var(--fg2); }
  .row { display: flex; align-items: center; gap: 8px; height: 28px; padding: 0 8px; border-radius: 7px; text-align: left; }
  .row:hover { background: var(--fill); }
  .row.active { background: var(--fill-hover); font-weight: 600; }
  .icon { width: 16px; text-align: center; color: var(--accent); }
  .icon.muted { font-size: 8px; color: var(--fg3); }
  .dot { width: 9px; height: 9px; margin: 0 3px 0 4px; border-radius: 50%; flex: none; }
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
  .who { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .seg { display: flex; gap: 2px; margin-right: 4px; padding: 2px; border-radius: 7px; background: var(--fill); }
  .seg button { height: 20px; padding: 0 7px; border-radius: 5px; font-size: 11px; font-weight: 600; color: var(--fg3); }
  .seg button.on { background: var(--card-solid, #fff); color: var(--fg); box-shadow: 0 1px 2px rgba(0,0,0,0.12); }
  @media (prefers-color-scheme: dark) { .seg button.on { --card-solid: #48484a; } }
  .gear { width: 24px; height: 24px; border-radius: 6px; display: grid; place-items: center; }
  .gear:hover { background: var(--fill-hover); }
  .gear svg { width: 15px; height: 15px; fill: none; stroke: var(--fg2); stroke-width: 1.4; stroke-linecap: round; }
</style>
