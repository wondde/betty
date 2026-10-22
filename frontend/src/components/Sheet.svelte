<script lang="ts">
  // Settings (⌘,), keyboard shortcuts and "Betty에 관하여" as one sheet over the task list.
  import { onMount } from 'svelte'
  import { Betty } from '../../bindings/github.com/wondde/betty'
  import { app, message } from '../lib/view.svelte'

  let { mode, onclose, ontour }: { mode: 'settings' | 'shortcuts' | 'about'; onclose: () => void; ontour: () => void } = $props()

  let notifications = $state(false)
  let autostart = $state(false)
  let error = $state('')

  onMount(async () => {
    notifications = await Betty.NotificationsAllowed()
    autostart = await Betty.Autostart()
  })

  async function run(fn: () => Promise<void>) {
    error = ''
    try {
      await fn()
    } catch (e) {
      error = message(e)
    }
  }

  const shortcuts = [
    ['⌥⌘N', '어디서든 빠른 추가'],
    ['⌥⌘B', '어디서든 Betty 창 열기'],
    ['⌘N', '새 할 일 (아래 입력창)'],
    ['⌘R', '새로고침'],
    ['⌘F', '업무 검색'],
    ['⌘1 ~ ⌘8', '내 할 일 · 팀 전체 · 확인해 주세요 · 캘린더 · 팀 현황 · 자동 정리 기록 · 개발 현황 · 팀·연결 설정'],
    ['↑ ↓', '목록에서 업무 고르기'],
    ['Esc', '검색 지우기 · 상세 닫기'],
    ['⇧⌘D', '최근 완료 보기/숨기기'],
    ['⇧⌘B', '웹 보드 열기'],
    ['⌘,', '설정'],
    ['⌘W', '창 닫기 (Betty는 계속 실행)'],
    ['⌘Q', 'Betty 종료'],
  ]
</script>

<div class="backdrop" role="presentation" onclick={onclose}></div>
<div class="sheet" role="dialog" aria-modal="true" aria-label={mode === 'settings' ? '설정' : mode === 'shortcuts' ? '키보드 단축키' : 'Betty에 관하여'}>
  {#if mode === 'settings'}
    <h1>설정</h1>
    <h2>계정</h2>
    <div class="row"><span>{app.view.member?.name}</span><button onclick={() => run(() => Betty.SignOut())}>로그아웃</button></div>
    <div class="row"><span class="muted">서버</span><span class="muted">{app.view.server?.replace(/^https?:\/\//, '')}</span></div>
    <h2>일반</h2>
    <div class="row">
      <span>새 업무 알림</span>
      {#if notifications}<span class="on">켜짐</span>{:else}<button onclick={() => run(async () => { notifications = await Betty.AllowNotifications() })}>허용하기</button>{/if}
    </div>
    <div class="row">
      <span>로그인 시 자동 실행</span>
      <button class="switch" class:on={autostart} role="switch" aria-checked={autostart} aria-label="로그인 시 자동 실행"
        onclick={() => run(async () => { await Betty.SetAutostart(!autostart); autostart = !autostart })}><i></i></button>
    </div>
    <h2>도움말</h2>
    <div class="row"><span>Betty 사용법</span><button onclick={ontour}>다시 보기</button></div>
    <div class="row"><span>웹 보드</span><button onclick={() => Betty.OpenBoard()}>열기 ↗</button></div>
    {#if error}<p class="error">{error}</p>{/if}
  {:else if mode === 'shortcuts'}
    <h1>키보드 단축키</h1>
    {#each shortcuts as [keys, what]}
      <div class="row"><span>{what}</span><kbd>{keys}</kbd></div>
    {/each}
  {:else}
    <div class="about">
      <img class="berry" src="/betty-icon.png" alt="" width="80" height="80" />
      <h1>Betty</h1>
      <p class="muted">버전 {app.view.version}</p>
      <p>블루베리 팀의 BerryTicket을 위한 맥 앱</p>
    </div>
  {/if}
  <button class="done" onclick={onclose}>완료</button>
</div>

<style>
  .backdrop { position: fixed; inset: 0; background: rgba(0, 0, 0, 0.18); z-index: 10; }
  .sheet {
    position: fixed; left: 16px; right: 16px; top: 52px; max-height: calc(100% - 72px); overflow-y: auto; z-index: 11;
    padding: 18px 18px 14px; border-radius: 14px; background: var(--sheet, #fff); color: var(--fg);
    box-shadow: 0 18px 50px rgba(0, 0, 0, 0.25), 0 0 0 0.5px var(--line); animation: drop 0.18s ease;
  }
  @media (prefers-color-scheme: dark) { .sheet { --sheet: #2c2c2e; } }
  @keyframes drop { from { transform: translateY(-8px); } }
  h1 { margin: 0 0 6px; font-size: 17px; }
  h2 { margin: 14px 0 4px; font-size: 11px; color: var(--fg3); font-weight: 700; }
  .row { display: flex; align-items: center; justify-content: space-between; gap: 10px; padding: 8px 0; border-bottom: 0.5px solid var(--line); }
  .row button { padding: 3px 10px; border-radius: 6px; background: var(--fill); font-weight: 600; font-size: 12px; }
  .row button:hover { background: var(--fill-hover); }
  .muted { color: var(--fg3); }
  .on { color: var(--green); font-weight: 600; }
  .switch { width: 36px; height: 22px; padding: 2px !important; border-radius: 11px !important; background: var(--fill-hover) !important; position: relative; }
  .switch i { display: block; width: 18px; height: 18px; border-radius: 50%; background: #fff; box-shadow: 0 1px 2px rgba(0,0,0,0.3); transition: transform 0.15s ease; }
  .switch.on { background: var(--green) !important; }
  .switch.on i { transform: translateX(14px); }
  kbd { padding: 2px 8px; border-radius: 6px; background: var(--fill); font: 600 12px -apple-system, sans-serif; }
  .about { text-align: center; padding: 10px 0; }
  .about p { margin: 4px 0; }
  .berry { width: 80px; height: 80px; }
  .error { color: var(--red); font-size: 12px; }
  .done { display: block; margin: 14px 0 0 auto; padding: 5px 16px; border-radius: 8px; background: var(--accent); color: #fff; font-weight: 600; }
</style>
