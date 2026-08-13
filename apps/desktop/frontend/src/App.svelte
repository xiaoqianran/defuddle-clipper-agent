<script lang="ts">
  import { onMount } from 'svelte'
  import MarkdownView from './MarkdownView.svelte'

  type AIStatus = 'pending' | 'ok' | 'failed' | 'disabled' | 'unknown'

  type CaptureSummary = {
    captureId: string
    capturedAt: string
    title: string
    url: string
    site?: string
    language?: string
    hasAnalysis: boolean
    hasNote: boolean
    aiStatus?: AIStatus | string
  }

  type BrowserState = {
    active: boolean
    page: { url: string; title: string; observedAt: string }
    updatedAt: string
  }

  type Policy = {
    revision?: number
    autoCapture: boolean
    archiveAll: boolean
    captureDelayMs: number
    domainAllowlist?: string[]
    domainDenylist?: string[]
  }

  type SensorView = {
    connected: boolean
    seenAt?: string
    queueLength: number
    lastError?: string
  }

  type Snapshot = {
    connected: boolean
    browser: BrowserState
    captures: CaptureSummary[]
    policy?: Policy
    sensor?: SensorView
  }

  type CaptureView = {
    packet: {
      captureId: string
      capturedAt: string
      source: { url: string; title: string; site?: string; author?: string; published?: string; language?: string; description?: string }
      content: { markdown: string; html?: string }
      metadata?: Record<string, unknown>
      media?: Record<string, unknown>
    }
    sourceMarkdown: string
    analysis?: unknown
    note?: string
    aiStatus?: AIStatus | string
  }

  let captures: CaptureSummary[] = []
  let browser: BrowserState = { active: false, page: { url: '', title: '', observedAt: '' }, updatedAt: '' }
  let selectedId = ''
  let view: CaptureView | null = null
  let query = ''
  let followBrowser = true
  let connected = false
  let error = ''
  let policy: Policy = { autoCapture: true, archiveAll: true, captureDelayMs: 1200 }
  let sensor: SensorView = { connected: false, queueLength: 0 }
  let savingPolicy = false
  let loading = false
  let refreshing = false
  let reprocessing = false
  let readerMode: 'content' | 'transcript' = 'content'
  let seenStatusKey = ''

  $: normalizedQuery = query.trim().toLowerCase()
  $: filteredCaptures = normalizedQuery
    ? captures.filter(item => `${item.title} ${item.url} ${item.site ?? ''}`.toLowerCase().includes(normalizedQuery))
    : captures
  $: transcript = typeof view?.packet?.media?.transcript === 'string' ? String(view.packet.media.transcript) : ''
  $: analysisText = view?.aiStatus === 'ok' && view?.analysis ? JSON.stringify(view.analysis, null, 2) : ''
  $: aiStatus = view?.aiStatus ?? ''
  $: canReprocess = Boolean(view && (aiStatus === 'ok' || aiStatus === 'failed'))

  function api(): any {
    return (window as any).go?.main?.App
  }

  function formatTime(value: string): string {
    if (!value) return ''
    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
  }

  function host(value: string): string {
    try { return new URL(value).hostname } catch { return value }
  }

  function statusKey(aiStatus?: string, hasAnalysis?: boolean): string {
    return `${aiStatus ?? ''}:${hasAnalysis ? '1' : '0'}`
  }

  async function loadCapture(captureId: string, manual = true, force = false): Promise<void> {
    if (!captureId || (!force && captureId === selectedId && view)) return
    if (manual) followBrowser = false
    const switching = captureId !== selectedId || !view
    if (switching) loading = true
    error = ''
    try {
      const next = await api().ReadCapture(captureId) as CaptureView
      selectedId = captureId
      view = next
      const item = captures.find(entry => entry.captureId === captureId)
      seenStatusKey = item ? statusKey(item.aiStatus, item.hasAnalysis) : statusKey(next.aiStatus, Boolean(next.analysis))
      if (switching) readerMode = 'content'
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause)
    } finally {
      loading = false
    }
  }

  async function refresh(): Promise<void> {
    if (refreshing) return
    refreshing = true
    try {
      const next = await api().GetSnapshot(250) as Snapshot
      captures = next.captures ?? []
      browser = next.browser ?? browser
      if (next.policy) policy = next.policy
      if (next.sensor) sensor = next.sensor
      connected = true
      error = ''

      if (selectedId && view) {
        const item = captures.find(entry => entry.captureId === selectedId)
        if (item && statusKey(item.aiStatus, item.hasAnalysis) !== seenStatusKey) {
          await loadCapture(selectedId, false, true)
        }
      }

      if (followBrowser && browser.active && browser.page.url) {
        const match = captures.find(item => item.url === browser.page.url)
        if (match && match.captureId !== selectedId) {
          await loadCapture(match.captureId, false)
        }
      } else if (!selectedId && captures.length > 0) {
        await loadCapture(captures[0].captureId, false)
      }
    } catch (cause) {
      connected = false
      error = cause instanceof Error ? cause.message : String(cause)
    } finally {
      refreshing = false
    }
  }

  async function reprocess(): Promise<void> {
    if (!selectedId || reprocessing) return
    reprocessing = true
    error = ''
    try {
      await api().ReprocessCapture(selectedId)
      captures = captures.map(item => item.captureId === selectedId
        ? { ...item, aiStatus: 'pending', hasAnalysis: false }
        : item)
      await loadCapture(selectedId, false, true)
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause)
    } finally {
      reprocessing = false
    }
  }

  async function putPolicy(next: Policy): Promise<void> {
    if (savingPolicy) return
    savingPolicy = true
    const previous = policy
    policy = next
    try {
      policy = await api().PutPolicy(next) as Policy
      error = ''
    } catch (cause) {
      policy = previous
      error = cause instanceof Error ? cause.message : String(cause)
    } finally {
      savingPolicy = false
    }
  }

  onMount(() => {
    void refresh()
    const runtime = (window as any).runtime
    const off = typeof runtime?.EventsOn === 'function' ? runtime.EventsOn('dca:changed', () => void refresh()) : null
    const timer = window.setInterval(() => void refresh(), 10_000)
    return () => {
      if (typeof off === 'function') off()
      window.clearInterval(timer)
    }
  })
