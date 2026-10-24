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

### Withdrawal is NOT a license for concurrent archival submission

Dual-submission policies forbid the concurrent *submission/review*, not just concurrent
acceptance; "I'll withdraw the loser" does not make it compliant, because the paper has
already been under review at two archival venues at once (the prohibited act). SACMAT/ACM
and USENIX state this explicitly; penalty is desk-reject of both plus possible multi-year
bans, and it is caught via overlapping PC/reviewers and dual-submission detection. So the
non-overlap rule above stands for **archival** venues.

Where withdrawal *is* valid:
- **Non-archival venues (all our talks; workshops that permit work-under-review):**
  concurrent is fine, withdraw as a courtesy. This is why the KubeCon co-located talks and
  AIDC lightning talks do not conflict with the papers.
- **Withdraw-then-submit:** pull from V1 before its decision, then submit to V2. No overlap.
- **Sequential safety-valve:** submit to V2 only when its deadline is after V1's result, so
  you submit only if V1 rejected; if V1 accepts first, withdraw/skip V2 before it reviews.

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
| **CCS 2027 Cycle 1** | ~Jan 14–15, 2027 *(tentative — official CFP not posted)* | ~Apr 2027 | A or B | soonest shot but ~3 wks after SACMAT result and top-tier/hard. Confirm when CFP posts. |
| **USENIX Security 2027 Cycle 2** | **Jan 26, 2027** *(official)* | **May 6, 2027** | A or B | ~5 wks after SACMAT; most reliable near-term. 13pp + mandatory Open Science appendix. |
| **IMC 2027 (R2)** | ~Apr 7, 2027 *(tentative)* | — | **Paper A** | the natural measurement home; most prep time. Confirm. |
| **CCS 2027 Cycle 2** | ~Apr 29, 2027 *(tentative)* | ~Jul 2027 | A or B | later CCS shot. |
| ESORICS / ARES / EuroS&P 2027 | spring 2027 | — | A or B | secondary options. |
| **SSR 2027** (Trento, Dec 9–10, 2027) | ~Sep 2027 *(est; SSR'26 was Sep 15)* | ~Oct 2027 | A or B | the natural *standardization* home — but SSR'26 rejected both, so it must lead with the fixes. A year out. |

### Fast-first chaining — NOT available in the Dec-2026/Jan-2027 window

The reverse (fast-workshop-first) pattern is sound in principle, but the workshops we
know — **SSR, WAITI, AIDC** — all recur only in **late 2027**, so none can be a fast hop
right after SACMAT's Dec 21, 2026 result:
- **SSR 2027** — Trento, Dec 9–10, 2027; deadline ~Sep 2027.
- **WAITI 2027 / AIDC 2027** — co-located with **ACSAC 2027 (Dec 6, 2027, California)**;
  their CFPs open mid-2027, deadlines ~Sep–Oct 2027.

So after a Dec 21 reject there is no quick workshop to chain first; the near-term shots
are the two January conference deadlines below. The fast-first pattern only becomes
useful again in ~Sep–Dec 2027 (if we're still iterating then). Dates are estimates
except the event dates and USENIX C2; confirm each CFP when posted.

### The realistic post-Dec-21 chain

1. **Dec 21, 2026** — SACMAT #13/#35 results.
2. If rejected → the two near-term deadlines: **CCS '27 C1 (~Jan 14, tentative, hard)**
   and/or **USENIX Sec '27 C2 (Jan 26, official)**. USENIX is the safer, confirmed target.
3. Next tier → **IMC '27 R2 (~Apr, Paper A)**, **CCS '27 C2 (~Apr)**.
4. Late-2027 homes → **SSR '27 (Sep, the standardization fit)**, **ACSAC/AIDC/WAITI '27 (Dec)**.

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
| Nov 6 / 17 / 17 / 23 | PAM / S&P-C2 / IMC-R1 / CODASPY deadlines — **skip** (would be concurrent) |
| **Dec 21, 2026** | **SACMAT #13 + #35 results** — the decision gate |
| ~Jan 14–15, 2027 | CCS '27 Cycle 1 deadline *(tentative)* — fallback |
| **Jan 26, 2027** | **USENIX Sec '27 Cycle 2** deadline *(official)* — primary fallback |
| ~Apr 7, 2027 | IMC '27 R2 deadline *(tentative)* — fallback, Paper A |
| ~Apr 29, 2027 | CCS '27 Cycle 2 deadline *(tentative)* |
| ~Sep 2027 | SSR '27 + ACSAC/AIDC/WAITI '27 CFPs — late homes / fast-first reopens |
| Dec 2027 | SSR '27 (Trento) & ACSAC '27 (California) events |

*Dates are confirmed for events and USENIX C2; CCS/IMC/SSR/ACSAC deadlines are tracker
estimates until their official CFPs post — re-verify before relying on them.*
