#!/usr/bin/env bash
# Reproduce go-xslt's published W3C conformance numbers.
#
# Clones the three W3C suites into test/conformance/ (at the commits the
# README's "Conformance benchmarks" table was measured against), builds the
# conformance test binary once, runs the four published measurements and
# prints a summary table next to the published numbers.
#
#   scripts/conformance.sh                  # everything, pinned suite commits
#   scripts/conformance.sh --only xslt,qt3  # subset: xslt, xslt-full, qt3, xsd
#   scripts/conformance.sh --latest         # suites' current HEAD instead
#   scripts/conformance.sh --update         # move existing clones to the
#                                           # pinned commit (or HEAD, --latest)
#   scripts/conformance.sh --out DIR        # results dir (conformance-results/)
#
# Needs: git, go, and network access for the first run.
set -uo pipefail

# ---- pinned suite commits (keep in sync with README.md) ---------------------
XSLT_REPO=https://github.com/w3c/xslt30-test; XSLT_PIN=fddf1cf9
QT3_REPO=https://github.com/w3c/qt3tests;     QT3_PIN=201a6e46
XSD_REPO=https://github.com/w3c/xsdtests;     XSD_PIN=7bc3365c

# ---- published numbers (README.md): passed/ran ------------------------------
PUB_XSLT="8614/8626"; PUB_XSLTFULL="12267/12296"; PUB_QT3="22099/22153"
PUB_XSD10="39346/39351"; PUB_XSD11="41499/41503"

ROOT=$(cd "$(dirname "$0")/.." && pwd)
OUT="$ROOT/conformance-results"
ONLY="xslt,xslt-full,qt3,xsd"
LATEST=0; UPDATE=0
# (no associative arrays: macOS ships bash 3.2)

while [ $# -gt 0 ]; do
  case "$1" in
    --latest) LATEST=1 ;;
    --update) UPDATE=1 ;;
    --only)   ONLY="${2:?--only needs a list}"; shift ;;
    --out)    OUT="${2:?--out needs a directory}"; shift ;;
    -h|--help) sed -n 2,16p "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "unknown option: $1" >&2; exit 2 ;;
  esac
  shift
