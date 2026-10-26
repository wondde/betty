<script lang="ts">
  // Settings: everything that isn't day-to-day work, in tabs, so the sidebar stays about tasks.
  // Everyone gets 일반 · 휴지통; admins also get 팀원·프로젝트 · 자동화 (with what Betty did) · GitHub.
  import { onMount } from 'svelte'
  import { Betty } from '../../bindings/github.com/wondde/betty'
  import { app, message, asAdmin, setAdminView } from '../lib/view.svelte'
  import { isWeb } from '../lib/platform'
  import AdminView from './AdminView.svelte'
  import History from './History.svelte'
  import TrashView from './TrashView.svelte'

  let { tab = 'general', onclose, ontour, onerror, onsaved }: {
    tab?: string; onclose: () => void; ontour: () => void; onerror: (m: string) => void; onsaved: (m: string) => void
  } = $props()

  // Messages show inside Settings: the main window's toast sits behind this screen.
  let note = $state<{ text: string; ok: boolean } | null>(null)
  let noteTimer: ReturnType<typeof setTimeout>
  const show = (text: string, ok: boolean) => { note = { text, ok }; clearTimeout(noteTimer); noteTimer = setTimeout(() => (note = null), 4000) }
  const err = (m: string) => (show(m, false), onerror(m))
  const ok = (m: string) => (show(m, true), onsaved(m))

  let current = $state('general')
  $effect.pre(() => { current = tab })
  const tabs = $derived([
    ['general', '일반', ''],
    ...(asAdmin() ? [['team', '팀원·프로젝트', ''], ['automation', '자동화', '베티가 무엇을 자동으로 하는지, 한 일 기록'], ['repos', 'GitHub', '']] : []),
    ['trash', `휴지통${app.view.trash?.length ? ` ${app.view.trash.length}` : ''}`, ''],
  ])
  // A tab someone can't see (or lost access to) falls back to 일반.
  $effect(() => { if (!tabs.some(([id]) => id === current)) current = 'general' })

  let checked = $state(false)
  let notifications = $state(false)
  let autostart = $state(false)
  onMount(async () => {
    notifications = await Betty.NotificationsAllowed()
    autostart = await Betty.Autostart()
  })
  async function run(fn: () => Promise<void>) {
    try {
      await fn()
    } catch (e) {
      err(message(e))
    }
  }
</script>

<svelte:window onkeydown={(e) => e.key === 'Escape' && !document.querySelector('.sheet, .dialog') && onclose()} />