</script>

<div class="shell">
  <header class="topbar">
    <div class="brand">
      <div class="logo">D</div>
      <div>
        <strong>Defuddle Browser Mirror</strong>
        <span>local web inbox</span>
      </div>
    </div>

    <div class="browser-now" title={browser.page.url || 'No active browser page'}>
      <span class:online={connected} class="dot"></span>
      <div>
        <small>{connected ? (sensor.connected ? 'Browser' : 'Waiting for extension') : 'Agent offline'}</small>
        <div>{browser.page.title || (connected ? (sensor.connected ? 'Waiting for a page…' : 'Load the extension once') : 'Start Defuddle.exe')}</div>
      </div>
    </div>

    <div class="top-actions">
      <label class="follow-toggle">
        <input type="checkbox" checked={policy.autoCapture} disabled={savingPolicy}
          on:change={() => void putPolicy({ ...policy, autoCapture: !policy.autoCapture })} />
        <span>Auto Capture</span>
      </label>
      <label class="follow-toggle">
        <input type="checkbox" checked={policy.archiveAll} disabled={savingPolicy}
          on:change={() => void putPolicy({ ...policy, archiveAll: !policy.archiveAll })} />
        <span>Archive All</span>
      </label>
      <label class="follow-toggle">
        <input type="checkbox" bind:checked={followBrowser} />
        <span>Follow Browser</span>
      </label>
      <button class="icon-button" on:click={() => void refresh()} title="Refresh">↻</button>
    </div>
  </header>

  {#if error}
    <div class="errorbar">{error}</div>
  {/if}

  <main class="workspace">
    <aside class="history-panel">
      <div class="panel-head">
        <div>
          <h2>History</h2>
          <span>{captures.length} mirrored pages</span>
        </div>
      </div>
      <div class="search-wrap">
        <input bind:value={query} placeholder="Search title, site or URL" aria-label="Search history" />
      </div>
      <div class="history-list">
        {#if filteredCaptures.length === 0}
          <div class="empty small">Load the extension once, then browse. Pages appear here automatically.</div>
        {:else}
          {#each filteredCaptures as item (item.captureId)}
            <button
              class:selected={item.captureId === selectedId}
              class:pending={item.aiStatus === 'pending'}
              class="history-item"
              on:click={() => void loadCapture(item.captureId, true)}
            >
              <div class="history-title">{item.title || 'Untitled'}</div>
              <div class="history-meta">
                <span>{item.site || host(item.url)}</span>
                <span class="history-meta-right">
                  {#if item.aiStatus}
                    <span class="history-status {item.aiStatus}">{item.aiStatus}</span>
                  {/if}
                  <time>{formatTime(item.capturedAt)}</time>
                </span>
              </div>
              <div class="history-url">{item.url}</div>
            </button>
          {/each}
        {/if}
      </div>
    </aside>

    <section class="reader-panel">
      {#if view}
        <div class="reader-head">
          <div class="reader-title-block">
            <div class="eyebrow">{view.packet.source.site || host(view.packet.source.url)}</div>
            <h1>{view.packet.source.title}</h1>
            <a href={view.packet.source.url} target="_blank" rel="noreferrer">{view.packet.source.url}</a>
            <div class="byline">
              {#if view.packet.source.author}<span>{view.packet.source.author}</span>{/if}
              <span>{formatTime(view.packet.capturedAt)}</span>
              {#if view.packet.source.language}<span>{view.packet.source.language}</span>{/if}
            </div>
          </div>
          {#if transcript}
            <div class="reader-tabs">
              <button class:active={readerMode === 'content'} on:click={() => readerMode = 'content'}>Content</button>
              <button class:active={readerMode === 'transcript'} on:click={() => readerMode = 'transcript'}>Transcript</button>
            </div>
          {/if}
        </div>
        <div class="reader-scroll">
          {#if loading}
            <div class="empty">Loading…</div>
          {:else if readerMode === 'transcript' && transcript}
            <article class="document"><pre>{transcript}</pre></article>
          {:else}
            <article class="document">
              <MarkdownView text={view.sourceMarkdown} />
            </article>
          {/if}
        </div>
      {:else}
        <div class="empty hero-empty">
          <div class="empty-icon">↘</div>
          <h1>Your browser, mirrored locally</h1>
          <p>Keep this window open and browse in Chrome or Edge. The extension is only a sensor — this is the reader.</p>
        </div>
      {/if}
    </section>

    <aside class="insight-panel">
      <div class="panel-head insights-title">
        <div>
          <h2>AI / Notes</h2>
          <span>derived, optional</span>
        </div>
        {#if canReprocess}
          <button class="text-button" disabled={reprocessing} on:click={() => void reprocess()}>
            {reprocessing ? 'Reprocessing…' : 'Reprocess'}
          </button>
        {/if}
      </div>
      {#if aiStatus === 'pending'}
        <div class="analyzing">Analyzing…</div>
      {/if}
      {#if view?.note}
        <section class="insight-section">
          <h3>Note</h3>
          <pre class="note-text">{view.note}</pre>
        </section>
      {/if}
      {#if analysisText}
        <section class="insight-section">
          <h3>Structured analysis</h3>
          <pre class="analysis-text">{analysisText}</pre>
        </section>
      {/if}
      {#if view && aiStatus !== 'pending' && !view.note && !analysisText}
        <div class="empty small side-empty">This page is archived. AI is disabled or has not processed it yet.</div>
      {:else if !view}
        <div class="empty small side-empty">Select a mirrored page to see its notes and analysis.</div>
      {/if}
    </aside>
  </main>

  <footer class="statusbar">
    <span>{connected ? '● local agent' : '○ agent offline'}</span>
    <span>{sensor.connected ? `sensor live · queue ${sensor.queueLength}` : 'sensor offline'}</span>
    {#if browser.page.url}<span class="active-url">active: {browser.page.url}</span>{/if}
    <span>{followBrowser ? 'following browser' : 'history inspection'}</span>
  </footer>
</div>
