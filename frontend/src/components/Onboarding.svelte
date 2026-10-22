<script lang="ts">
  // First-run tour after sign-in (Help → "Betty 사용법 다시 보기" shows it again).
  // Ends with the two permissions worth asking for in context: notifications and launch at login.
  import { onMount } from 'svelte'
  import { Betty } from '../../bindings/github.com/wondde/betty'
  import { app, message } from '../lib/view.svelte'

  let { ondone }: { ondone: () => void } = $props()

  let step = $state(0)
  let notifications = $state(false)
  let autostart = $state(false)
  let error = $state('')

  onMount(async () => {
    notifications = await Betty.NotificationsAllowed()
    autostart = await Betty.Autostart()
  })

  const steps = [
    {
      icon: '👋',
      title: `${app.view.member?.name ?? ''}님, 반가워요`,
      body: 'Betty는 BerryTicket에서 나에게 배정된 업무를 모아 보여 주는 앱이에요. Discord에서 정리된 업무가 여기로 자동으로 들어와요.',
    },
    {
      icon: '🗂️',
      title: '급한 일부터 보여 드려요',
      body: '마감이 지났거나 오늘·내일 마감인 일, 새로 들어온 일, 진행 중인 일 순서로 묶여요. 새로 배정된 업무에는 NEW 표시가 붙어요.',
      demo: 'cards',
    },
    {
      icon: '✍️',
      title: '할 일은 이렇게 적어요',
      body: '맨 아래 입력창이나 ⌘N으로 추가해요. 담당자는 @, 프로젝트는 #, 마감은 "내일", "금요일까지", "10/24"처럼 쓰면 돼요.',
      demo: 'compose',
    },
    {
      icon: '⚡',
      title: '어디서든 바로 추가',
      body: '다른 앱을 쓰다가도 ⌥⌘N을 누르면 화면 가운데에 입력창이 떠요. ⌥⌘B를 누르면 이 창이 앞으로 나와요.',
      demo: 'keys',
    },
    {
      icon: '🔔',
      title: '마지막으로 두 가지만',
      body: '새 업무가 배정되면 알림으로 알려 드릴게요. 맥을 켤 때 Betty가 자동으로 열리게 할 수도 있어요.',
      demo: 'permissions',
    },
  ]
  const last = steps.length - 1

  async function allowNotifications() {
    error = ''
    try {
      notifications = await Betty.AllowNotifications()
      if (!notifications) error = '시스템 설정 → 알림 → Betty에서 켤 수 있어요.'
    } catch (e) {
      error = message(e)
    }
  }

  async function toggleAutostart() {
    error = ''
    try {
      await Betty.SetAutostart(!autostart)
      autostart = !autostart
    } catch (e) {
      error = message(e)
    }
  }

  async function finish() {
    await Betty.FinishOnboarding()
    ondone()
  }
</script>

