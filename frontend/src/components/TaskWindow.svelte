<script lang="ts">
  // Main window, Mac three-pane layout:
  //   sidebar (my tasks · team · unassigned · review inbox · projects · people)
  //   | list for that scope, with search (⌘F) and an add field
  //   | detail pane for the selected task.
  import { onMount } from 'svelte'
  import { Events } from '@wailsio/runtime'
  import { Betty } from '../../bindings/github.com/wondde/betty'
  import { BettyCommand } from '../../bindings/events'
  import type { Ticket } from '../../bindings/github.com/wondde/betty/internal/berry'
  import { app, message } from '../lib/view.svelte'
  import { nav, go, isTaskList } from '../lib/nav.svelte'
  import { daysUntil, groupByStatus, matches, sortTasks, todayLabel } from '../lib/tasks'
  import Sidebar from './Sidebar.svelte'
  import TaskCard from './TaskCard.svelte'
  import TaskDetail from './TaskDetail.svelte'
  import Inbox from './Inbox.svelte'
  import Calendar from './Calendar.svelte'
  import History from './History.svelte'
  import TeamStatus from './TeamStatus.svelte'
  import Composer from './Composer.svelte'
  import SignIn from './SignIn.svelte'
  import Onboarding from './Onboarding.svelte'
  import Sheet from './Sheet.svelte'

  let toast = $state<{ text: string; ok: boolean } | null>(null)
  let refreshing = $state(false)
  let toastTimer: ReturnType<typeof setTimeout>
  let composer = $state<Composer>()
  let search = $state<HTMLInputElement>()
  let tour = $state(false)
  let sheet = $state<'settings' | 'shortcuts' | 'about' | null>(null)

  // App menu commands (파일 · 편집 · 보기 · 도움말 …) arrive from Go as "betty:command".
  onMount(() =>
    Events.On(BettyCommand, (e) => {
      const name = e.data as string
      if (name === 'new-task') {
        sheet = null
        if (!isTaskList(nav.scope)) go({ kind: 'mine' })
        queueMicrotask(() => composer?.focus())
      } else if (name === 'search') {
        sheet = null
        search?.focus()
        search?.select()
      } else if (name.startsWith('go-')) {
        sheet = null
        const kind = name.slice(3) as 'mine' | 'team' | 'inbox' | 'calendar' | 'teamStatus' | 'history'
        go({ kind })
      } else if (name === 'toggle-done') nav.showDone = !nav.showDone
      else if (name === 'onboarding') {
        sheet = null
        tour = true
      } else if (name === 'settings' || name === 'shortcuts' || name === 'about') sheet = name
    }),
  )

  const v = $derived(app.view)
  const me = $derived(v.member?.id)
  const isNew = $derived(new Set(v.new ?? []))
  const all = $derived(v.tickets ?? [])

  const scopeTitle = $derived.by(() => {
    const s = nav.scope
    if (s.kind === 'mine') return '내 할 일'
    if (s.kind === 'team') return '팀 전체'
    if (s.kind === 'unassigned') return '담당자 없음'
    if (s.kind === 'inbox') return '확인해 주세요'
    if (s.kind === 'calendar') return '캘린더'
    if (s.kind === 'teamStatus') return '팀 현황'
    if (s.kind === 'history') return '자동 정리 기록'
    if (s.kind === 'project') return v.projects?.find((p) => p.id === s.id)?.name ?? '프로젝트'
    return v.members?.find((m) => m.id === s.id)?.name ?? '팀원'
  })

  const inScope = $derived.by(() => {
    const s = nav.scope
    return all.filter((t) => {
      if (s.kind === 'mine') return t.assignee_id === me
      if (s.kind === 'unassigned') return !t.assignee_id
      if (s.kind === 'project') return t.project?.id === s.id
      if (s.kind === 'member') return t.assignee_id === s.id
      return true
    })
  })
  const searching = $derived(nav.query.trim().length > 0)
  const visible = $derived(
    inScope.filter((t) => (nav.showDone || searching || t.status !== 'done') && matches(t, nav.query, v.members ?? [])),
  )

  // My list keeps the urgency order; other lists group by status.
  const groups = $derived.by(() => {
    if (nav.scope.kind !== 'mine') return groupByStatus(visible)
    const open = sortTasks(visible.filter((t) => t.status !== 'done'), isNew, v.order ?? [])
    const out: { title: string; items: Ticket[] }[] = []
    for (const t of open) {
      const days = daysUntil(t.due_date)
      const title = days !== null && days <= 1 ? '마감 임박' : isNew.has(t.id) ? '새로 들어온 일' : t.status === 'in_progress' ? '진행 중' : t.status === 'blocked' ? '막힘' : '할 일'
      if (out.at(-1)?.title !== title) out.push({ title, items: [] })
      out.at(-1)!.items.push(t)
    }
    const done = visible.filter((t) => t.status === 'done').sort((a, b) => b.updated_at.localeCompare(a.updated_at))
    if (done.length) out.push({ title: '완료', items: done })
    return out
  })
  const flat = $derived(groups.flatMap((g) => g.items))
  const selected = $derived(all.find((t) => t.id === nav.selected) ?? null)

  const summary = $derived.by(() => {
    const open = inScope.filter((t) => t.status !== 'done')
    const urgent = open.filter((t) => (daysUntil(t.due_date) ?? 99) <= 1).length
    const working = open.filter((t) => t.status === 'in_progress').length
    return [`남은 일 ${open.length}`, working && `진행 중 ${working}`, urgent && `마감 임박 ${urgent}`].filter(Boolean).join(' · ')
  })
  const subtitle = $derived.by(() => {
    switch (nav.scope.kind) {
      case 'inbox': return '대화에서 찾은 일이에요. 맞으면 넣어 주세요.'
      case 'calendar': return '팀 회의 일정 · 한국·일본 시간'
      case 'teamStatus': return '누가 얼마나 맡고 있는지 · 이름을 누르면 그 사람 목록'
      case 'history': return '봇이 스스로 연결하거나 진행·완료 처리한 기록 · 틀리면 되돌리세요'
      case 'mine': return `${todayLabel()} · ${summary}`
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
    if (typing || sheet || !(e.key === 'ArrowDown' || e.key === 'ArrowUp') || !flat.length) return
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
    <Sidebar onsettings={() => (sheet = 'settings')} />

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
              <input bind:this={search} bind:value={nav.query} placeholder="검색 (⌘F)" aria-label="업무 검색" spellcheck="false" />
            </label>
            <button class="toggle" class:on={nav.showDone} onclick={() => (nav.showDone = !nav.showDone)} title="완료한 일 보기 (⇧⌘D)">완료 포함</button>
          {/if}
          <button class="icon" onclick={refresh} class:spin={refreshing} title={synced ? `새로고침 (⌘R) · ${synced} 동기화` : '새로고침 (⌘R)'} aria-label="새로고침">
            <svg viewBox="0 0 16 16" aria-hidden="true"><path d="M13.5 8a5.5 5.5 0 1 1-1.6-3.9M13.5 2.5v3h-3" /></svg>
          </button>
        </div>
      </header>
      {#if v.offline}<p class="offline">오프라인 · 마지막으로 불러온 내용이에요</p>{/if}

      <section class="list">
        {#if nav.scope.kind === 'inbox'}
          <Inbox onerror={(m) => say(m)} onsaved={(m) => say(m, true)} />
        {:else if nav.scope.kind === 'calendar'}
          <Calendar onerror={(m) => say(m)} onsaved={(m) => say(m, true)} />
        {:else if nav.scope.kind === 'teamStatus'}
          <TeamStatus />
        {:else if nav.scope.kind === 'history'}
          <History onerror={(m) => say(m)} onsaved={(m) => say(m, true)} />
        {:else}
          {#each groups as g (g.title)}
            <h2>{g.title} <span>{g.items.length}</span></h2>
            {#each g.items as t (t.id)}
              <div data-task={t.id}>
                <TaskCard ticket={t} isNew={isNew.has(t.id)} selected={nav.selected === t.id} showAssignee={nav.scope.kind !== 'mine' && nav.scope.kind !== 'member'} onselect={() => select(t)} onerror={(m) => say(m)} />
              </div>
            {/each}
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
  {#if sheet}<Sheet mode={sheet} onclose={() => (sheet = null)} ontour={() => ((sheet = null), (tour = true))} />{/if}
{/if}

<style>
  /* The window keeps its traffic lights (hidden-inset title bar); this strip is where you drag it. */
  .titlebar { position: fixed; top: 0; left: 0; right: 0; height: 40px; --wails-draggable: drag; z-index: 5; }
  .layout { height: 100%; display: flex; }
  main { flex: 1; min-width: 0; display: flex; flex-direction: column; }
  header { display: flex; align-items: flex-end; justify-content: space-between; gap: 12px; padding: 44px 18px 8px; }
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
  h2 { margin: 14px 10px 4px; font-size: 11px; font-weight: 700; color: var(--fg3); }
  h2 span { font-weight: 500; margin-left: 2px; }
  .empty { text-align: center; padding: 70px 20px; color: var(--fg3); line-height: 1.6; }
  .empty p { margin: 4px 0; }
  .empty b { color: var(--fg); font-size: 14px; }
  .empty .big { font-size: 32px; }
  footer { position: relative; padding: 8px 14px 12px; border-top: 0.5px solid var(--line); }
  .toast { position: absolute; left: 14px; right: 14px; bottom: calc(100% + 6px); margin: 0; padding: 8px 12px; border-radius: 10px; background: rgba(30, 30, 32, 0.92); color: #fff; font-size: 12px; font-weight: 500; z-index: 4; }
  .toast.inline { position: static; }
  .toast.ok { background: color-mix(in srgb, var(--green) 85%, black); }
</style>
