<script lang="ts">
  // 휴지통: tasks someone removed. They stay out of every list and the automations won't
  // recreate them; 되살리기 puts one back where it was.
  import { Betty } from '../../bindings/github.com/wondde/betty'
  import { app, message } from '../lib/view.svelte'

  let { onerror, onsaved }: { onerror: (m: string) => void; onsaved: (m: string) => void } = $props()
  let busy = $state<string | null>(null)

  const project = (id: string) => app.view.projects?.find((p) => p.id === id)
  const when = (iso: string) => new Date(iso).toLocaleString('ko-KR', { month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit' })

  async function restore(id: string, number: number) {
    busy = id
    try {
      await Betty.Restore(id)
      onsaved(`BT-${number}을(를) 되살렸어요`)
    } catch (e) {
      onerror(message(e))
    } finally {
      busy = null
    }
  }
</script>

<div class="trash">
  {#each app.view.trash ?? [] as t (t.id)}
    {@const p = project(t.project_id)}
    <article>
      <div class="body">
        <b>{t.title}</b>
        <span class="meta">BT-{t.number}{#if p} · <i style:background={p.color}></i>{p.name}{/if} · {t.deleted_by}님이 {when(t.deleted_at)}에 옮김</span>
      </div>
      <button disabled={busy === t.id} onclick={() => restore(t.id, t.number)}>되살리기</button>
    </article>
  {:else}
    <div class="empty">
      <div class="big">🗑</div>
      <p><b>휴지통이 비어 있어요</b></p>
      <p>업무 상세에서 "휴지통으로 옮기기"나 ⌘⌫로 지운 업무가 여기 모여요.</p>
    </div>
  {/each}
</div>

<style>
  .trash { display: flex; flex-direction: column; gap: 2px; padding: 0 8px 16px; }
  article { display: flex; align-items: center; gap: 10px; padding: 8px 10px; border-radius: 9px; }
  article:hover { background: var(--fill); }
  .body { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
  b { font-size: 13px; font-weight: 500; color: var(--fg2); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .meta { display: flex; align-items: center; gap: 4px; font-size: 11px; color: var(--fg3); }
  i { display: inline-block; width: 7px; height: 7px; border-radius: 50%; }
  button { flex: none; height: 26px; padding: 0 12px; border-radius: 7px; background: var(--fill); font-size: 12px; font-weight: 600; }
  button:hover { background: var(--fill-hover); }
  button:disabled { opacity: 0.5; }
  .empty { text-align: center; padding: 80px 20px; color: var(--fg3); line-height: 1.6; }
  .empty .big { font-size: 30px; }
  .empty p { margin: 4px 0; }
  .empty b { color: var(--fg); font-size: 14px; }
</style>
