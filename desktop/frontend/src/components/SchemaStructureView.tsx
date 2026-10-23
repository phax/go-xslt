import { createContext, useContext, useEffect, useMemo, useState } from "react";
import { schemaStructure, type SchemaModel, type SMAttr, type SMNode, type SMType } from "../lib/api";
import { DiagnosticsList } from "./ResultParts";
import { useT } from "../i18n";
import { viewsMessages } from "../i18n/messages/views";

/** The resolved structure of an XSD — across every file it imports/includes —
 * as an expandable tree: global elements, types and attributes on the left,
 * the selected one on the right. Elements show their type, occurrence bounds
 * and (expanded) their attributes and content model; sequence / choice / all
 * groups are drawn as such. Types are looked up by id from a flat table, so a
 * type used a thousand times is sent once and expanded on demand; a type that
 * contains itself is cut at the repeat. Read-only for now. */

type Sel = { kind: "element" | "type" | "attribute"; key: string };

interface Ctx {
  model: SchemaModel;
  isOpen: (path: string, dflt: boolean, depth: number) => boolean;
  toggle: (path: string, now: boolean) => void;
  goto: (typeId: string) => void;
}
const C = createContext<Ctx | null>(null);

const XS = "http://www.w3.org/2001/XMLSchema";

export function SchemaStructureView({
  schemas,
  baseDir,
  projectDir,
  version,
}: {
  schemas: string[];
  baseDir: string;
  projectDir: string;
  version: string;
}) {
  const [model, setModel] = useState<SchemaModel | null>(null);
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState("");
  const [sel, setSel] = useState<Sel | null>(null);
  const [list, setList] = useState<"element" | "type" | "attribute">("element");
  const [filter, setFilter] = useState("");
  const [toggles, setToggles] = useState<Record<string, boolean>>({});
  const [level, setLevel] = useState(0);
  const tr = useT(viewsMessages);

  const key = JSON.stringify([schemas, baseDir, projectDir, version]);
  useEffect(() => {
    let live = true;
    setBusy(true);
    const t = window.setTimeout(() => {
      schemaStructure({ schemas, baseDir, projectDir, version })
        .then((m) => {
          if (!live) return;
          setModel(m);
          setFailure("");
        })
        .catch((e) => live && setFailure(String(e)))
        .finally(() => live && setBusy(false));
    }, 300);
    return () => {
      live = false;
      window.clearTimeout(t);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key]);

  const elements = useMemo(() => (model?.elements ?? []).filter((e): e is SMNode => !!e), [model]);
  const types = model?.types ?? {};
  const typeIds = model?.globalTypes ?? [];
  const attrs = model?.attributes ?? [];

  // Pick something to show as soon as a model arrives (or the selection vanished).
  useEffect(() => {
    if (!model) return;
    const stillThere =
      sel && (sel.kind === "element" ? elements.some((e) => e.name === sel.key) : sel.kind === "type" ? !!types[sel.key] : attrs.some((a) => a.name === sel.key));
    if (stillThere) return;
    if (elements.length) setSel({ kind: "element", key: elements[0].name ?? "" });
    else if (typeIds.length) {
      setList("type");
      setSel({ kind: "type", key: typeIds[0] });
    } else setSel(null);
  }, [model]); // eslint-disable-line react-hooks/exhaustive-deps

  const ctx: Ctx | null = model
    ? {
        model,
        isOpen: (path, dflt, depth) => (path in toggles ? toggles[path] : dflt || depth < level),
        toggle: (path, now) => setToggles((t) => ({ ...t, [path]: !now })),
        goto: (id) => {
          if (!types[id]) return;
          setList("type");
          setSel({ kind: "type", key: id });
          setFilter("");
        },
      }
    : null;

  const q = filter.trim().toLowerCase();
  const shownElements = elements.filter((e) => !q || (e.name ?? "").toLowerCase().includes(q));
  const shownTypes = typeIds.filter((id) => !q || (types[id]?.name ?? id).toLowerCase().includes(q));
  const shownAttrs = attrs.filter((a) => !q || a.name.toLowerCase().includes(q));
  const diags = model?.diagnostics ?? [];
  const hasErrors = diags.some((d) => d.severity === "error");

  return (
    <div className="struct">
      <div className="struct-side">
        <div className="struct-tabs">
          {(
            [
              ["element", tr("tabElements"), elements.length],
              ["type", tr("tabTypes"), typeIds.length],
              ["attribute", tr("tabAttributes"), attrs.length],
            ] as const
          ).map(([k, label, n]) => (
            <button key={k} className={list === k ? "on" : ""} onClick={() => setList(k)}>
              {label} <span className="count">{n}</span>
            </button>
          ))}
        </div>
        <input className="struct-filter" placeholder={tr("filterPlaceholder")} value={filter} onChange={(e) => setFilter(e.target.value)} />
        <ul className="struct-list">
          {list === "element" &&
            shownElements.map((e) => (
              <li key={e.name} className={sel?.kind === "element" && sel.key === e.name ? "sel" : ""} onClick={() => setSel({ kind: "element", key: e.name ?? "" })} title={e.ns}>
                <span className="glyph el">E</span>
                {e.name}
              </li>
            ))}
          {list === "type" &&
            shownTypes.map((id) => (
              <li key={id} className={sel?.kind === "type" && sel.key === id ? "sel" : ""} onClick={() => setSel({ kind: "type", key: id })} title={types[id]?.ns}>
                <span className={`glyph ${types[id]?.simple ? "st" : "ct"}`}>{types[id]?.simple ? "S" : "C"}</span>
                {types[id]?.name || id}
              </li>
            ))}
          {list === "attribute" &&
            shownAttrs.map((a) => (
              <li key={a.name} className={sel?.kind === "attribute" && sel.key === a.name ? "sel" : ""} onClick={() => setSel({ kind: "attribute", key: a.name })} title={a.ns}>
                <span className="glyph at">@</span>
                {a.name}
              </li>
            ))}
        </ul>
      </div>
      <div className="struct-main">
        <div className="struct-bar">
          <button onClick={() => { setToggles({}); setLevel(8); }}>{tr("expandAll")}</button>
          <button onClick={() => { setToggles({}); setLevel(0); }}>{tr("collapse")}</button>
          <span className="spacer" />
          {busy && <span className="meta">{tr("readingSchema")}</span>}
          {!busy && model && <span className="meta">{tr("summary", { types: Object.keys(types).length, elements: elements.length })}</span>}
        </div>
        {failure && <div className="struct-empty">{failure}</div>}
        {diags.length > 0 && <DiagnosticsList diagnostics={diags} />}
        <div className="struct-tree">
          {ctx && sel && (
            <C.Provider value={ctx}>
              {sel.kind === "element" && elements.filter((e) => e.name === sel.key).map((e) => <ElementRow key={sel.key} node={e} path="root" depth={0} trail={[]} root />)}
              {sel.kind === "type" && types[sel.key] && <TypeRoot type={types[sel.key]!} />}
              {sel.kind === "attribute" && attrs.filter((a) => a.name === sel.key).map((a) => <AttrRow key={a.name} a={a} />)}
            </C.Provider>
          )}
          {!sel && !busy && !hasErrors && !failure && <div className="struct-empty">{tr("emptySchema")}</div>}
          {!sel && hasErrors && <div className="struct-empty">{tr("schemaBroken")}</div>}
        </div>
      </div>
    </div>
  );
}

function occ(min: number, max: number): string {
  const hi = max < 0 ? "∞" : String(max);
  return `${min}..${hi}`;
}

function Chevron({ open, show }: { open: boolean; show: boolean }) {
  return <span className={`chev ${show ? (open ? "open" : "") : "none"}`}>▸</span>;
}

function TypeChip({ id, name }: { id?: string; name?: string }) {
  const c = useContext(C)!;
  const tr = useT(viewsMessages);
  if (!id) return null;
  const t = c.model.types?.[id];
  const label = name || t?.name || tr("anonymous");
  const navigable = !!t && !t.builtin && !!(t.name || t.model || t.attributes?.length);
  return (
    <span
      className={`tchip ${t?.simple ? "st" : "ct"} ${navigable && t?.name ? "link" : ""}`}
      title={t?.ns ? `{${t.ns}}${label}` : label}
      onClick={(e) => {
        if (navigable && t?.name) {
          e.stopPropagation();
          c.goto(id);
        }
      }}
    >
      {label}
    </span>
  );
}

function GroupIcon({ kind }: { kind: string }) {
  return (
    <svg width="26" height="16" viewBox="0 0 26 16" className={`gicon ${kind}`} aria-hidden="true">
      <rect x="1" y="1" width="24" height="14" rx="3" fill="none" stroke="currentColor" />
      {kind === "sequence" && (
        <>
          <path d="M6 8h14" stroke="currentColor" />
          {[7, 13, 19].map((x) => <circle key={x} cx={x} cy="8" r="1.8" fill="currentColor" />)}
        </>
      )}
      {kind === "choice" && (
        <>
          <path d="M5 8h5l4-3.5h6M10 8l4 3.5h6M10 8h10" stroke="currentColor" fill="none" />
          {[4.5, 8, 11.5].map((y) => <circle key={y} cx="20.5" cy={y} r="1.5" fill="currentColor" />)}
        </>
      )}
      {kind === "all" && (
        <>
          {[7, 13, 19].map((x) => <rect key={x} x={x - 2} y="5" width="4" height="6" fill="currentColor" />)}
        </>
      )}
    </svg>
  );
}

function Flags({ min, max, extra }: { min: number; max: number; extra?: React.ReactNode }) {
  const o = occ(min, max);
  return (
    <>
      <span className={`occ ${max < 0 || max > 1 ? "many" : ""} ${min === 1 && max === 1 ? "once" : ""}`} title="minOccurs..maxOccurs">{o}</span>
      {extra}
    </>
  );
}

/** One node of a content model; `trail` holds the type ids above it (to cut recursion). */
function ModelNode({ node, path, depth, trail }: { node: SMNode; path: string; depth: number; trail: string[] }) {
  const c = useContext(C)!;
  const tr = useT(viewsMessages);
  if (node.kind === "element") return <ElementRow node={node} path={path} depth={depth} trail={trail} />;
  if (node.kind === "any") {
    return (
      <li>
        <div className="row any optional-box">
          <Chevron open={false} show={false} />
          <span className="name">any</span>
          <span className="meta">{node.mode === "other" ? "##other" : node.mode === "list" ? tr("listedNamespaces") : "##any"} · {node.process}</span>
          <Flags min={node.min} max={node.max} />
        </div>
      </li>
    );
  }
  const kids = (node.children ?? []).filter((k): k is SMNode => !!k);
  const open = c.isOpen(path, true, depth);
  return (
    <li>
      <div className={`row group ${node.kind}`} onClick={() => c.toggle(path, open)}>
        <Chevron open={open} show={kids.length > 0} />
        <GroupIcon kind={node.kind} />
        <span className="gname">{node.kind}</span>
        <Flags min={node.min} max={node.max} />
        <span className="meta">{tr(kids.length === 1 ? "itemsOne" : "itemsMany", { n: kids.length })}</span>
      </div>
      {open && kids.length > 0 && (
        <ul className="tree">
          {kids.map((k, i) => <ModelNode key={i} node={k} path={`${path}/${i}`} depth={depth + 1} trail={trail} />)}
        </ul>
      )}
    </li>
  );
}

function ElementRow({ node, path, depth, trail, root }: { node: SMNode; path: string; depth: number; trail: string[]; root?: boolean }) {
  const c = useContext(C)!;
  const tr = useT(viewsMessages);
  const t = node.typeId ? c.model.types?.[node.typeId] : undefined;
  const recursive = !!node.typeId && trail.includes(node.typeId);
  const expandable = !!t && !recursive && hasBody(t);
  const open = expandable && c.isOpen(path, !!root, depth);
  const rowEl = (
    <div className={`row element ${node.min === 0 ? "optional-box" : ""} ${node.abstract ? "abstract" : ""}`} onClick={() => expandable && c.toggle(path, open)}>
      <Chevron open={open} show={expandable} />
      <span className="glyph el">E</span>
      <span className="name" title={node.ns ? `{${node.ns}}${node.name}` : node.name}>{node.name}</span>
      <TypeChip id={node.typeId} name={node.typeName} />
      {recursive && <span className="rec" title={tr("recursiveTitle")}>{tr("recursive")}</span>}
      <Flags
        min={node.min}
        max={node.max}
        extra={
          <>
            {node.nillable && <span className="flag" title="nillable">nil</span>}
            {node.abstract && <span className="flag">abstract</span>}
            {node.fixed !== undefined && node.fixed !== "" && <span className="flag" title={tr("fixedValueTitle")}>= {node.fixed}</span>}
            {node.default !== undefined && node.default !== "" && <span className="flag" title={tr("defaultValueTitle")}>default {node.default}</span>}
            {(node.substitutes ?? []).map((s) => <span key={s} className="flag" title={tr("substitutionHeadTitle")}>≈ {s}</span>)}
          </>
        }
      />
    </div>
  );
  return (
    <li className={root ? "root" : undefined}>
      {rowEl}
      {open && t && <TypeBody type={t} path={path} depth={depth + 1} trail={[...trail, t.id]} />}
    </li>
  );
}

function hasBody(t: SMType): boolean {
  if (t.simple) return !!(t.enumeration?.length || t.baseId || t.itemId || t.members?.length);
  return !!(t.model || t.attributes?.length || t.anyAttribute || t.contentId || t.baseId);
}

function AttrRow({ a }: { a: SMAttr }) {
  return (
    <div className="row attr">
      <Chevron open={false} show={false} />
      <span className="glyph at">@</span>
      <span className="name" title={a.ns ? `{${a.ns}}${a.name}` : a.name}>{a.name}</span>
      <TypeChip id={a.typeId} name={a.typeName} />
      {a.use && a.use !== "optional" && <span className={`flag ${a.use}`}>{a.use}</span>}
      {a.use === "optional" && <span className="meta">optional</span>}
      {a.fixed && <span className="flag">= {a.fixed}</span>}
      {a.default && <span className="flag">default {a.default}</span>}
    </div>
  );
}

function SimpleDetails({ t }: { t: SMType }) {
  const c = useContext(C)!;
  const tr = useT(viewsMessages);
  const enums = t.enumeration ?? [];
  return (
    <div className="details">
      <div className="kv">
        <span className="k">{tr("kindLabel")}</span>
        <span>{t.variety ?? "atomic"}{t.primitive && t.variety !== "union" ? ` · ${t.primitive}` : ""}</span>
      </div>
      {t.baseId && (
        <div className="kv">
          <span className="k">{tr("restricts")}</span>
          <TypeChip id={t.baseId} name={t.baseName} />
        </div>
      )}
      {t.itemId && (
        <div className="kv">
          <span className="k">{tr("listOf")}</span>
          <TypeChip id={t.itemId} name={t.itemName} />
        </div>
      )}
      {(t.members ?? []).length > 0 && (
        <div className="kv">
          <span className="k">{tr("unionOf")}</span>
          <span className="chips">{t.members!.map((m) => <TypeChip key={m} id={m} name={c.model.types?.[m]?.name} />)}</span>
        </div>
      )}
      {enums.length > 0 && (
        <div className="kv">
          <span className="k">{tr("values")}</span>
          <span className="chips">
            {enums.slice(0, 60).map((v, i) => <span key={i} className="enum">{v}</span>)}
            {enums.length > 60 && <span className="meta">{tr("more", { n: enums.length - 60 })}</span>}
          </span>
        </div>
      )}
    </div>
  );
}

/** Attributes, then the content model (or the simple-content type) of a type. */
function TypeBody({ type: t, path, depth, trail }: { type: SMType; path: string; depth: number; trail: string[] }) {
  const c = useContext(C)!;
  const tr = useT(viewsMessages);
  if (t.simple) return <div className="body"><SimpleDetails t={t} /></div>;
  const attrs = t.attributes ?? [];
  const content = t.contentId ? c.model.types?.[t.contentId] : undefined;
  return (
    <ul className="tree body">
      {t.baseId && t.derivation && (
        <li>
          <div className="row note">
            <Chevron open={false} show={false} />
            <span className="meta">{tr("derivationOf", { derivation: t.derivation })}</span> <TypeChip id={t.baseId} name={t.baseName} />
            {t.mixed && <span className="flag">mixed</span>}
            {t.abstract && <span className="flag">abstract</span>}
          </div>
        </li>
      )}
      {content && (
        <li>
          <div className="row note">
            <Chevron open={false} show={false} />
            <span className="meta">{tr("textContent")}</span> <TypeChip id={t.contentId} name={content.name} />
          </div>
          {content.simple && <SimpleDetails t={content} />}
        </li>
      )}
      {attrs.map((a) => <li key={a.name}><AttrRow a={a} /></li>)}
      {t.anyAttribute && (
        <li>
          <div className="row any optional-box"><Chevron open={false} show={false} /><span className="name">anyAttribute</span><span className="meta">{t.anyAttribute}</span></div>
        </li>
      )}
      {t.model && <ModelNode node={t.model} path={`${path}/m`} depth={depth} trail={trail} />}
      {!t.model && !content && attrs.length === 0 && !t.anyAttribute && (
        <li><div className="row note"><Chevron open={false} show={false} /><span className="meta">{t.content === "empty" ? tr("emptyContent") : tr("noContent")}</span></div></li>
      )}
    </ul>
  );
}

function TypeRoot({ type: t }: { type: SMType }) {
  const tr = useT(viewsMessages);
  return (
    <ul className="tree">
      <li className="root">
        <div className="row element">
          <Chevron open show={false} />
          <span className={`glyph ${t.simple ? "st" : "ct"}`}>{t.simple ? "S" : "C"}</span>
          <span className="name" title={t.ns ? `{${t.ns}}${t.name}` : t.name}>{t.name || tr("anonymous")}</span>
          {t.ns === XS && <span className="meta">{tr("builtIn")}</span>}
          {t.abstract && <span className="flag">abstract</span>}
          {t.mixed && <span className="flag">mixed</span>}
        </div>
        <TypeBody type={t} path={`t:${t.id}`} depth={0} trail={[t.id]} />
      </li>
    </ul>
  );
}
