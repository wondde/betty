<script lang="ts">
  // "확인해 주세요": tasks and meetings the assistant found in conversations, and design
  // questions. Approve or skip each; an unconfirmed project has to be chosen first.
  import { Betty } from '../../bindings/github.com/wondde/betty'
  import type { Proposal, MeetingProposal } from '../../bindings/github.com/wondde/betty/internal/berry'
  import { app, message } from '../lib/view.svelte'

  let { onerror, onsaved }: { onerror: (m: string) => void; onsaved: (m: string) => void } = $props()

  const v = $derived(app.view)
  let picks = $state<Record<string, string>>({})
  let busy = $state<string | null>(null)

  const projectName = (id: string) => v.projects?.find((p) => p.id === id)?.name ?? ''
  const memberName = (id?: string | null) => v.members?.find((m) => m.id === id)?.name
  const unconfirmed = (p: { payload: { _routing?: { confirmed?: boolean | null } | null } }) => p.payload?._routing?.confirmed === false
  const md = (d?: string | null) => (d ? `${Number(d.slice(5, 7))}/${Number(d.slice(8, 10))}` : '')

  async function run(id: string, fn: () => Promise<void>, note: string) {
    busy = id
    try {
      await fn()
      onsaved(note)
    } catch (e) {
      onerror(message(e))
    } finally {
      busy = null
    }
  }

  function approveTask(p: Proposal) {
    if (unconfirmed(p) && !picks[p.id]) return onerror('어느 프로젝트 업무인지 먼저 골라 주세요')
    run(p.id, () => Betty.ApproveProposal(p.id, unconfirmed(p) ? picks[p.id] : ''), '할 일에 넣었어요')
  }
  function approveMeeting(m: MeetingProposal) {
    if (unconfirmed(m) && !picks[m.id]) return onerror('어느 프로젝트 회의인지 먼저 골라 주세요')
    if (!m.payload.date) return onerror('날짜가 정해지지 않은 회의는 웹 캘린더에서 날짜를 정해 주세요')
    run(m.id, () => Betty.ApproveMeeting(m.id, unconfirmed(m) ? picks[m.id] : ''), '캘린더에 넣었어요')
  }
</script>

