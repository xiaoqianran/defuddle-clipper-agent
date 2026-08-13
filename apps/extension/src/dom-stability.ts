function snapshot(): string {
  const body = document.body;
  return [location.href, document.title, body?.childElementCount ?? 0, body?.innerText.length ?? 0].join('|');
}

function sleep(ms: number): Promise<void> {
  return new Promise(resolve => window.setTimeout(resolve, ms));
}

export async function waitForDOMStability(maxWaitMs = 3000, intervalMs = 300): Promise<void> {
  const started = Date.now();
  let previous = snapshot();
  let stableRounds = 0;
  while (Date.now() - started < maxWaitMs) {
    await sleep(intervalMs);
    const current = snapshot();
    if (current === previous) {
      stableRounds += 1;
      if (stableRounds >= 2) return;
    } else {
      previous = current;
      stableRounds = 0;
    }
  }
}
