<script lang="ts">
  // 사용법 둘러보기: after the first-run tour, a step-by-step walk over the real screen. Each
  // step dims everything but one part of the window (found by its data-coach name) and says
  // what it is for. Steps whose part isn't on screen (nothing to confirm, not an admin…) are
  // skipped. Same in the app and on the web; seen once per device (coach:v1).
  import { onMount } from 'svelte'
  import { app } from '../lib/view.svelte'
  import { isWeb } from '../lib/platform'

  let { ondone }: { ondone: () => void } = $props()

  type Step = { target?: string; title: string; body: string; admin?: boolean }
  const all: Step[] = [
    { title: 'Betty 둘러보기', body: '화면 곳곳이 무엇을 하는지 1분 안에 보여 드릴게요. 언제든 Esc로 그만둘 수 있어요.' },
    { target: 'now', title: '지금 할 일', body: '마감이 가깝거나 하고 있는 내 일이 맨 위에 와요. 오른쪽 시작·완료 버튼으로 바로 바꿔요. 줄을 누르면 오른쪽에 자세한 내용이 열려요.' },
    { target: 'ask', title: '확인해 주세요', body: '베티가 대화에서 찾은 내 일처럼, 내 대답이 필요한 것만 여기 모여요.' },
    { target: 'long', title: '길게 보는 일', body: '개선·조사·운영처럼 조금씩 진행하는 일은 따로 모아요. 오래 그대로여도 괜찮아요. 업무 상세의 기간에서 바꿀 수 있어요.' },
    { target: 'composer', title: '할 일 추가', body: '여기에 적으면 바로 내 일이 돼요. @이름으로 담당자, #프로젝트, "내일"·"금요일까지"처럼 마감도 한 줄에 같이 적을 수 있어요.' },
    { target: 'nav-inbox', title: '베티가 찾은 일', body: '디스코드 대화에서 베티가 찾은 할 일 후보예요. 맞으면 넣고, 아니면 넘기면 돼요.' },
    { target: 'nav-team', title: '우리 팀', body: '같은 팀 사람들이 무엇을 하는지 보여요. 다른 팀은 한 줄 요약으로만 보여요.' },
    { target: 'nav-all', title: '모든 업무', body: '팀 전체 업무를 목록이나 보드로 봐요. 아무도 안 맡은 일, 길게 보는 일만 골라 볼 수도 있어요.' },
    { target: 'view-switch', title: '관리자 · 팀원 화면', body: '관리자는 팀원·프로젝트·GitHub·자동화 설정까지 보여요. 팀원 쪽을 누르면 다른 팀원이 보는 화면 그대로 볼 수 있어요.', admin: true },
    { target: 'settings', title: '설정', body: isWeb ? '알림과 휴지통이 여기 있어요. 이 안내는 설정 → 일반 → Betty 사용법에서 다시 볼 수 있어요.' : '알림, 자동 실행, 휴지통이 여기 있어요. 이 안내는 설정 → 일반 → Betty 사용법에서 다시 볼 수 있어요.' },
  ]

  const el = (name?: string) => (name ? (document.querySelector(`[data-coach="${name}"]`) as HTMLElement | null) : null)
  const steps = $derived(all.filter((s) => (!s.admin || app.view.isAdmin) && (!s.target || el(s.target))))
  let i = $state(0)
  let rect = $state<DOMRect | null>(null)
  const step = $derived(steps[Math.min(i, steps.length - 1)])

  function place() {
    const target = el(step?.target)
    if (!target) return void (rect = null)
    target.scrollIntoView({ block: 'nearest' })
    rect = target.getBoundingClientRect()
  }
  $effect(() => { void step; requestAnimationFrame(place) })
  onMount(() => {
    const re = () => place()
    window.addEventListener('resize', re)
    return () => window.removeEventListener('resize', re)
  })

  const next = () => (i < steps.length - 1 ? (i += 1) : ondone())
  const back = () => (i = Math.max(0, i - 1))
  function key(e: KeyboardEvent) {
    if (e.key === 'Escape') { e.preventDefault(); ondone() }
    else if (e.key === 'ArrowRight' || e.key === 'Enter') { e.preventDefault(); next() }
    else if (e.key === 'ArrowLeft') { e.preventDefault(); back() }
  }

  // The card sits beside the highlighted part where there is room, else below or above it.
  const PAD = 6, W = 300
  const card = $derived.by(() => {
    if (!rect) return { left: `calc(50% - ${W / 2}px)`, top: '34%' }
    const vw = window.innerWidth, vh = window.innerHeight
    if (rect.right + 16 + W < vw && rect.width < vw * 0.4) return { left: `${rect.right + 16}px`, top: `${Math.min(Math.max(12, rect.top), vh - 200)}px` }
    const left = `${Math.min(Math.max(12, rect.left), vw - W - 12)}px`
    return rect.bottom + 190 < vh ? { left, top: `${rect.bottom + 12}px` } : { left, top: `${Math.max(12, rect.top - 182)}px` }
  })
</script>

<svelte:window onkeydown={key} />

{#if step}
  <div class="coach" role="dialog" aria-modal="true" aria-label="Betty 사용법">
    <!-- Blocks clicks on the page while the tour runs. -->
    <div class="shield" class:dim={!rect} aria-hidden="true"></div>
    {#if rect}
      <div class="spot" aria-hidden="true" style:left="{rect.left - PAD}px" style:top="{rect.top - PAD}px" style:width="{rect.width + PAD * 2}px" style:height="{rect.height + PAD * 2}px"></div>
    {/if}
    <div class="card" style:left={card.left} style:top={card.top} style:width="{W}px">
      <p class="count">{i + 1} / {steps.length}</p>
      <h2>{step.title}</h2>
      <p>{step.body}</p>
      <div class="actions">
        <button class="skip" onclick={ondone}>그만 보기</button>
        <span></span>
        {#if i > 0}<button onclick={back}>이전</button>{/if}
        <button class="primary" onclick={next}>{i === steps.length - 1 ? '시작하기' : '다음'}</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .coach { position: fixed; inset: 0; z-index: 40; }
  .shield { position: absolute; inset: 0; }
  .shield.dim { background: rgba(0, 0, 0, 0.5); }
  .spot { position: absolute; border-radius: 12px; box-shadow: 0 0 0 9999px rgba(0, 0, 0, 0.5), 0 0 0 2px var(--accent); transition: all 0.2s ease; pointer-events: none; }
  .card { position: absolute; padding: 14px 16px 12px; border-radius: 14px; background: var(--sheet, #fff); color: var(--fg); box-shadow: 0 18px 50px rgba(0, 0, 0, 0.3); transition: top 0.2s ease, left 0.2s ease; }
  @media (prefers-color-scheme: dark) { .card { --sheet: #2c2c2e; } }
  .count { margin: 0 0 2px; font-size: 11px; font-weight: 600; color: var(--accent); }
  h2 { margin: 0 0 6px; font-size: 15px; }
  p { margin: 0; font-size: 12.5px; line-height: 1.55; color: var(--fg2); }
  .actions { display: flex; align-items: center; gap: 6px; margin-top: 12px; }
  .actions span { flex: 1; }
  .actions button { height: 28px; padding: 0 12px; border-radius: 8px; font-size: 12.5px; font-weight: 600; background: var(--fill); }
  .actions .primary { background: var(--accent); color: #fff; }
  .actions .skip { padding: 0 4px; background: none; color: var(--fg3); font-weight: 500; }
</style>
