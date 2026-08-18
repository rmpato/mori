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

/* The topbar stays out of the way until the masthead is gone, and marks the
   section you're in. Same behaviour as tuki's — the two pages should move the
   same way, even where they don't look the same. */
(function topbar() {
  const bar = document.getElementById('topbar');
  const masthead = document.querySelector('.masthead');
  if (!bar || !masthead || !('IntersectionObserver' in window)) return;

  new IntersectionObserver(
    ([entry]) => bar.classList.toggle('show', !entry.isIntersecting),
    { rootMargin: '-70px 0px 0px 0px' }
  ).observe(masthead);

  const links = new Map();
  for (const a of bar.querySelectorAll("a[href^='#']")) {
    const id = a.getAttribute('href').slice(1);
    if (id !== 'top') links.set(id, a);
  }

  const sections = [...links.keys()]
    .map((id) => document.getElementById(id))
    .filter(Boolean);
  if (!sections.length) return;

  // Whichever section crosses the middle band of the viewport is the current
  // one. aria-current rather than a class, so the marker is in the semantics
  // and not only in the paint — tuki's page does exactly the same.
  const visible = new Map();
  const observer = new IntersectionObserver(
    (entries) => {
      for (const e of entries) visible.set(e.target.id, e.isIntersecting);

      const current = sections.find((el) => visible.get(el.id));
      for (const a of links.values()) a.removeAttribute('aria-current');
      if (current && links.has(current.id)) {
        links.get(current.id).setAttribute('aria-current', 'true');
      }
    },
    { rootMargin: '-45% 0px -45% 0px' }
  );
  for (const section of sections) observer.observe(section);
})();
