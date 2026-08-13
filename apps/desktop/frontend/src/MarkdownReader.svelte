<script lang="ts">
  export let text = ''

  type Block = {
    type: 'heading' | 'paragraph' | 'code' | 'quote' | 'ul' | 'ol' | 'hr'
    level?: number
    text?: string
    items?: string[]
    language?: string
  }

  function beginsBlock(line: string): boolean {
    return /^```/.test(line)
      || /^#{1,6}\s+/.test(line)
      || /^>\s?/.test(line)
      || /^\s*[-*+]\s+/.test(line)
      || /^\s*\d+\.\s+/.test(line)
      || /^\s*(?:-{3,}|\*{3,}|_{3,})\s*$/.test(line)
  }

  function parseMarkdown(input: string): Block[] {
    const lines = input.replace(/\r\n/g, '\n').split('\n')
    const blocks: Block[] = []
    let i = 0

    while (i < lines.length) {
      const line = lines[i]
      if (!line.trim()) {
        i += 1
        continue
      }

      const fence = line.match(/^```\s*([^\s`]*)/)
      if (fence) {
        const body: string[] = []
        i += 1
        while (i < lines.length && !/^```\s*$/.test(lines[i])) {
          body.push(lines[i])
          i += 1
        }
        if (i < lines.length) i += 1
        blocks.push({ type: 'code', text: body.join('\n'), language: fence[1] || '' })
        continue
      }

      const heading = line.match(/^(#{1,6})\s+(.+)$/)
      if (heading) {
        blocks.push({ type: 'heading', level: heading[1].length, text: heading[2].trim() })
        i += 1
        continue
      }

      if (/^\s*(?:-{3,}|\*{3,}|_{3,})\s*$/.test(line)) {
        blocks.push({ type: 'hr' })
        i += 1
        continue
      }

      if (/^>\s?/.test(line)) {
        const quote: string[] = []
        while (i < lines.length && /^>\s?/.test(lines[i])) {
          quote.push(lines[i].replace(/^>\s?/, ''))
          i += 1
        }
        blocks.push({ type: 'quote', text: quote.join('\n') })
        continue
      }

      if (/^\s*[-*+]\s+/.test(line)) {
        const items: string[] = []
        while (i < lines.length && /^\s*[-*+]\s+/.test(lines[i])) {
          items.push(lines[i].replace(/^\s*[-*+]\s+/, '').trim())
          i += 1
        }
        blocks.push({ type: 'ul', items })
        continue
      }

      if (/^\s*\d+\.\s+/.test(line)) {
        const items: string[] = []
        while (i < lines.length && /^\s*\d+\.\s+/.test(lines[i])) {
          items.push(lines[i].replace(/^\s*\d+\.\s+/, '').trim())
          i += 1
        }
        blocks.push({ type: 'ol', items })
        continue
      }

      const paragraph: string[] = [line.trim()]
      i += 1
      while (i < lines.length && lines[i].trim() && !beginsBlock(lines[i])) {
        paragraph.push(lines[i].trim())
        i += 1
      }
      blocks.push({ type: 'paragraph', text: paragraph.join(' ') })
    }

    return blocks
  }

  $: blocks = parseMarkdown(text)
</script>

<div class="markdown-document">
  {#each blocks as block}
    {#if block.type === 'heading'}
      {#if block.level === 1}<h1>{block.text}</h1>
      {:else if block.level === 2}<h2>{block.text}</h2>
      {:else if block.level === 3}<h3>{block.text}</h3>
      {:else if block.level === 4}<h4>{block.text}</h4>
      {:else if block.level === 5}<h5>{block.text}</h5>
      {:else}<h6>{block.text}</h6>{/if}
    {:else if block.type === 'paragraph'}
      <p>{block.text}</p>
    {:else if block.type === 'quote'}
      <blockquote>{block.text}</blockquote>
    {:else if block.type === 'code'}
      <div class="code-block">
        {#if block.language}<div class="code-language">{block.language}</div>{/if}
        <pre><code>{block.text}</code></pre>
      </div>
    {:else if block.type === 'ul'}
      <ul>{#each block.items ?? [] as item}<li>{item}</li>{/each}</ul>
    {:else if block.type === 'ol'}
      <ol>{#each block.items ?? [] as item}<li>{item}</li>{/each}</ol>
    {:else if block.type === 'hr'}
      <hr />
    {/if}
  {/each}
</div>

<style>
  .markdown-document { color: #dedee2; font-size: 15px; line-height: 1.82; }
  h1, h2, h3, h4, h5, h6 { color: #f0f0f2; line-height: 1.3; letter-spacing: -.015em; }
  h1 { margin: 2.2em 0 .8em; font-size: 2em; }
  h2 { margin: 2em 0 .75em; font-size: 1.55em; padding-bottom: .35em; border-bottom: 1px solid #303036; }
  h3 { margin: 1.7em 0 .65em; font-size: 1.25em; }
  h4, h5, h6 { margin: 1.5em 0 .55em; font-size: 1.05em; }
  p { margin: 0 0 1.2em; white-space: pre-wrap; overflow-wrap: anywhere; }
  ul, ol { margin: 0 0 1.25em; padding-left: 1.7em; }
  li { margin: .32em 0; padding-left: .2em; }
  blockquote { margin: 1.35em 0; padding: .75em 1em; border-left: 3px solid #555560; background: #202024; color: #bcbcc5; white-space: pre-wrap; }
  hr { border: 0; border-top: 1px solid #33333a; margin: 2em 0; }
  .code-block { position: relative; margin: 1.4em 0; border: 1px solid #303037; border-radius: 10px; background: #111114; overflow: hidden; }
  .code-language { padding: 6px 10px; border-bottom: 1px solid #29292f; color: #777783; font: 10px ui-monospace, SFMono-Regular, Consolas, monospace; text-transform: uppercase; }
  pre { margin: 0; padding: 15px 17px; overflow: auto; white-space: pre; }
  code { color: #d7d7dd; font: 12px/1.65 ui-monospace, SFMono-Regular, Consolas, monospace; }
</style>
