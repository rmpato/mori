/* mori — the little bit of behaviour this page needs, and nothing more. */

// Copy the install command.
for (const button of document.querySelectorAll('button.copy')) {
  button.addEventListener('click', async () => {
    const source = document.querySelector(button.dataset.copy);
    if (!source) return;
    try {
      await navigator.clipboard.writeText(source.textContent.trim());
    } catch {
      // Clipboard access can be refused; select the text so it can be
      // copied by hand rather than leaving the button looking broken.
      const range = document.createRange();
      range.selectNodeContents(source);
      const selection = window.getSelection();
      selection.removeAllRanges();
      selection.addRange(range);
      return;
    }
    const was = button.textContent;
    button.textContent = 'copied';
    button.classList.add('done');
    setTimeout(() => {
      button.textContent = was;
      button.classList.remove('done');
    }, 1600);
  });
}

// Install tabs.
for (const tabs of document.querySelectorAll('[data-tabs]')) {
  const buttons = [...tabs.querySelectorAll('[data-tab]')];
  const panels = [...tabs.querySelectorAll('[data-panel]')];

  const show = (name) => {
    for (const b of buttons) b.setAttribute('aria-selected', String(b.dataset.tab === name));
    for (const p of panels) p.hidden = p.dataset.panel !== name;
  };

  for (const b of buttons) {
    b.addEventListener('click', () => show(b.dataset.tab));
    b.addEventListener('keydown', (e) => {
      const step = e.key === 'ArrowRight' ? 1 : e.key === 'ArrowLeft' ? -1 : 0;
      if (!step) return;
      e.preventDefault();
      const next = buttons[(buttons.indexOf(b) + step + buttons.length) % buttons.length];
      next.focus();
      show(next.dataset.tab);
    });
  }
}
