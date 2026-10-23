<script lang="ts">
  import { ask } from '../lib/ask.svelte'
  // Settings tabs (팀원·프로젝트, 자동화, GitHub): automation switches, projects and their Discord channels, people (Discord /
  // work channel / GitHub), and the GitHub repositories the server reads.
  import { Betty } from '../../bindings/github.com/wondde/betty'
  import { app, message } from '../lib/view.svelte'

  // Shown as tabs in Settings: 'team' (people and projects), 'automation', 'repos'.
  let { section, onerror, onsaved }: { section: 'team' | 'automation' | 'repos'; onerror: (m: string) => void; onsaved: (m: string) => void } = $props()

  const v = $derived(app.view)
  let busy = $state(false)

  // Switches the team decided to keep off (no Discord noise). Turning them on asks first.
  const switches: { key: string; label: string; note: string; guarded?: boolean }[] = [
    { key: 'auto_tickets_enabled', label: '대화에서 업무 자동 만들기', note: '명확한 할 일은 바로 만들고, 애매한 건 "베티가 찾은 일"로 보내요.' },
    { key: 'design_review_enabled', label: '피그마 변경 자동 검토', note: '디자인 변경을 기존 업무와 대조해요. 디스코드 알림은 보내지 않아요.' },
    { key: 'assignment_notifications_enabled', label: '배정 알림 (디스코드)', note: '새 배정을 공유 채널에 요약해 보내요.', guarded: true },
    { key: 'reminders_enabled', label: '마감 리마인더 (디스코드)', note: '내일 마감인 업무를 하루 한 번 알려요.', guarded: true },
    { key: 'source_reactions_enabled', label: '원문에 체크 반응 달기', note: '업무로 정리한 디스코드 메시지에 반응을 달아요.', guarded: true },
    { key: 'stale_nudges_enabled', label: '조용한 일 확인 (개인 채널)', note: '5일 넘게 소식 없는 진행 중인 일, 10일 넘게 시작 안 한 일을 평일 낮에 본인 업무 채널에서 한 번씩 물어봐요. 한 사람에 하루 한 번, 같은 일은 일주일에 한 번.', guarded: true },
  ]

  async function run(fn: () => Promise<void>, note: string) {
    busy = true
    try {
      await fn()
      onsaved(note)
      return true
    } catch (e) {
      onerror(message(e))
      return false
    } finally {
      busy = false
    }
  }

  async function toggle(sw: (typeof switches)[number]) {
    const next = !v.settings?.[sw.key]
    if (next && sw.guarded && !(await ask(`"${sw.label}"을(를) 켜면 팀 디스코드에 메시지나 반응이 가요.\n팀에서 꺼 두기로 한 설정이에요. 그래도 켤까요?`, '켜기', true))) return
    run(() => Betty.SetSetting(sw.key, next), `${sw.label}: ${next ? '켰어요' : '껐어요'}`)
  }

  // ---------- projects ----------
  let editingProject = $state<{ id: string; name: string; channels: string[]; color: string } | null>(null)
  const swatches = ['#0a84ff', '#30a46c', '#f76b15', '#d6409f', '#12a594', '#e5484d', '#8e4ec6', '#c99a06', '#3e63dd', '#978365']
  let channelPick = $state('')
  const channelName = (id: string) => {
    const c = v.discordChannels?.find((c) => c.id === id)
    return c ? `${c.category ? c.category + ' / ' : ''}#${c.name}` : id
  }
  const projectChannels = (projectId: string) => (v.projectChannels ?? []).filter((c) => c.project_id === projectId).map((c) => c.channel_id)
  const taken = $derived(new Set((v.projectChannels ?? []).filter((c) => c.project_id !== editingProject?.id).map((c) => c.channel_id)))

  function editProject(id = '') {
    const p = v.projects?.find((p) => p.id === id)
    editingProject = { id, name: p?.name ?? '', channels: id ? projectChannels(id) : [], color: p?.color ?? '' }
    channelPick = ''
  }
  async function saveProject() {
    const p = editingProject!
    if (!p.name.trim()) return onerror('프로젝트 이름을 적어 주세요')
    if (await run(() => Betty.SaveProject(p.id, p.name.trim(), p.channels, p.color), p.id ? '프로젝트를 바꿨어요' : '프로젝트를 만들었어요')) editingProject = null
  }

  // ---------- members ----------
  let editingMember = $state<{ id: string; name: string; discord_id: string; channel_id: string; github_login: string; area: string } | null>(null)
  const areaOptions = [['', '영역 미정'], ['engineering', '개발'], ['design', '디자인'], ['marketing', '마케팅']]
  const githubOf = (memberId: string) => v.github?.identities?.find((i) => i.member_id === memberId)?.login ?? ''
  function editMember(id = '') {
    const m = v.members?.find((m) => m.id === id)
    editingMember = { id, name: m?.name ?? '', discord_id: m?.discord_id ?? '', channel_id: m?.channel_id ?? '', github_login: id ? githubOf(id) : '', area: m?.area ?? '' }
  }
  async function saveMember() {
    const m = editingMember!
    if (!m.name.trim()) return onerror('이름을 적어 주세요')
    if (await run(() => Betty.SaveMember(m.id, { name: m.name.trim(), discord_id: m.discord_id.trim(), channel_id: m.channel_id.trim(), github_login: m.github_login.trim(), area: m.area }), m.id ? '팀원 정보를 바꿨어요' : '팀원을 추가했어요')) editingMember = null
  }
  // Area can be set right in the list, without opening the editor.
  function setArea(id: string, area: string) {
    const m = v.members!.find((x) => x.id === id)!
    run(() => Betty.SaveMember(id, { name: m.name, discord_id: m.discord_id ?? '', channel_id: m.channel_id ?? '', github_login: githubOf(id), area }), `${m.name}: ${areaOptions.find((o) => o[0] === area)![1]}`)
  }

  // ---------- repos ----------
  let repoName = $state('')
  let repoProject = $state('')
  // Accepts owner/name or a pasted GitHub link (https://github.com/owner/name/…, .git, git@github.com:owner/name).
  function repoFrom(input: string): string | null {
    const t = input.trim().replace(/^git@github\.com:/i, '').replace(/^https?:\/\//i, '').replace(/^(www\.)?github\.com\//i, '').replace(/\.git$/i, '')
    const [owner, name] = t.split(/[/?#]/).filter(Boolean)
    return owner && name && /^[\w.-]+$/.test(owner) && /^[\w.-]+$/.test(name) ? `${owner}/${name.replace(/\.git$/i, '')}` : null
  }
  async function addRepo() {
    const repo = repoFrom(repoName)
    if (!repo) return onerror('저장소 주소를 확인해 주세요. 예: blueberry-team/naru 또는 GitHub 링크')
    if (!repoProject) return onerror('저장소를 연결할 프로젝트를 골라 주세요')
    if (await run(() => Betty.AddRepo(repo, repoProject), `${repo}를 연결했어요. 몇 분 안에 불러와요`)) repoName = ''
  }
  const projectName = (id: string) => v.projects?.find((p) => p.id === id)?.name ?? ''
</script>

<div class="admin">
  {#if section === 'automation'}
  <section>
    <h2>자동화</h2>
    {#each switches as sw (sw.key)}
      <div class="row">
        <div class="text"><b>{sw.label}</b><small>{sw.note}{sw.guarded ? ' · 팀에서 꺼 두기로 한 설정' : ''}</small></div>
        <button class="switch" class:on={v.settings?.[sw.key]} role="switch" aria-checked={!!v.settings?.[sw.key]} aria-label={sw.label} disabled={busy} onclick={() => toggle(sw)}><i></i></button>
      </div>
    {/each}
  </section>
  {/if}

  {#if section === 'team'}
  <section>
    <div class="head"><h2>팀원</h2><button class="add" onclick={() => editMember()}>＋ 팀원</button></div>
    <p class="none">영역을 정하면 사이드바의 팀이 개발·디자인·마케팅으로 나뉘어요.</p>
    {#each v.members ?? [] as m (m.id)}
      <div class="row">
        <span class="avatar">{Array.from(m.name)[0]}</span>
        <div class="text"><b>{m.name}</b><small>{m.discord_id ? 'Discord 연결됨' : 'Discord ID 없음'}{githubOf(m.id) ? ` · GitHub @${githubOf(m.id)}` : ''}{m.channel_id ? ` · 업무 채널 ${channelName(m.channel_id)}` : ''}</small></div>
        <select class="area" value={m.area ?? ''} disabled={busy} onchange={(e) => setArea(m.id, (e.target as HTMLSelectElement).value)} aria-label="{m.name} 영역">
          {#each areaOptions as [value, label]}<option {value}>{label}</option>{/each}
        </select>
        <button class="edit" onclick={() => editMember(m.id)}>편집</button>
      </div>
    {/each}
  </section>

  <section>
    <div class="head"><h2>프로젝트</h2><button class="add" onclick={() => editProject()}>＋ 프로젝트</button></div>
    {#each v.projects ?? [] as p (p.id)}
      <div class="row">
        <i class="dot" style:background={p.color}></i>
        <div class="text"><b>{p.name}</b><small>{projectChannels(p.id).map(channelName).join(', ') || '대화 채널 없음'}</small></div>
        <button class="edit" onclick={() => editProject(p.id)}>편집</button>
      </div>
    {/each}
  </section>

  {/if}

  {#if section === 'repos'}
  <section>
    <h2>GitHub 저장소</h2>
    <p class="none">연결한 저장소의 이슈·PR을 업무와 맞춰 봐요. 같은 업무 같다는 제안은 업무 상세에 작게 나와요.</p>
    {#each v.github?.repos ?? [] as r (r.id)}
      <div class="row">
        <div class="text"><b>{r.full_name}</b><small>{projectName(r.project_id)} · {r.error ? `오류: ${r.error}` : r.synced_at ? `마지막 동기화 ${new Date(r.synced_at).toLocaleString('ko-KR', { month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit' })}` : '아직 불러오지 않았어요'}</small></div>
        <button class="edit danger" disabled={busy} onclick={async () => (await ask(`${r.full_name} 연결을 해제할까요?\n연결된 업무 링크도 사라져요.`, '해제', true)) && run(() => Betty.RemoveRepo(r.id), '저장소 연결을 해제했어요')}>해제</button>
      </div>
    {:else}
      <p class="none">연결된 저장소가 없어요.</p>
    {/each}
    <div class="addrepo">
      <input bind:value={repoName} placeholder="blueberry-team/lingding" aria-label="저장소 이름" spellcheck="false" />
      <select bind:value={repoProject} aria-label="연결할 프로젝트">
        <option value="">프로젝트 선택</option>
        {#each v.projects ?? [] as p (p.id)}<option value={p.id}>{p.name}</option>{/each}
      </select>
      <button class="add" disabled={busy} onclick={addRepo}>연결</button>
    </div>
    <p class="none">서버의 GitHub 읽기 권한으로 가져와요. 저장소에 무언가를 쓰지는 않아요.</p>
  </section>
  {/if}
</div>

{#if editingProject}
  <div class="backdrop" role="presentation" onclick={() => (editingProject = null)}></div>
  <div class="sheet" role="dialog" aria-modal="true" aria-label="프로젝트">
    <h1>{editingProject.id ? '프로젝트 편집' : '새 프로젝트'}</h1>
    <label>이름<input bind:value={editingProject.name} /></label>
    {#if editingProject.id}
      <p class="lbl">색</p>
      <div class="swatches" role="radiogroup" aria-label="프로젝트 색">
        {#each swatches as c (c)}<button role="radio" aria-checked={editingProject.color === c} aria-label={c} class:on={editingProject.color === c} style:background={c} onclick={() => (editingProject!.color = c)}></button>{/each}
      </div>
    {/if}
    <p class="lbl">대화를 모을 디스코드 채널</p>
    <div class="chips">
      {#each editingProject.channels as id (id)}
        <span class="chip">{channelName(id)}<button aria-label="{channelName(id)} 빼기" onclick={() => (editingProject!.channels = editingProject!.channels.filter((c) => c !== id))}>✕</button></span>
      {:else}<span class="none">채널 없음</span>{/each}
    </div>
    <select bind:value={channelPick} onchange={() => { if (channelPick && !editingProject!.channels.includes(channelPick)) editingProject!.channels = [...editingProject!.channels, channelPick]; channelPick = '' }}>
      <option value="">＋ 채널 추가</option>
      {#each (v.discordChannels ?? []).filter((c) => !taken.has(c.id) && !editingProject!.channels.includes(c.id)) as c (c.id)}<option value={c.id}>{channelName(c.id)}</option>{/each}
    </select>
    <p class="none">한 채널은 한 프로젝트에만 연결할 수 있어요.</p>
    <div class="actions"><button onclick={() => (editingProject = null)}>취소</button><button class="primary" disabled={busy} onclick={saveProject}>저장</button></div>
  </div>
{/if}

{#if editingMember}
  <div class="backdrop" role="presentation" onclick={() => (editingMember = null)}></div>
  <div class="sheet" role="dialog" aria-modal="true" aria-label="팀원">
    <h1>{editingMember.id ? '팀원 편집' : '새 팀원'}</h1>
    <label>이름 (디스코드에서 쓰는 이름)<input bind:value={editingMember.name} /></label>
    <label>Discord 사용자 ID<input bind:value={editingMember.discord_id} placeholder="숫자 ID (개발자 모드에서 복사)" spellcheck="false" /></label>
    <label>개인 업무 채널 ID<input bind:value={editingMember.channel_id} placeholder="선택" spellcheck="false" /></label>
    <label>GitHub 아이디<input bind:value={editingMember.github_login} placeholder="선택" spellcheck="false" /></label>
    <label>영역<select bind:value={editingMember.area}>{#each areaOptions as [value, label]}<option {value}>{label}</option>{/each}</select></label>
    <p class="none">Discord ID가 있어야 Betty와 웹에 Discord로 로그인할 수 있어요.</p>
    <div class="actions"><button onclick={() => (editingMember = null)}>취소</button><button class="primary" disabled={busy} onclick={saveMember}>저장</button></div>
  </div>
{/if}

<svelte:window onkeydown={(e) => e.key === 'Escape' && (editingProject || editingMember) && (e.stopPropagation(), (editingProject = null), (editingMember = null))} />

<style>
  .admin { display: flex; flex-direction: column; gap: 18px; padding: 0 18px 24px; max-width: 760px; }
  section { display: flex; flex-direction: column; }
  .head { display: flex; align-items: center; justify-content: space-between; }
  h2 { margin: 0 0 6px; font-size: 13px; }
  .row { display: flex; align-items: center; gap: 10px; padding: 8px 4px; border-bottom: 0.5px solid var(--line); }
  .text { flex: 1; min-width: 0; display: flex; flex-direction: column; }
  .text b { font-size: 13px; font-weight: 600; }
  .text small { font-size: 11.5px; color: var(--fg3); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .dot { width: 10px; height: 10px; border-radius: 50%; flex: none; }
  .avatar { width: 24px; height: 24px; border-radius: 50%; background: var(--fill-hover); display: grid; place-items: center; font-size: 11px; font-weight: 700; color: var(--fg2); flex: none; }
  .switch { width: 38px; height: 22px; padding: 2px; border-radius: 11px; background: var(--fill-hover); flex: none; }
  .switch i { display: block; width: 18px; height: 18px; border-radius: 50%; background: #fff; box-shadow: 0 1px 2px rgba(0,0,0,0.3); transition: transform 0.15s ease; }
  .switch.on { background: var(--green); }
  .switch.on i { transform: translateX(16px); }
  .edit, .add { height: 26px; padding: 0 10px; border-radius: 7px; background: var(--fill); font-size: 12px; font-weight: 600; flex: none; }
  .edit:hover, .add:hover { background: var(--fill-hover); }
  .danger { color: var(--red); }
  .addrepo { display: flex; gap: 6px; margin-top: 8px; }
  .addrepo input, .addrepo select { height: 28px; padding: 0 8px; border: none; outline: none; border-radius: 7px; background: var(--fill); font: inherit; font-size: 12.5px; color: inherit; }
  .addrepo input { flex: 1; }
  .none { font-size: 11.5px; color: var(--fg3); margin: 6px 4px; }
  .backdrop { position: fixed; inset: 0; background: rgba(0, 0, 0, 0.18); z-index: 10; }
  .sheet { position: fixed; left: 50%; top: 52px; transform: translateX(-50%); width: 420px; max-height: calc(100% - 72px); overflow-y: auto; z-index: 11; padding: 18px; border-radius: 14px; background: var(--sheet, #fff); box-shadow: 0 18px 50px rgba(0,0,0,0.25), 0 0 0 0.5px var(--line); display: flex; flex-direction: column; gap: 10px; }
  @media (prefers-color-scheme: dark) { .sheet { --sheet: #2c2c2e; } }
  .sheet h1 { margin: 0; font-size: 17px; }
  .sheet label { display: flex; flex-direction: column; gap: 4px; font-size: 12px; color: var(--fg2); }
  .sheet input, .sheet select { font: inherit; font-size: 13px; color: var(--fg); border: none; outline: none; background: var(--fill); border-radius: 7px; padding: 6px 8px; }
  .sheet input:focus, .sheet select:focus { box-shadow: inset 0 0 0 1.5px var(--accent); }
  select.area { height: 26px; padding: 0 6px; border: none; outline: none; border-radius: 7px; background: var(--fill); font: inherit; font-size: 12px; color: inherit; }
  .lbl { margin: 0; font-size: 12px; color: var(--fg2); }
  .swatches { display: flex; gap: 6px; }
  .swatches button { width: 22px; height: 22px; border-radius: 50%; }
  .swatches button.on { box-shadow: 0 0 0 2px var(--sheet, #fff), 0 0 0 4px currentColor; color: var(--fg); }
  .chips { display: flex; flex-wrap: wrap; gap: 5px; }
  .chip { display: inline-flex; align-items: center; gap: 4px; padding: 2px 4px 2px 8px; border-radius: 6px; background: var(--fill); font-size: 12px; }
  .chip button { color: var(--fg3); padding: 0 4px; }
  .actions { display: flex; justify-content: flex-end; gap: 8px; }
  .actions button { height: 30px; padding: 0 14px; border-radius: 8px; background: var(--fill); font-weight: 600; }
  .actions .primary { background: var(--accent); color: #fff; }
  button:disabled { opacity: 0.5; }
</style>
