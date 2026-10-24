# Paper submission pipeline + sequential-fallback plan

Tracks the two papers, their edit/result dates, and where to send them next **without
double-submitting**. Companion to [conference-watchlist.md](conference-watchlist.md)
(which tracks the non-archival talks). Dates verified 2026-10-24; AoE = UTC-12.

**Papers**
- **Paper A** — *Authorization Contracts in the Wild* (JWT-config measurement, 530 services).
- **Paper B** — *SoK: Authorization for Autonomous Agents* (systematization + 9-verifier eval).

## The no-double-submit rule (symmetric)

The only hard constraint is that **no two review windows for the same paper overlap**.
That gives two ways to chain venues V1 and V2, in *either* temporal order:

- **Forward (wait-and-see):** deadline(V2) ≥ result(V1). Submit to V2 only after V1's
  result; V1 rejects → send the improved paper to V2, V1 accepts → withdraw/skip V2.
- **Reverse (fast-first):** result(V2) ≤ deadline(V1). Put the **fast-turnaround** venue
  *first*; once its result lands you still make V1's deadline. Fast workshops are the
  usual V2 here, and chaining a fast reject into a later deadline buys extra shots.

So a candidate is judged by comparing *its* (deadline, result) pair to the other's — not
by deadline alone. The table below lists both dates for every venue.

**For the papers already at SACMAT** the reverse direction is unavailable: their review
is already running (Oct → **Dec 21, 2026**), so nothing can finish "before it started."
Any next hop for *these two papers* must therefore open on/after **Dec 21**. The reverse
pattern becomes available the moment SACMAT releases on Dec 21 (chain a fast venue first)
and for any paper not yet committed.

## Currently under review (do not re-submit elsewhere before Dec 21)

| Paper | Venue | ID | Edit window closes | **Result** |
|---|---|---|---|---|
| A | SACMAT 2027 (main research) | #13 | **Oct 31, 2026** | **Dec 21, 2026** |
| B | SACMAT 2027 (SoK track) | #35 | **Oct 31, 2026** | **Dec 21, 2026** |

Both are editable only until **Oct 31**; after that they are locked until the Dec 21
result. So any improvement we want *in the SACMAT versions* must land by Oct 31.

## Venues whose deadline is BEFORE Dec 21 — EXCLUDED as fallbacks

Submitting a SACMAT paper to any of these now = concurrent double submission.

| Venue | Deadline | Notify | Fit | Why excluded |
|---|---|---|---|---|
| PAM 2027 | **Nov 6, 2026** | Dec 14 | Paper A (measurement) | deadline < Dec 21 |
| S&P 2027 Cycle 2 | Nov 17, 2026 | Mar 5 | A or B | deadline < Dec 21 |
| IMC 2027 (R1) | Nov 17, 2026 | — | Paper A | deadline < Dec 21 |
| CODASPY 2027 (incl. SoK track) | Nov 23, 2026 | Jan 25 | A or B | deadline < Dec 21 |

## Valid sequential fallbacks — deadline ON/AFTER Dec 21

Prepare improved papers now; fire only if SACMAT rejects on Dec 21.

| Venue | Deadline | Notify | Best paper | Notes |
|---|---|---|---|---|
| **USENIX Security 2027 Cycle 2** | **Jan 26, 2027** | May 6 | A or B | ~5 wks after SACMAT result; 13pp + Open Science appendix. Takes measurement and SoK. |
| **IMC 2027 (R2)** | **~Apr 7, 2027** (confirm) | — | **Paper A** | the natural measurement home; most time. |
| **ACM CCS 2027** | ~Apr 2027 (confirm) | — | A or B | large venue; confirm exact date. |
| ESORICS / ARES / EuroS&P 2027 | spring 2027 | — | A or B | secondary options. |

### Fast-first chaining (the reverse direction), usable after Dec 21

Once SACMAT releases on Dec 21, a *reject* doesn't have to go straight to a slow venue.
A fast-turnaround workshop whose result lands before a later deadline can go **first**,
buying an extra shot. We already know the fast workshops in this space (they rejected the
earlier drafts and will have 2027 editions with the same quick cadence): **SSR, WAITI,
AIDC** workshops. Pattern: SACMAT reject (Dec 21) → fast workshop (result in weeks, if a
2027 edition is open) → still make **USENIX Sec C2 (Jan 26)** or **IMC R2 (Apr)**. Only
run a workshop that is itself non-archival or whose timeline clears the next deadline, so
the windows never overlap. Confirm each 2027 workshop's dates when its CFP opens.

## Plan

1. **SACMAT #13 / #35 (by Oct 31):** leave #13 as-is — it already addresses the SSR
   Paper A reviews (declared-contract vs. effective-enforcement framing; consumer/
   verifier terminology; RFC 9068/8707/8725). #35 already carries the full reviewer
   rework + the 9-verifier eval. **Do not add the verifier eval to #13:** it is Paper
   B's contribution and duplicating it across two same-venue submissions is a
   self-overlap risk. Nothing else is urgent before Oct 31.
2. **Prepare improved "next-venue" copies now** (keep improving off every review +
   the new eval), so on a Dec 21 reject we can submit within days:
   - Paper A → folds the enforcement-gap evidence in (its config-vs-enforcement claim is
     currently illustrated hypothetically; the verifier eval makes it measured) → target
     **IMC 2027 R2** or **USENIX Sec '27 C2**. At a *different* venue the eval no longer
     overlaps with Paper B.
   - Paper B → already strong post-eval → target **USENIX Sec '27 C2** or CCS.
3. **Dec 21 decision gate:**
   - Accept → done; withdraw from any fallback not yet acted on.
   - Reject → submit the prepared improved paper to the chosen fallback (deadline after
     Dec 21), with the reviews addressed.

## Key dates to watch

| Date | Event |
|---|---|
| **Oct 31, 2026** | SACMAT edit window closes (#13, #35 locked after) |
| Nov 6 / 17 / 23 | PAM / S&P-C2, IMC-R1 / CODASPY deadlines — **skip** (double-submit) |
| **Dec 21, 2026** | **SACMAT #13 + #35 results** — the decision gate |
| **Jan 26, 2027** | USENIX Sec '27 Cycle 2 deadline (fallback) |
| ~Apr 7, 2027 | IMC 2027 R2 deadline (fallback, Paper A) |