<div class="inbox">
  {#if !(v.proposals?.length || v.meetingProposals?.length || v.designQuestions?.length)}
    <div class="empty">
      <div class="big">✓</div>
      <p><b>지금은 확인할 일이 없어요</b></p>
      <p>대화에서 판단하기 어려운 일이 생기면 여기에 모아 둘게요.</p>
    </div>
  {/if}

  {#each v.proposals ?? [] as p (p.id)}
    <article class="card">
      <div class="tags">
        <span>{unconfirmed(p) ? '프로젝트 미정' : projectName(p.project_id)}</span>
        <span>{p.kind === 'update' ? '원래 할 일 바꾸기' : '새로 찾은 할 일'}</span>
      </div>
      <h3>{p.payload.title}</h3>
      {#if p.payload.description}<p class="desc">{p.payload.description}</p>{/if}
      <p class="meta">
        {memberName(p.payload.assignee_id) ? `${memberName(p.payload.assignee_id)}님이 맡아요` : '아직 맡은 사람은 없어요'}{p.payload.due_date ? ` · ${md(p.payload.due_date)}까지` : ''}
      </p>
      <details>
        <summary>어떤 대화에서 찾았나요?</summary>
        <p>{p.reason}</p>
        {#if p.payload.source_url}<button class="link" onclick={() => Betty.OpenURL(p.payload.source_url)}>대화 보러 가기 ↗</button>{/if}
      </details>
      <div class="actions">
        {#if unconfirmed(p)}
          <select bind:value={picks[p.id]} aria-label="업무 대상 프로젝트">
            <option value="">프로젝트 선택</option>
            {#each v.projects ?? [] as pr (pr.id)}<option value={pr.id}>{pr.name}</option>{/each}
          </select>
        {/if}
        <button class="primary" disabled={busy === p.id} onclick={() => approveTask(p)}>할 일에 넣기</button>
        <button disabled={busy === p.id} onclick={() => run(p.id, () => Betty.DismissProposal(p.id), '이번엔 넘어갔어요')}>넘어가기</button>
      </div>
    </article>
  {/each}

  {#each v.meetingProposals ?? [] as m (m.id)}
    <article class="card">
      <div class="tags">
        <span>{unconfirmed(m) ? '프로젝트 미정' : projectName(m.project_id)}</span>
        <span>{m.kind === 'update' ? '회의 변경' : '새 회의'}{m.payload.status === 'cancelled' ? ' · 취소 제안' : ''}</span>
      </div>
      <h3>📅 {m.payload.title}</h3>
      <p class="meta">{m.payload.date ? `${md(m.payload.date)} ${m.payload.start_time || ''}` : '날짜 미정'}{m.payload.location ? ` · ${m.payload.location}` : ''}</p>
      <p class="desc">{m.reason}</p>
      <div class="actions">
        {#if unconfirmed(m)}
          <select bind:value={picks[m.id]} aria-label="회의 프로젝트">
            <option value="">프로젝트 선택</option>
            {#each v.projects ?? [] as pr (pr.id)}<option value={pr.id}>{pr.name}</option>{/each}
          </select>
        {/if}
        <button class="primary" disabled={busy === m.id} onclick={() => approveMeeting(m)}>캘린더에 넣기</button>
        <button disabled={busy === m.id} onclick={() => run(m.id, () => Betty.DismissMeeting(m.id), '이번엔 넘어갔어요')}>넘어가기</button>
        {#if m.payload.source_url}<button class="link" onclick={() => Betty.OpenURL(m.payload.source_url)}>대화 ↗</button>{/if}
      </div>
    </article>
  {/each}

  {#each v.designQuestions ?? [] as q (q.id)}
    <article class="card">
      <div class="tags"><span>디자인 확인</span></div>
      <h3>{q.question}</h3>
      <p class="desc">{q.reason}</p>
      <div class="actions">
        <button class="primary" onclick={() => Betty.OpenURL(q.url)}>피그마에서 보기 ↗</button>
        <button disabled={busy === q.id} onclick={() => run(q.id, () => Betty.DismissDesignQuestion(q.id), '확인했어요')}>확인했어요</button>
      </div>
    </article>
  {/each}
</div>

<style>
  .inbox { display: flex; flex-direction: column; gap: 10px; padding: 4px 16px 20px; }
  .card { padding: 12px 14px; border-radius: 12px; background: var(--card); box-shadow: 0 0 0 0.5px var(--line); }
  .tags { display: flex; gap: 6px; }
  .tags span { padding: 1px 7px; border-radius: 5px; background: var(--fill); font-size: 11px; font-weight: 600; color: var(--fg2); }
  h3 { margin: 8px 0 4px; font-size: 14px; }
  .desc { margin: 4px 0; font-size: 12.5px; line-height: 1.5; color: var(--fg2); white-space: pre-wrap; }
  .meta { margin: 4px 0; font-size: 12px; color: var(--fg2); }
  details { margin: 6px 0; font-size: 12px; color: var(--fg2); }
  summary { cursor: default; color: var(--fg3); }
  .actions { display: flex; flex-wrap: wrap; align-items: center; gap: 6px; margin-top: 8px; }
  .actions button, .actions select { height: 28px; padding: 0 12px; border-radius: 7px; background: var(--fill); font-size: 12.5px; font-weight: 600; border: none; color: inherit; }
  .actions button:hover { background: var(--fill-hover); }
  .actions .primary { background: var(--accent); color: #fff; }
  .link { background: transparent !important; color: var(--accent) !important; padding: 0 4px !important; }
  button:disabled { opacity: 0.5; }
  .empty { text-align: center; padding: 80px 20px; color: var(--fg3); line-height: 1.6; }
  .empty .big { font-size: 34px; color: var(--green); }
  .empty p { margin: 4px 0; }
  .empty b { color: var(--fg); font-size: 14px; }
</style>
