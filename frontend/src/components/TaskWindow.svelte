<script lang="ts">
  import { ask, asking } from '../lib/ask.svelte'
  import AskDialog from './AskDialog.svelte'
  // Main window, three panes:
  //   sidebar (오늘 · 베티가 찾은 일 · 모든 업무 · 캘린더 · my team · other teams · projects)
  //   | 오늘 home, or the list (or board) for that place, with search and an add field
  //   | detail pane for the selected task. Settings cover the window.
  import { onMount } from 'svelte'
  import { Events } from '@wailsio/runtime'
  import { Betty } from '../../bindings/github.com/wondde/betty'
  import { BettyCommand } from '../../bindings/events'
  import type { Ticket } from '../../bindings/github.com/wondde/betty/internal/berry'
  import { app, message } from '../lib/view.svelte'
  import { nav, go, isTaskList, byProject, areaName, memberArea, type Area } from '../lib/nav.svelte'
  import { daysUntil, groupByProject, groupByStatus, matches, needsCleanup, sortTasks, statusLabel, todayLabel, type Group } from '../lib/tasks'
  import ProjectChips from './ProjectChips.svelte'
  import Sidebar from './Sidebar.svelte'
  import TaskCard from './TaskCard.svelte'
  import TaskDetail from './TaskDetail.svelte'
  import Inbox from './Inbox.svelte'
  import Calendar from './Calendar.svelte'
  import Composer from './Composer.svelte'
  import SignIn from './SignIn.svelte'
  import Onboarding from './Onboarding.svelte'
  import Sheet from './Sheet.svelte'
  import Board from './Board.svelte'
  import CaptureSheet from './CaptureSheet.svelte'
  import SettingsView from './SettingsView.svelte'
  import CleanupView from './CleanupView.svelte'
  import NowDoing from './NowDoing.svelte'
  import Today from './Today.svelte'
  import { mod } from '../lib/platform'

  let toast = $state<{ text: string; ok: boolean } | null>(null)
  let refreshing = $state(false)
  let toastTimer: ReturnType<typeof setTimeout>
  let composer = $state<Composer>()
  let search = $state<HTMLInputElement>()
  let tour = $state(false)
  let sheet = $state<'shortcuts' | 'about' | null>(null)
  let settings = $state<string | null>(null) // open tab, or null when closed
  const openSettings = (tab = 'general') => ((sheet = null), (settings = tab))
  let capturing = $state(false)
  let cleaning = $state(false)

  // App menu commands (파일 · 편집 · 보기 · 도움말 …) arrive from Go as "betty:command".
  onMount(() =>
    Events.On(BettyCommand, (e) => {
      const name = e.data as string
      if (name === 'new-task') {
        sheet = null
        if (!isTaskList(nav.scope)) go({ kind: 'mine' })
        queueMicrotask(() => composer?.focus())
      } else if (name === 'capture') {
        sheet = null
        capturing = true
      } else if (name === 'search') {
        sheet = null
        search?.focus()
        search?.select()
      } else if (name.startsWith('go-')) {
        sheet = null
        settings = null
        const kind = name.slice(3) as 'mine' | 'team' | 'inbox' | 'calendar'
        go({ kind })
      } else if (name === 'toggle-board') {
        if (nav.scope.kind !== 'team') go({ kind: 'team' })
        nav.board = !nav.board
      } else if (name === 'toggle-done') nav.showDone = !nav.showDone
      else if (name === 'onboarding') {
        sheet = null
        tour = true
      } else if (name === 'settings') openSettings()
      else if (name === 'shortcuts' || name === 'about') sheet = name
    }),
  )

  const v = $derived(app.view)
  const me = $derived(v.member?.id)
  const isNew = $derived(new Set(v.new ?? []))
  const all = $derived(v.tickets ?? [])

  const scopeTitle = $derived.by(() => {
    const s = nav.scope
    if (s.kind === 'mine') return '오늘'
    if (s.kind === 'team') return '모든 업무'
    if (s.kind === 'inbox') return '베티가 찾은 일'
    if (s.kind === 'calendar') return '캘린더'
    if (s.kind === 'area') return `${areaName[s.id]} 팀`
    if (s.kind === 'project') return v.projects?.find((p) => p.id === s.id)?.name ?? '프로젝트'
    return v.members?.find((m) => m.id === s.id)?.name ?? '팀원'
  })

  const areaMembers = $derived(new Set(nav.scope.kind === 'area' ? (v.members ?? []).filter((m) => memberArea(m) === (nav.scope as { id: string }).id).map((m) => m.id) : []))
  const boardMode = $derived(nav.scope.kind === 'team' && nav.board)
  // Other teams open as a summary; their full list waits behind a button.
  const otherTeam = $derived(nav.scope.kind === 'area' && nav.scope.id !== memberArea(v.member ?? {}) && memberArea(v.member ?? {}) !== 'none')
  let showOtherList = $state(false)
  $effect(() => { void nav.scope; showOtherList = false })
  const unassignedCount = $derived(all.filter((t) => t.status !== 'done' && !t.assignee_id).length)
  const longTermCount = $derived(all.filter((t) => t.status !== 'done' && v.longTerm?.[t.id]).length)
  const cleanupCount = $derived(all.filter((t) => needsCleanup(t, Date.now(), Boolean(v.longTerm?.[t.id])) && (!nav.project || t.project?.id === nav.project)).length)

  // The scope before the project chip narrows it (the chips count from this).
  const scoped = $derived.by(() => {
    const s = nav.scope
    return all.filter((t) => {
      if (s.kind === 'mine') return t.assignee_id === me
      if (s.kind === 'team' && (nav.unassigned || nav.longTerm)) return (!nav.unassigned || !t.assignee_id) && (!nav.longTerm || Boolean(v.longTerm?.[t.id]))
      if (s.kind === 'area') return !!t.assignee_id && areaMembers.has(t.assignee_id)
      if (s.kind === 'project') return t.project?.id === s.id
      if (s.kind === 'member') return t.assignee_id === s.id
      return true
    })
  })
  const inScope = $derived(byProject(nav.scope) && nav.project ? scoped.filter((t) => t.project?.id === nav.project) : scoped)
  // Team-wide lists without a chosen project are sectioned by project.
  const projectSections = $derived(byProject(nav.scope) && !nav.project && nav.scope.kind !== 'area' && !boardMode)
  let collapsed = $state(new Set<string>())
  const toggleGroup = (key: string) => (collapsed = collapsed.has(key) ? new Set([...collapsed].filter((k) => k !== key)) : new Set([...collapsed, key]))
  const searching = $derived(nav.query.trim().length > 0)
  const visible = $derived(
    inScope.filter((t) => (nav.showDone || searching || t.status !== 'done') && matches(t, nav.query, v.members ?? [])),
  )

  // My list keeps the urgency order; other lists group by status.
  const groups = $derived.by((): Group[] => {
    if (nav.scope.kind === 'area') return groupByMember(visible)
    if (projectSections) return groupByProject(visible, v.projects ?? [])
    if (nav.scope.kind !== 'mine') return groupByStatus(visible).map((g) => ({ ...g, key: g.title }))
    const open = sortTasks(visible.filter((t) => t.status !== 'done'), isNew, v.order ?? [])
    const out: Group[] = []
    for (const t of open) {
      const days = daysUntil(t.due_date)
      const title = days !== null && days <= 1 ? '마감 임박' : isNew.has(t.id) ? '새로 들어온 일' : t.status === 'in_progress' ? '진행 중' : t.status === 'blocked' ? '막힘' : '할 일'
      if (out.at(-1)?.title !== title) out.push({ key: title, title, items: [] })
      out.at(-1)!.items.push(t)
    }
    const done = visible.filter((t) => t.status === 'done').sort((a, b) => b.updated_at.localeCompare(a.updated_at))
    if (done.length) out.push({ key: '완료', title: '완료', items: done })
    return out
  })
  // An area lists each person's work under their name, busiest first.
  function groupByMember(tickets: Ticket[]): Group[] {
    const by = new Map<string, Ticket[]>()
    for (const t of groupByStatus(tickets).flatMap((g) => g.items)) by.has(t.assignee_id!) ? by.get(t.assignee_id!)!.push(t) : by.set(t.assignee_id!, [t])
    return [...by.entries()]
      .sort((a, b) => b[1].length - a[1].length)
      .map(([id, items]) => ({ key: id, title: v.members?.find((m) => m.id === id)?.name ?? '팀원', items }))
  }
  const flat = $derived(groups.filter((g) => !collapsed.has(g.key)).flatMap((g) => g.items))
  const showProjectChip = $derived(nav.scope.kind !== 'project' && !nav.project && !projectSections)
  // The board shows every status itself (done is limited to recent work inside the board).
  const boardTickets = $derived(inScope.filter((t) => matches(t, nav.query, v.members ?? [])))
  // The board is per project: arriving without one picks the busiest.
  $effect(() => {
    if (!boardMode || nav.project) return
    const n = new Map<string, number>()
    for (const t of scoped) if (t.status !== 'done' && t.project) n.set(t.project.id, (n.get(t.project.id) ?? 0) + 1)
    const busiest = [...n.entries()].sort((a, b) => b[1] - a[1])[0]
    if (busiest) nav.project = busiest[0]
  })
  const selected = $derived(all.find((t) => t.id === nav.selected) ?? null)

  const summary = $derived.by(() => {
    const open = inScope.filter((t) => t.status !== 'done')
    const urgent = open.filter((t) => (daysUntil(t.due_date) ?? 99) <= 1).length
    const working = open.filter((t) => t.status === 'in_progress').length
    return [`남은 일 ${open.length}`, working && `진행 중 ${working}`, urgent && `마감 임박 ${urgent}`].filter(Boolean).join(' · ')
  })
  const subtitle = $derived.by(() => {
    switch (nav.scope.kind) {
      case 'inbox': return '베티가 디스코드 대화에서 찾은 할 일 후보예요. 맞으면 넣고, 아니면 넘기세요.'
      case 'calendar': return '팀 회의 일정 · 한국·일본 시간'
      case 'team': return boardMode ? `${summary} · 카드를 끌어 상태를 바꿔요` : nav.longTerm ? `조금씩 진행하는 일 · ${summary}` : nav.unassigned ? `아직 아무도 맡지 않은 일 · ${summary}` : summary
      case 'area': return `${summary} · 사람별로 묶었어요`
      case 'mine': return `${v.member?.name ?? ''}님 · ${todayLabel()}`
      default: return summary
    }
  })
  const synced = $derived(v.syncedAt ? new Date(v.syncedAt).toLocaleTimeString('ko-KR', { hour: '2-digit', minute: '2-digit' }) : '')

  function say(text: string, ok = false) {
    toast = { text, ok }
    clearTimeout(toastTimer)
    toastTimer = setTimeout(() => (toast = null), 3200)
  }

  function select(t: Ticket) {
    nav.selected = nav.selected === t.id ? null : t.id
    if (isNew.has(t.id)) Betty.Acknowledge(t.id)
  }

  async function refresh() {
    refreshing = true
    try {
      await Betty.Refresh()
    } catch (e) {
      say(message(e))
    } finally {
      refreshing = false
    }
  }

  async function trash(t: Ticket) {
    if (!(await ask(`BT-${t.number} "${t.title}"을(를) 휴지통으로 옮길까요?\n휴지통에서 되살릴 수 있어요.`, '휴지통으로', true))) return
    try {
      await Betty.Delete(t.id)
      nav.selected = null
      say('휴지통으로 옮겼어요', true)
    } catch (err) {
      say(message(err))
    }
  }

  // ↑/↓ move the selection through the list; Esc clears search, then the selection.
  function keydown(e: KeyboardEvent) {
    const target = e.target as HTMLElement
    const typing = target.matches('input, textarea, select')
    if (e.key === 'Escape') {
      if (nav.query) nav.query = ''
      else if (nav.selected) nav.selected = null
      if (typing) target.blur()
      return
    }
    // ⌘⌫ moves the selected task to the trash (only outside text fields).
    if (!typing && !sheet && !capturing && !cleaning && !settings && e.metaKey && e.key === 'Backspace' && selected && isTaskList(nav.scope)) {
      e.preventDefault()
      const t = selected
      trash(t)
      return
    }
    if (typing || sheet || capturing || cleaning || settings || asking.current || !(e.key === 'ArrowDown' || e.key === 'ArrowUp') || !flat.length) return
    e.preventDefault()
    const i = flat.findIndex((t) => t.id === nav.selected)
    const next = e.key === 'ArrowDown' ? Math.min(flat.length - 1, i + 1) : Math.max(0, i < 0 ? 0 : i - 1)
    nav.selected = flat[next].id
    document.querySelector(`[data-task="${flat[next].id}"]`)?.scrollIntoView({ block: 'nearest' })
  }
