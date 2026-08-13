type Block =
  | { type: 'heading'; level: number; text: string }
  | { type: 'paragraph'; text: string }
  | { type: 'quote'; text: string }
  | { type: 'code'; text: string; language: string }
  | { type: 'ul' | 'ol'; items: string[] }
  | { type: 'hr' }

function beginsBlock(line: string): boolean {
  return /^```/.test(line)
    || /^#{1,6}\s+/.test(line)
    || /^>\s?/.test(line)
    || /^\s*[-*+]\s+/.test(line)
    || /^\s*\d+\.\s+/.test(line)
    || /^\s*(?:-{3,}|\*{3,}|_{3,})\s*$/.test(line)
}

function parse(input: string): Block[] {
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

function appendTextElement(parent: HTMLElement, tag: string, text: string): HTMLElement {
  const element = document.createElement(tag)
  element.textContent = text
  parent.appendChild(element)
  return element
}

function render(pre: HTMLPreElement): void {
  if (pre.dataset.dcaRendered === 'true') return
  const transcriptActive = [...document.querySelectorAll('.reader-tabs button.active')]
    .some(button => button.textContent?.trim() === 'Transcript')
  if (transcriptActive) return

  const text = pre.textContent ?? ''
  if (!text.trim()) return

  const root = document.createElement('div')
  root.className = 'markdown-rendered'
  root.dataset.source = 'safe-text-dom'

  for (const block of parse(text)) {
    if (block.type === 'heading') {
      appendTextElement(root, `h${Math.min(6, Math.max(1, block.level))}`, block.text)
    } else if (block.type === 'paragraph') {
      appendTextElement(root, 'p', block.text)
    } else if (block.type === 'quote') {
      appendTextElement(root, 'blockquote', block.text)
    } else if (block.type === 'hr') {
      root.appendChild(document.createElement('hr'))
    } else if (block.type === 'code') {
      const wrapper = document.createElement('div')
      wrapper.className = 'reader-code-block'
      if (block.language) {
        const language = appendTextElement(wrapper, 'div', block.language)
        language.className = 'reader-code-language'
      }
      const codePre = document.createElement('pre')
      const code = document.createElement('code')
      code.textContent = block.text
      codePre.appendChild(code)
      wrapper.appendChild(codePre)
      root.appendChild(wrapper)
    } else {
      const list = document.createElement(block.type)
      for (const item of block.items) appendTextElement(list, 'li', item)
      root.appendChild(list)
    }
  }

  pre.dataset.dcaRendered = 'true'
  pre.replaceWith(root)
}

function enhance(): void {
  document.querySelectorAll<HTMLPreElement>('.reader-scroll .document > pre').forEach(render)
}

export function installMarkdownEnhancer(): () => void {
  enhance()
  const observer = new MutationObserver(() => queueMicrotask(enhance))
  observer.observe(document.body, { childList: true, subtree: true })
  return () => observer.disconnect()
}
