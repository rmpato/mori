# Change the website

The site is `docs/`, served by GitHub Pages from `main`. There is no build
step: three files and an image.

| file | what |
| --- | --- |
| `docs/index.html` | the page |
| `docs/style.css` | all of the styling |
| `docs/site.js` | the copy button and the install tabs |
| `docs/screenshot.png` | the terminal capture — see [screenshot.md](screenshot.md) |
| `docs/favicon.svg` | a sprout |
| `docs/.nojekyll` | stops Pages running Jekyll over it |

`docs/DESIGN.md` and `docs/WEB.md` also live there. They're for the repo, not
the site; Pages just serves them as text, which is harmless.

## Work on it locally

```sh
cd docs && python3 -m http.server 8731
open http://127.0.0.1:8731/
```

Hard-reload after CSS edits, or add `?v=2` — the browser caches `style.css`
aggressively enough to waste ten minutes of your life.

## Rules the page is meant to keep

- **No network requests.** No webfonts, no analytics, no CDN, nothing. Every
  typeface is a system stack. A page about a private local journal should not
  be phoning anywhere, and the only way to be certain is to have nothing to
  phone.
- **Both themes.** The palette is defined light-first with a
  `prefers-color-scheme: dark` block. Check both — the quickest way is to
  temporarily delete the dark block and reload.
- **Less text, not more.** The page had 900 words once and read like a wall.
  It's around 200 now. The README carries the detail; the page carries the
  feeling and the install line.
- **The terminal blocks are HTML, not images.** They stay sharp, they're
  selectable, they adapt to the theme, and they can't silently go stale the
  way a screenshot can. Only the hero is a real capture.

## Deploy

Push to `main`. Pages redeploys on its own.

```sh
git push origin main
sleep 40
gh api repos/rmpato/mori/pages --jq '.status'    # "built"
curl -s -o /dev/null -w '%{http_code}\n' https://rmpato.github.io/mori/
```

## Check

- Loads at <https://rmpato.github.io/mori/>.
- `style.css`, `site.js`, `favicon.svg` and `screenshot.png` all return 200.
- Copy button copies; the install tabs switch.
- Nothing in the network panel but this origin.