</script>

<svelte:window onkeydown={keydown} />

<div class="titlebar" aria-hidden="true"></div>
{#if !v.signedIn}
  <SignIn />
{:else if !v.onboarded || tour}
  <Onboarding ondone={() => (tour = false)} />
{:else}
  <div class="layout">
    <Sidebar onsettings={(tab) => openSettings(tab)} />

    <main>
      <header>
        <div class="heading">
          <h1>{scopeTitle}</h1>
          <p class="sub">{subtitle}</p>
        </div>
        <div class="tools">
          {#if isTaskList(nav.scope)}
            <label class="search">
              <svg viewBox="0 0 16 16" aria-hidden="true"><circle cx="7" cy="7" r="4.5" /><path d="M10.5 10.5L14 14" /></svg>
              <input bind:this={search} bind:value={nav.query} placeholder={`검색 (${mod}F)`} aria-label="업무 검색" spellcheck="false" />
            </label>
            {#if nav.scope.kind === 'team' && cleanupCount}
              <button class="tidy" onclick={() => (cleaning = true)} title="아무도 안 맡았거나 일주일 넘게 그대로인 일을 하나씩 정리해요">정리하기 <span>{cleanupCount}</span></button>
            {/if}
            {#if nav.scope.kind === 'team'}
              <div class="seg" role="radiogroup" aria-label="보기">
                <button role="radio" aria-checked={!nav.board} class:on={!nav.board} onclick={() => (nav.board = false)}>목록</button>
                <button role="radio" aria-checked={nav.board} class:on={nav.board} onclick={() => (nav.board = true)} title="보드 (⌘B)">보드</button>
              </div>
            {/if}
            <button class="toggle" class:on={nav.showDone} onclick={() => (nav.showDone = !nav.showDone)} title="완료한 일 보기 (⇧⌘D)">완료 포함</button>
          {/if}
          <button class="icon" onclick={refresh} class:spin={refreshing} title={synced ? `새로고침 (⌘R) · ${synced} 동기화` : '새로고침 (⌘R)'} aria-label="새로고침">
            <svg viewBox="0 0 16 16" aria-hidden="true"><path d="M13.5 8a5.5 5.5 0 1 1-1.6-3.9M13.5 2.5v3h-3" /></svg>
          </button>
        </div>
      </header>
      {#if byProject(nav.scope)}
        <div class="filters">
          {#if nav.scope.kind === 'team'}
            <button class="who" class:on={nav.unassigned} onclick={() => (nav.unassigned = !nav.unassigned)} title="아직 아무도 맡지 않은 일만 보기">아무도 안 맡은 일 <span>{unassignedCount}</span></button>
            {#if longTermCount}<button class="who" class:on={nav.longTerm} onclick={() => (nav.longTerm = !nav.longTerm)} title="개선·조사·운영처럼 조금씩 진행하는 일만 보기">🌱 길게 보는 일 <span>{longTermCount}</span></button>{/if}
          {/if}
          <ProjectChips tickets={scoped} all={!boardMode} />
        </div>
      {/if}
      {#if v.offline}<p class="offline">오프라인 · 마지막으로 불러온 내용이에요</p>{/if}

      <section class="list">
        {#if nav.scope.kind === 'inbox'}
          <div class="inbox-tools"><button onclick={() => (capturing = true)}>＋ 대화 붙여 넣어 할 일 찾기 <span>⇧⌘N</span></button></div>
          <Inbox onerror={(m) => say(m)} onsaved={(m) => say(m, true)} />
        {:else if nav.scope.kind === 'calendar'}
          <Calendar onerror={(m) => say(m)} onsaved={(m) => say(m, true)} />
        {:else if nav.scope.kind === 'mine' && !searching}
          <Today selected={nav.selected} {isNew} onselect={select} onerror={(m) => say(m)} />
        {:else if boardMode}
          <Board tickets={boardTickets} onselect={select} onerror={(m) => say(m)} />
        {:else}
          {#if (nav.scope.kind === 'team' && !nav.unassigned && !nav.longTerm) || nav.scope.kind === 'area'}<NowDoing tickets={inScope} onselect={select} />{/if}
          {#if otherTeam && !showOtherList && !searching}
            <!-- Another team: who is on what is enough; the full list is one click away. -->
            <button class="showall" onclick={() => (showOtherList = true)}>{areaName[(nav.scope as { id: Area }).id]} 팀 업무 목록 모두 보기 · {visible.length}개</button>
          {:else}
          {#each groups as g (g.key)}
            <h2 class:section={g.color} style:--pc={g.color}>
              <button onclick={() => toggleGroup(g.key)} aria-expanded={!collapsed.has(g.key)}>
                <span class="chev" class:shut={collapsed.has(g.key)} aria-hidden="true">›</span>{#if g.color}<i aria-hidden="true"></i>{/if}{g.title} <span class="n">{g.items.length}</span>
              </button>
            </h2>
            {#if !collapsed.has(g.key)}
              {#each g.items as t (t.id)}
                <div data-task={t.id}>
                  <TaskCard ticket={t} isNew={isNew.has(t.id)} selected={nav.selected === t.id} showAssignee={!['mine', 'member', 'area'].includes(nav.scope.kind)}
                    showProject={showProjectChip} showStatus={g.title !== statusLabel[t.status]} onselect={() => select(t)} onerror={(m) => say(m)} />
                </div>
              {/each}
            {/if}
          {:else}
            <div class="empty">
              {#if searching}
                <p><b>"{nav.query}"에 맞는 업무가 없어요</b></p>
                <p>완료한 일까지 함께 찾아봤어요.</p>
              {:else}
                <div class="big">🎉</div>
                <p><b>남은 일이 없어요</b></p>
                <p>{nav.scope.kind === 'mine' ? '새 업무가 배정되면 알림으로 알려 드릴게요.' : '이 목록에는 지금 할 일이 없어요.'}</p>
              {/if}
            </div>
          {/each}
          {/if}
        {/if}
      </section>

      {#if isTaskList(nav.scope)}
        <footer>
          {#if toast}<p class="toast" class:ok={toast.ok}>{toast.text}</p>{/if}
          <Composer bind:this={composer} onresult={(text, ok) => say(text, ok)} />
        </footer>
      {:else if toast}
        <footer><p class="toast inline" class:ok={toast.ok}>{toast.text}</p></footer>
      {/if}
    </main>

    {#if selected && isTaskList(nav.scope)}
      <TaskDetail ticket={selected} onerror={(m) => say(m)} onsaved={(m) => say(m, true)} />
    {/if}
  </div>
  <AskDialog />
  {#if capturing}<CaptureSheet onclose={() => (capturing = false)} onsent={(m) => say(m, true)} />{/if}
  {#if cleaning}<CleanupView project={nav.project} onclose={() => (cleaning = false)} onsaved={(m) => say(m, true)} onerror={(m) => say(m)} />{/if}
  {#if settings}<SettingsView tab={settings} onclose={() => (settings = null)} ontour={() => ((settings = null), (tour = true))} onerror={(m) => say(m)} onsaved={(m) => say(m, true)} />{/if}
  {#if sheet}<Sheet mode={sheet} onclose={() => (sheet = null)} ontour={() => ((sheet = null), (tour = true))} />{/if}
{/if}

<style>
  /* The window keeps its traffic lights (hidden-inset title bar); this strip is where you drag it. */
  .titlebar { position: fixed; top: 0; left: 0; right: 0; height: 40px; --wails-draggable: drag; z-index: 5; }
  .layout { height: 100%; display: flex; }
  main { flex: 1; min-width: 0; display: flex; flex-direction: column; }
  header { display: flex; align-items: flex-end; justify-content: space-between; gap: 12px; padding: calc(var(--top) - 2px) 18px 8px; }
  .heading { min-width: 0; }
  h1 { margin: 0; font-size: 20px; font-weight: 700; letter-spacing: -0.4px; }
  .sub { margin: 2px 0 0; font-size: 11.5px; color: var(--fg3); font-weight: 500; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .tools { display: flex; align-items: center; gap: 6px; flex: none; position: relative; z-index: 6; }
  .search { display: flex; align-items: center; gap: 6px; height: 28px; padding: 0 8px; border-radius: 7px; background: var(--fill); width: 190px; }
  .search:focus-within { box-shadow: inset 0 0 0 1.5px var(--accent); }
  .search svg { width: 13px; height: 13px; fill: none; stroke: var(--fg3); stroke-width: 1.6; stroke-linecap: round; }
  .search input { flex: 1; min-width: 0; border: none; outline: none; background: transparent; font-size: 12.5px; }
  .toggle { height: 28px; padding: 0 10px; border-radius: 7px; font-size: 12px; font-weight: 600; color: var(--fg2); background: var(--fill); }
  .toggle.on { background: var(--accent); color: #fff; }
  .icon { width: 28px; height: 28px; border-radius: 7px; display: grid; place-items: center; }
  .icon:hover { background: var(--fill-hover); }
  .icon svg { width: 15px; height: 15px; fill: none; stroke: var(--fg2); stroke-width: 1.5; stroke-linecap: round; stroke-linejoin: round; }
  .spin svg { animation: spin 0.8s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
  .offline { margin: 0 18px 6px; padding: 5px 10px; border-radius: 8px; background: var(--fill); font-size: 11px; color: var(--fg2); }
  .list { flex: 1; overflow-y: auto; padding: 0 10px 10px; }
  h2 { margin: 12px 4px 2px; font-size: 11px; font-weight: 700; color: var(--fg3); }
  h2 button { display: inline-flex; align-items: center; gap: 6px; padding: 3px 6px; border-radius: 6px; font: inherit; color: inherit; }
  h2 button:hover { background: var(--fill); }
  h2 .n { font-weight: 500; }
  h2.section { position: sticky; top: 0; z-index: 2; margin: 0 -10px; padding: 8px 14px 4px; background: var(--sticky); backdrop-filter: blur(12px); font-size: 12.5px; color: var(--fg); }
  h2.section i { width: 9px; height: 9px; border-radius: 50%; background: var(--pc); }
  .chev { display: inline-block; width: 8px; transition: transform 0.12s ease; transform: rotate(90deg); color: var(--fg3); }
  .chev.shut { transform: rotate(0deg); }
  .empty { text-align: center; padding: 70px 20px; color: var(--fg3); line-height: 1.6; }
  .empty p { margin: 4px 0; }
  .empty b { color: var(--fg); font-size: 14px; }
  .empty .big { font-size: 32px; }
  footer { position: relative; padding: 8px 14px 12px; border-top: 0.5px solid var(--line); }
  .toast { position: absolute; left: 14px; right: 14px; bottom: calc(100% + 6px); margin: 0; padding: 8px 12px; border-radius: 10px; background: rgba(30, 30, 32, 0.92); color: #fff; font-size: 12px; font-weight: 500; z-index: 4; }
  .toast.inline { position: static; }
  .inbox-tools { padding: 2px 8px 6px; }
  .inbox-tools button { width: 100%; display: flex; align-items: center; justify-content: space-between; padding: 8px 12px; border-radius: 9px; border: 1px dashed var(--line); color: var(--fg2); font-size: 12.5px; font-weight: 600; }
  .inbox-tools button:hover { background: var(--fill); color: var(--fg); }
  .inbox-tools span { font-weight: 500; color: var(--fg3); font-size: 11px; }
  .showall { width: calc(100% - 8px); margin: 6px 4px; padding: 9px 12px; border-radius: 9px; border: 1px dashed var(--line); font-size: 12.5px; font-weight: 600; color: var(--fg2); }
  .showall:hover { background: var(--fill); color: var(--fg); }
  .tidy { display: inline-flex; align-items: center; gap: 5px; height: 28px; padding: 0 10px; border-radius: 7px; font-size: 12px; font-weight: 700; color: #fff; background: var(--accent); }
  .tidy span { font-weight: 500; opacity: 0.85; }
  .seg { display: flex; gap: 2px; padding: 2px; border-radius: 8px; background: var(--fill); }
  .seg button { height: 24px; padding: 0 10px; border-radius: 6px; font-size: 12px; font-weight: 600; color: var(--fg2); }
  .seg button.on { background: var(--card-solid, #fff); color: var(--fg); box-shadow: 0 1px 2px rgba(0,0,0,0.12); }
  @media (prefers-color-scheme: dark) { .seg button.on { --card-solid: #48484a; } }
  .filters { display: flex; align-items: center; gap: 6px; padding-left: 18px; min-width: 0; }
  .filters :global(.chips) { padding-left: 0; flex: 1; min-width: 0; }
  .who { flex: none; display: inline-flex; align-items: center; gap: 5px; height: 26px; margin-bottom: 8px; padding: 0 10px; border-radius: 13px; font-size: 12px; font-weight: 600; color: var(--fg2); background: var(--fill); box-shadow: inset 0 0 0 1px var(--line); }
  .who span { font-weight: 500; color: var(--fg3); }
  .who.on { background: var(--fg); color: var(--bg-invert, #fff); }
  @media (prefers-color-scheme: dark) { .who.on { color: #1c1c1e; } }
  .who.on span { color: inherit; opacity: 0.7; }
  .toast.ok { background: color-mix(in srgb, var(--green) 85%, black); }
</style>
