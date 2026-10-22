# Logo and brand assets

- `logo.svg` — the app icon (master). `<x>`: XML brackets around the crossing
  strokes of the "X" in XSLT; the right bracket is Go's cyan.
- `logo-glyph.svg` — the same mark without the background, the foreground
  layer of `desktop/build/appicon.icon` (macOS Icon Composer).
- `logo.png` — `logo.svg` at 512 px.
- `social-preview.svg` / `.png` — the 1280×640 image GitHub shows when the
  repository is shared (upload it under *Settings → General → Social preview*;
  GitHub has no API for it).

The desktop icons are generated from `desktop/build/appicon.png` (the master at
1024 px) by `wails3 generate icons`; on a Mac with Xcode that also builds
`Assets.car` from `appicon.icon`. To re-render the PNGs after editing an SVG,
any renderer that keeps transparency works; `@resvg/resvg-js` is what was used:

```bash
npm i --no-save @resvg/resvg-js
node -e "const {Resvg}=require('@resvg/resvg-js'),fs=require('fs');fs.writeFileSync('logo.png',new Resvg(fs.readFileSync('logo.svg'),{fitTo:{mode:'width',value:512}}).render().asPng())"
```
