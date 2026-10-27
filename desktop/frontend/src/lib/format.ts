// formatXml pretty-prints an XML/HTML string for display. It collapses
// whitespace that sits purely between tags, then re-indents one level per
// nesting depth. Text content inside an element (>text<) is left untouched.
export function formatXml(xml: string): string {
  if (!xml) return "";
  const PAD = "  ";

  // Pull off a leading XML declaration / PI so it stays on its own line.
  let head = "";
  let body = xml;
  const decl = body.match(/^\s*<\?[^>]*\?>\s*/);
  if (decl) {
    head = decl[0].trim() + "\n";
    body = body.slice(decl[0].length);
  }

  // Collapse whitespace-only gaps between tags, then break adjacent tags apart.
  body = body.replace(/>\s+</g, "><").replace(/></g, ">\n<").trim();

  let depth = 0;
  const out: string[] = [];
  for (const raw of body.split("\n")) {
    const line = raw.trim();
    if (!line) continue;
    const isClose = /^<\/[^>]+>/.test(line);
    const isSelf = /\/>\s*$/.test(line) || /^<\?/.test(line) || /^<!--/.test(line) || /^<!/.test(line);
    // A line that opens and closes the same element (<a>text</a>) keeps depth.
    const isOpenClose = /^<[^!?][^>]*>.*<\/[^>]+>\s*$/.test(line);

    if (isClose) depth = Math.max(0, depth - 1);
    out.push(PAD.repeat(depth) + line);
    if (!isClose && !isSelf && !isOpenClose && /^<[^!?]/.test(line)) depth += 1;
  }
  return head + out.join("\n");
}

// formatJson pretty-prints a JSON string for display with two-space
// indentation, or returns null when the text is not valid JSON (the caller then
// shows the raw text unchanged). It goes through JSON.parse, so it is for
// display only: integers beyond 2^53 lose precision, a number's spelling is
// normalised (1.0 becomes 1, 1e2 becomes 100), and duplicate object keys
// collapse to the last one (keys otherwise keep their insertion order, except
// that integer-like keys are listed first, ascending).
export function formatJson(text: string): string | null {
  if (!text.trim()) return null;
  try {
    return JSON.stringify(JSON.parse(text), null, 2);
  } catch {
    return null;
  }
}
