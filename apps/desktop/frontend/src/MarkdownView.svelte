<script lang="ts">
  import { parseMarkdown, type MarkdownBlock } from './markdown'

  export let text = ''

  $: blocks = parseMarkdown(text || '')

  function headingTag(block: MarkdownBlock): string {
    if (block.type !== 'heading') return 'p'
    return `h${Math.min(6, Math.max(1, block.level))}`
  }
</script>

<div class="markdown-rendered">
  {#each blocks as block, index (index)}
    {#if block.type === 'heading'}
      <svelte:element this={headingTag(block)}>{block.text}</svelte:element>
    {:else if block.type === 'paragraph'}
      <p>{block.text}</p>
    {:else if block.type === 'quote'}
      <blockquote>{block.text}</blockquote>
    {:else if block.type === 'hr'}
      <hr />
    {:else if block.type === 'code'}
      <div class="reader-code-block">
        {#if block.language}
          <div class="reader-code-language">{block.language}</div>
        {/if}
        <pre><code>{block.text}</code></pre>
      </div>
    {:else if block.type === 'ul'}
      <ul>
        {#each block.items as item, itemIndex (itemIndex)}
          <li>{item}</li>
        {/each}
      </ul>
    {:else}
      <ol>
        {#each block.items as item, itemIndex (itemIndex)}
          <li>{item}</li>
        {/each}
      </ol>
    {/if}
  {/each}
</div>
