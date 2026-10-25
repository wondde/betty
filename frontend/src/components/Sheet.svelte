<script lang="ts">
  // Keyboard shortcuts and "Betty에 관하여" as one sheet over the task list.
  import { app } from '../lib/view.svelte'

  let { mode, onclose }: { mode: 'shortcuts' | 'about'; onclose: () => void; ontour?: () => void } = $props()

  const shortcuts = [
    ['⌥⌘N', '어디서든 빠른 추가'],
    ['⌥⌘B', '어디서든 Betty 창 열기'],
    ['⌘N', '새 할 일 (아래 입력창)'],
    ['⇧⌘N', '대화 붙여 넣어 할 일 만들기'],
    ['⌘R', '새로고침'],
    ['⌘F', '업무 검색'],
    ['⌘1 ~ ⌘4', '오늘 · 베티가 찾은 일 · 모든 업무 · 캘린더'],
    ['⌘B', '모든 업무를 목록/보드로 보기'],
    ['↑ ↓', '목록에서 업무 고르기'],
    ['⌘⌫', '고른 업무를 휴지통으로'],
    ['Esc', '검색 지우기 · 상세 닫기'],
    ['⇧⌘D', '최근 완료 보기/숨기기'],
    ['⇧⌘B', '웹 보드 열기'],
    ['⌘,', '설정'],
    ['⌘W', '창 닫기 (Betty는 계속 실행)'],
    ['⌘Q', 'Betty 종료'],
  ]
</script>

<div class="backdrop" role="presentation" onclick={onclose}></div>
<div class="sheet" role="dialog" aria-modal="true" aria-label={mode === 'shortcuts' ? '키보드 단축키' : 'Betty에 관하여'}>
  {#if mode === 'shortcuts'}
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
  .row { display: flex; align-items: center; justify-content: space-between; gap: 10px; padding: 8px 0; border-bottom: 0.5px solid var(--line); }
  .muted { color: var(--fg3); }
  kbd { padding: 2px 8px; border-radius: 6px; background: var(--fill); font: 600 12px -apple-system, sans-serif; }
  .about { text-align: center; padding: 10px 0; }
  .about p { margin: 4px 0; }
  .berry { width: 80px; height: 80px; }
  .done { display: block; margin: 14px 0 0 auto; padding: 5px 16px; border-radius: 8px; background: var(--accent); color: #fff; font-weight: 600; }
</style>