<section class="tour" aria-label="Betty 사용법">
  <div class="dots" aria-hidden="true">
    {#each steps as _, i}<span class:on={i === step}></span>{/each}
  </div>

  {#key step}
    <div class="page">
      <div class="icon" aria-hidden="true">{steps[step].icon}</div>
      <h1>{steps[step].title}</h1>
      <p>{steps[step].body}</p>

      {#if steps[step].demo === 'cards'}
        <div class="demo">
          <div class="dcard"><span class="ring"></span><div><b>로그인 버튼 색 수정</b><small><span class="red">오늘 마감</span> · 링딩</small></div><span class="play">▶</span></div>
          <div class="dcard"><span class="ring"></span><div><b>온보딩 카피 검토 <em>NEW</em></b><small>BT-12 · 나루</small></div><span class="play">▶</span></div>
          <p class="legend">○ 누르면 완료 · ▶ 누르면 시작 · 카드를 누르면 자세히</p>
        </div>
      {:else if steps[step].demo === 'compose'}
        <div class="demo">
          <div class="dfield">＋ <b>@민수</b> 배너 문구 정리 <b>#나루</b> <b>금요일까지</b></div>
          <div class="chips"><span class="to">담당 민수</span><span>#나루</span><span>10/24 마감</span></div>
        </div>
      {:else if steps[step].demo === 'keys'}
        <div class="demo keys">
          <div><kbd>⌥</kbd><kbd>⌘</kbd><kbd>N</kbd><span>빠른 추가</span></div>
          <div><kbd>⌥</kbd><kbd>⌘</kbd><kbd>B</kbd><span>Betty 창 열기</span></div>
          <div><kbd>⌘</kbd><kbd>N</kbd><span>새 할 일</span></div>
          <div><kbd>⌘</kbd><kbd>R</kbd><span>새로고침</span></div>
        </div>
      {:else if steps[step].demo === 'permissions'}
        <div class="demo perms">
          <button class="perm" onclick={allowNotifications} disabled={notifications}>
            <span>🔔 새 업무 알림</span><b>{notifications ? '켜짐' : '허용하기'}</b>
          </button>
          <button class="perm" onclick={toggleAutostart}>
            <span>🚀 로그인 시 자동 실행</span><b>{autostart ? '켜짐' : '꺼짐'}</b>
          </button>
          {#if error}<p class="error">{error}</p>{/if}
        </div>
      {/if}
    </div>
  {/key}

  <div class="nav">
    {#if step > 0}<button class="ghost" onclick={() => step--}>이전</button>{:else}<button class="ghost" onclick={finish}>건너뛰기</button>{/if}
    {#if step < last}
      <button class="primary" onclick={() => step++}>다음</button>
    {:else}
      <button class="primary" onclick={finish}>시작하기</button>
    {/if}
  </div>
</section>

<style>
  .tour { height: 100%; display: flex; flex-direction: column; padding: 56px 28px 22px; }
  .dots { display: flex; gap: 6px; justify-content: center; }
  .dots span { width: 6px; height: 6px; border-radius: 50%; background: var(--fill-hover); transition: width 0.2s ease; }
  .dots span.on { width: 18px; border-radius: 3px; background: var(--accent); }
  .page { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; text-align: center; gap: 10px; animation: in 0.25s ease; }
  @keyframes in { from { transform: translateY(6px); } } /* no opacity: paused animations in a background window must not hide the page */
  .icon { font-size: 44px; }
  h1 { margin: 0; font-size: 20px; letter-spacing: -0.3px; }
  p { margin: 0; color: var(--fg2); line-height: 1.6; font-size: 13px; max-width: 320px; }
  .demo { width: 100%; margin-top: 10px; display: flex; flex-direction: column; gap: 6px; text-align: left; }
  .dcard { display: flex; align-items: center; gap: 10px; padding: 9px 10px; border-radius: 11px; background: var(--card); box-shadow: 0 0 0 0.5px var(--line); }
  .dcard div { flex: 1; display: flex; flex-direction: column; }
  .dcard small { color: var(--fg3); font-size: 11px; }
  .dcard em { font-style: normal; font-size: 9px; padding: 0 5px; border-radius: 4px; background: var(--accent); color: #fff; vertical-align: 2px; }
  .ring { width: 18px; height: 18px; border-radius: 50%; border: 1.4px solid var(--fg3); }
  .play { width: 22px; height: 22px; border-radius: 50%; background: var(--fill); display: grid; place-items: center; font-size: 9px; color: var(--fg2); }
  .red { color: var(--red); font-weight: 600; }
  .legend { font-size: 11.5px; text-align: center; }
  .dfield { padding: 10px 12px; border-radius: 10px; background: var(--fill); box-shadow: inset 0 0 0 1.5px var(--accent); }
  .dfield b { color: var(--accent); font-weight: 600; }
  .chips { display: flex; gap: 5px; }
  .chips span { padding: 2px 8px; border-radius: 6px; font-size: 11px; font-weight: 600; background: var(--fill); color: var(--fg2); }
  .chips .to { color: var(--accent); }
  .keys div { display: flex; align-items: center; gap: 4px; padding: 4px 0; }
  .keys span { margin-left: 10px; color: var(--fg2); }
  kbd { min-width: 24px; padding: 3px 6px; border-radius: 6px; background: var(--card); box-shadow: 0 0 0 0.5px var(--line), 0 1px 0 var(--line); font: 600 12px -apple-system, sans-serif; text-align: center; }
  .perm { display: flex; justify-content: space-between; align-items: center; padding: 11px 12px; border-radius: 11px; background: var(--card); box-shadow: 0 0 0 0.5px var(--line); }
  .perm b { color: var(--accent); }
  .perm:disabled b { color: var(--green); }
  .error { color: var(--red); font-size: 12px; text-align: center; max-width: none; }
  .nav { display: flex; justify-content: space-between; gap: 10px; }
  .nav button { height: 34px; padding: 0 18px; border-radius: 9px; font-weight: 600; }
  .primary { background: var(--accent); color: #fff; margin-left: auto; }
  .ghost { background: var(--fill); }
</style>