done
case "$OUT" in /*) ;; *) OUT="$PWD/$OUT" ;; esac
want() { case ",$ONLY," in *",$1,"*) return 0 ;; esac; return 1; }
for s in ${ONLY//,/ }; do
  case "$s" in xslt|xslt-full|qt3|xsd) ;; *) echo "unknown suite '$s' (xslt, xslt-full, qt3, xsd)" >&2; exit 2 ;; esac
done

# ---- fetch a suite ----------------------------------------------------------
# fetch_suite DIRNAME REPO PIN
fetch_suite() {
  local dir="$ROOT/test/conformance/$1" repo=$2 pin=$3 full head
  if [ -e "$dir" ]; then
    if [ ! -d "$dir/.git" ]; then
      echo "$1: $dir exists but is not a git clone; using it as is (commit unknown)"; return 0
    fi
    head=$(git -C "$dir" rev-parse HEAD)
    if [ "$LATEST" = 1 ]; then
      if [ "$UPDATE" = 1 ]; then
        git -C "$dir" fetch -q --depth 1 origin HEAD && git -C "$dir" checkout -q FETCH_HEAD
        echo "$1: moved to current HEAD $(git -C "$dir" rev-parse HEAD)"
      else
        echo "$1: using existing clone at ${head:0:8} (--update would move it to HEAD)"
      fi
      return 0
    fi
    full=$(git -C "$dir" rev-parse --verify -q "$pin^{commit}" || true)
    if [ -n "$full" ] && [ "$full" = "$head" ]; then
      echo "$1: already at pinned commit $head"; return 0
    fi
    if [ "$UPDATE" != 1 ]; then
      echo "$1: existing clone is at ${head:0:8}, pinned commit is $pin; keeping it (use --update to check out $pin)"
      return 0
    fi
    git -C "$dir" fetch -q --unshallow 2>/dev/null || git -C "$dir" fetch -q origin
    git -C "$dir" checkout -q "$pin" || { echo "$1: cannot check out $pin" >&2; return 1; }
    echo "$1: checked out pinned commit $(git -C "$dir" rev-parse HEAD)"
    return 0
  fi
  if [ "$LATEST" = 1 ]; then
    git clone -q --depth 1 "$repo" "$dir" || return 1
  else
    # A blobless clone keeps the full commit history (so the short SHA
    # resolves) but downloads file contents only for the commit checked out.
    git clone -q --filter=blob:none --no-checkout "$repo" "$dir" || return 1
    git -C "$dir" checkout -q "$pin" || { echo "$1: cannot check out $pin" >&2; return 1; }
  fi
  echo "$1: cloned, at $(git -C "$dir" rev-parse HEAD)"
}

echo "== fetching suites =="
if [ "$LATEST" = 1 ]; then
  echo "NOTE: --latest uses each suite's current HEAD. Numbers may differ from the published ones by a case or two."
fi
if want xslt || want xslt-full; then fetch_suite xslt30-test "$XSLT_REPO" "$XSLT_PIN" || exit 1; fi
if want qt3; then fetch_suite qt3tests "$QT3_REPO" "$QT3_PIN" || exit 1; fi
if want xsd; then fetch_suite xsdtests "$XSD_REPO" "$XSD_PIN" || exit 1; fi

# ---- build the test binary once ---------------------------------------------
mkdir -p "$OUT"
BIN="$OUT/conformance.test"
echo "== building test binary =="
(cd "$ROOT" && go test -c -o "$BIN" ./test/conformance/) || { echo "build failed" >&2; exit 1; }

# ---- run --------------------------------------------------------------------
# The binary must run from test/conformance/ (the harness resolves ../..).
# The harnesses have no per-case timeouts; -test.timeout is only a backstop
# for a genuine hang.
FAILED=0
run() { # NAME TESTNAME LOGBASE [ENV=VAL...]
  local name=$1 test=$2 base=$3; shift 3
  local log="$OUT/$base.log" t0=$SECONDS rc
  echo "== running $name =="
  (cd "$ROOT/test/conformance" && env "$@" "$BIN" -test.run "^$test\$" -test.v -test.timeout 60m) >"$log" 2>&1
  rc=$?
  eval "T_${name//-/_}=$((SECONDS - t0)); RC_${name//-/_}=$rc"
  # unanchored on purpose: t.Log lines carry a "file:line:" prefix
  grep -o 'FAIL .*' "$log" | sort -u >"$OUT/$base.fails.txt"
  if [ $rc -ne 0 ]; then FAILED=1; fi
  return 0
}

if want xslt;      then run xslt      TestXSLT30 xslt30      XSLT30_FAILS=1; fi
if want xslt-full; then run xslt-full TestXSLT30 xslt30-full XSLT30_FAILS=1 XSLT30_FULL=1; fi
if want qt3;       then run qt3       TestQT3    qt3         QT3_FAILDUMP=1; fi
if want xsd;       then run xsd       TestXSD    xsd; fi

# ---- summary ----------------------------------------------------------------
# "XSLT30 results: ... ran N (pass P, ..." / "QT3 results: ..." -> P/N
suite_result() { sed -n 's/.*results:.*ran \([0-9]*\) (pass \([0-9]*\),.*/\2\/\1/p' "$1" 2>/dev/null | tail -1; }
# "XSD 1.0: 99.99% pass (P/N ran) ..." -> P/N
xsd_result()   { sed -n "s/.*XSD $2: .*(\([0-9]*\)\/\([0-9]*\) ran).*/\1\/\2/p" "$1" 2>/dev/null | tail -1; }
row() { # LABEL GOT PUBLISHED TIME
  local mark="MISMATCH"
  if [ -z "$2" ]; then mark="NO RESULT"; set -- "$1" "-" "$3" "$4"
  elif [ "$2" = "$3" ]; then mark="match"; fi
  printf '%-27s %-14s %-14s %-10s %s\n' "$1" "$2" "$3" "$mark" "$4"
}
tm() { local v="T_${1//-/_}"; local s=${!v:-0}; printf '%dm%02ds' $((s / 60)) $((s % 60)); }

echo
echo "== summary (passed/ran) =="
printf '%-27s %-14s %-14s %-10s %s\n' "Suite" "This run" "Published" "" "Time"
if want xslt;      then row "XSLT 3.0 default"      "$(suite_result "$OUT/xslt30.log")"      "$PUB_XSLT"     "$(tm xslt)"; fi
if want xslt-full; then row "XSLT 3.0 combined"     "$(suite_result "$OUT/xslt30-full.log")" "$PUB_XSLTFULL" "$(tm xslt-full)"; fi
if want qt3;       then row "QT3 (XPath 3.1, F&O)"  "$(suite_result "$OUT/qt3.log")"         "$PUB_QT3"      "$(tm qt3)"; fi
if want xsd; then
  row "XSD 1.0" "$(xsd_result "$OUT/xsd.log" 1.0)" "$PUB_XSD10" "$(tm xsd)"
  row "XSD 1.1" "$(xsd_result "$OUT/xsd.log" 1.1)" "$PUB_XSD11" "(same run)"
fi
echo
echo "Suite commits:"
for d in xslt30-test qt3tests xsdtests; do
  if [ -d "$ROOT/test/conformance/$d/.git" ]; then
    echo "  $d  $(git -C "$ROOT/test/conformance/$d" rev-parse HEAD)"
  fi
done
if [ "$LATEST" = 1 ]; then echo "(--latest: numbers may differ from the published ones by a case or two.)"; fi
echo "Logs and fail-name lists: $OUT"

for n in xslt xslt-full qt3 xsd; do
  v="RC_${n//-/_}"; rc=${!v:-0}
  if want $n && [ "$rc" -ne 0 ]; then
    echo "WARNING: the $n test run exited with status $rc (see its log)."
    if grep -q 'undocumented failures' "$OUT"/*.log 2>/dev/null; then
      echo "         A failing case that is not named in docs/accepted-failures/ fails the harness."
    fi
  fi
done
exit $FAILED
