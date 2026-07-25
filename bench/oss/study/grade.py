#!/usr/bin/env python3
"""Grade Keyway discovery against an INDEPENDENT parse of the same real manifests.

Truth = what a plain YAML walk finds declared (issuers, audiences, claim names).
Captured = what `keyway discover --output json` extracted.
Reports per-field recall (did Keyway capture every declared value?) + coverage.
"""
import glob, json, os, re, sys
import yaml

MANIFEST_DIR = sys.argv[1]
KEYWAY_JSON = sys.argv[2]

CLAIM_RE = re.compile(r'request\.auth\.claims\[([^\]]+)\]')

def walk(node, issuers, auds, claims):
    if isinstance(node, dict):
        # An Istio jwtRule or an Envoy provider: a dict carrying an issuer.
        if 'issuer' in node and isinstance(node.get('issuer'), str):
            issuers.add(node['issuer'].strip())
            a = node.get('audiences')
            if isinstance(a, list):
                for x in a:
                    if isinstance(x, str):
                        auds.add(x.strip())
        for v in node.values():
            walk(v, issuers, auds, claims)
    elif isinstance(node, list):
        for v in node:
            walk(v, issuers, auds, claims)
    elif isinstance(node, str):
        for m in CLAIM_RE.findall(node):
            # nested-claim edge case: claims[a][b] -> take first segment 'a'
            claims.add(m.split('][')[0].strip())

def truth_of(path):
    iss, aud, cl = set(), set(), set()
    with open(path) as f:
        try:
            for doc in yaml.safe_load_all(f):
                walk(doc, iss, aud, cl)
        except Exception:
            return None  # unparseable
    return iss, aud, cl

# ---- Keyway's captured values (global union) ----
consumers = json.load(open(KEYWAY_JSON))
kw_iss, kw_aud, kw_cl = set(), set(), set()
for c in consumers:
    e = c.get('expects') or {}
    for x in (e.get('issuers') or []):
        kw_iss.add(x.strip())
    for x in (e.get('audiences') or []):
        kw_aud.add(x.strip())
    for x in (e.get('required_claims') or []):
        kw_cl.add(x.strip())

# ---- Truth over all files ----
files = sorted(glob.glob(os.path.join(MANIFEST_DIR, '*.yaml')))
T_iss, T_aud, T_cl = set(), set(), set()
files_with_issuer = 0
unparsed = 0
per_file = []
for p in files:
    t = truth_of(p)
    if t is None:
        unparsed += 1
        per_file.append((os.path.basename(p), 'UNPARSED'))
        continue
    iss, aud, cl = t
    if iss:
        files_with_issuer += 1
    T_iss |= iss; T_aud |= aud; T_cl |= cl
    # per-file issuer capture
    missed = iss - kw_iss
    per_file.append((os.path.basename(p), 'ok' if not missed else 'MISSED:' + ';'.join(sorted(missed))))

def rec(captured, truth):
    return (len(truth & captured) / len(truth)) if truth else 1.0

print(f"Files analysed: {len(files)}  (declared an issuer: {files_with_issuer}, unparsed: {unparsed})")
print(f"Keyway discovered consumers: {len(consumers)}")
print()
print(f"{'Field':12} {'declared':>9} {'captured':>9} {'recall':>8}")
for name, T, K in (("issuers", T_iss, kw_iss), ("audiences", T_aud, kw_aud), ("claims", T_cl, kw_cl)):
    hit = len(T & K)
    print(f"{name:12} {len(T):>9} {hit:>9} {rec(K,T):>7.1%}")
print()
missed_iss = sorted(T_iss - kw_iss)
if missed_iss:
    print(f"Issuers declared but NOT captured ({len(missed_iss)}):")
    for m in missed_iss:
        print("   -", m)
else:
    print("Every declared issuer was captured.")

# emit machine-readable summary
summary = {
    "files": len(files), "files_with_issuer": files_with_issuer, "unparsed": unparsed,
    "consumers": len(consumers),
    "issuers": {"declared": len(T_iss), "captured": len(T_iss & kw_iss), "recall": rec(kw_iss, T_iss)},
    "audiences": {"declared": len(T_aud), "captured": len(T_aud & kw_aud), "recall": rec(kw_aud, T_aud)},
    "claims": {"declared": len(T_cl), "captured": len(T_cl & kw_cl), "recall": rec(kw_cl, T_cl)},
    "missed_issuers": missed_iss,
}
json.dump(summary, open(os.path.join(os.path.dirname(KEYWAY_JSON), "grade.json"), "w"), indent=2)