<div class="settings" role="dialog" aria-modal="true" aria-label="설정">
  <div class="drag" aria-hidden="true"></div>
  <aside>
    <h1>설정</h1>
    {#each tabs as [id, label, hint] (id)}
      <button class:on={current === id} onclick={() => (current = id)} title={hint}>{label}</button>
    {/each}
    <span class="spacer"></span>
    <button class="close" onclick={onclose}>닫기 (Esc)</button>
  </aside>

  <section class="body">
    {#if current === 'general'}
      <h2>계정</h2>
      <div class="row"><span>{app.view.member?.name}</span><button onclick={() => run(() => Betty.SignOut())}>로그아웃</button></div>
      <div class="row"><span class="muted">서버</span><span class="muted">{app.view.server?.replace(/^https?:\/\//, '')}</span></div>
      {#if app.view.isAdmin}
        <div class="row">
          <span>관리자 화면<small class="muted"> · 끄면 다른 팀원이 보는 화면이 돼요</small></span>
          <button class="switch" class:on={asAdmin()} role="switch" aria-checked={asAdmin()} aria-label="관리자 화면" onclick={() => setAdminView(!asAdmin())}><i></i></button>
        </div>
      {/if}
      <h2>이 기기</h2>
      <div class="row">
        <span>새 업무 알림</span>
        {#if notifications}<span class="on">켜짐</span>{:else}<button onclick={() => run(async () => { notifications = await Betty.AllowNotifications() })}>허용하기</button>{/if}
      </div>
      {#if !isWeb}
      <div class="row">
        <span>로그인 시 자동 실행</span>
        <button class="switch" class:on={autostart} role="switch" aria-checked={autostart} aria-label="로그인 시 자동 실행"
          onclick={() => run(async () => { await Betty.SetAutostart(!autostart); autostart = !autostart })}><i></i></button>
      </div>
      {/if}
      <h2>도움말</h2>
      <div class="row"><span>Betty 사용법</span><button onclick={ontour}>다시 보기</button></div>
      {#if !isWeb}<div class="row"><span>웹 보드</span><button onclick={() => Betty.OpenBoard()}>열기 ↗</button></div>{/if}
      {#if isWeb}
        <div class="row"><span>맥 앱</span><button onclick={() => Betty.OpenURL(location.origin + '/download')}>받기 ↗</button></div>
        <p class="muted small">Betty {app.view.version}</p>
      {:else}
        <div class="row">
          <span>Betty {app.view.version}<small class="muted">{app.view.update ? ` · 새 버전 ${app.view.update.version}` : checked ? ' · 최신 버전이에요' : ''}</small></span>
          {#if app.view.update}
            <button disabled={app.view.updating} onclick={() => run(() => Betty.InstallUpdate())}>{app.view.updating ? '업데이트하는 중…' : '업데이트'}</button>
          {:else}
            <button onclick={() => run(async () => { await Betty.CheckUpdate(); checked = true })}>업데이트 확인</button>
          {/if}
        </div>
      {/if}
    {:else if current === 'team'}
      <AdminView section="team" onerror={err} onsaved={ok} />
    {:else if current === 'automation'}
      <AdminView section="automation" onerror={err} onsaved={ok} />
      <h2 class="gap">베티가 한 일</h2>
      <p class="muted small">베티가 스스로 GitHub 연결, 진행 반영, 완료 처리한 기록이에요. 틀린 건 되돌리세요. 업무 하나의 기록은 그 업무 상세에도 있어요.</p>
      <History onerror={err} onsaved={ok} />
    {:else if current === 'repos'}
      <AdminView section="repos" onerror={err} onsaved={ok} />
    {:else}
      <p class="muted small">지운 업무예요. 목록에서 빠지고, 베티가 같은 일을 다시 만들지 않아요.</p>
      <TrashView onerror={err} onsaved={ok} />
    {/if}
  </section>
  {#if note}<p class="note" class:ok={note.ok} role="status">{note.text}</p>{/if}
</div>

<style>
  .settings { position: fixed; inset: 0; z-index: 20; display: flex; background: var(--settings-bg, #f5f5f7); color: var(--fg); }
  @media (prefers-color-scheme: dark) { .settings { --settings-bg: #1e1e20; } }
  .drag { position: absolute; top: 0; left: 0; right: 0; height: 40px; --wails-draggable: drag; }
  aside { width: 200px; flex: none; display: flex; flex-direction: column; gap: 2px; padding: var(--top) 10px 12px; border-right: 0.5px solid var(--line); }
  aside h1 { margin: 0 8px 10px; font-size: 18px; }
  aside button { height: 30px; padding: 0 10px; border-radius: 7px; text-align: left; font-size: 13px; }
  aside button:hover { background: var(--fill); }
  aside button.on { background: var(--accent); color: #fff; font-weight: 600; }
  .spacer { flex: 1; }
  .close { color: var(--fg2); }
  .body { flex: 1; min-width: 0; overflow-y: auto; padding: var(--top) 24px 24px; }
  h2 { margin: 18px 0 6px; font-size: 12px; color: var(--fg3); font-weight: 700; }
  h2:first-child { margin-top: 0; }
  h2.gap { margin-top: 26px; }
  .row { display: flex; align-items: center; justify-content: space-between; gap: 10px; padding: 9px 0; border-bottom: 0.5px solid var(--line); }
  .row button { padding: 3px 10px; border-radius: 6px; background: var(--fill); font-weight: 600; font-size: 12px; }
  .row button:hover { background: var(--fill-hover); }
  .muted { color: var(--fg3); }
  .small { font-size: 12px; line-height: 1.5; margin: 4px 0 10px; }
  .on { color: var(--green); font-weight: 600; }
  .note { position: fixed; left: 224px; right: 24px; bottom: 20px; z-index: 21; margin: 0; padding: 9px 14px; border-radius: 10px; background: rgba(30, 30, 32, 0.94); color: #fff; font-size: 12.5px; font-weight: 500; }
  .note.ok { background: color-mix(in srgb, var(--green) 85%, black); }
  .switch { width: 36px; height: 22px; padding: 2px !important; border-radius: 11px !important; background: var(--fill-hover) !important; }
  .switch i { display: block; width: 18px; height: 18px; border-radius: 50%; background: #fff; box-shadow: 0 1px 2px rgba(0,0,0,0.3); transition: transform 0.15s ease; }
  .switch.on { background: var(--green) !important; }
  .switch.on i { transform: translateX(14px); }
</style>
