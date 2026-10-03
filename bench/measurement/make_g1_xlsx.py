#!/usr/bin/env python3
"""Build the human-ready G1 labelling workbook from the AI-drafted packet.

Turns `labeling-packet-ai.csv` (independent-model first-pass labels) into an
ergonomic .xlsx a co-author can adjudicate in about an hour: dropdown-validated
label cells, the AI draft and its rationale parked on the right as a reference,
and a BLIND subset floated to the top so the honest (un-anchored) Cohen's kappa
can be computed. Fill it, save the Label sheet back to CSV, and grade with
grade_dual.py. See LABELING.md.

  python3 bench/measurement/make_g1_xlsx.py \
      bench/measurement/out-val/labeling-packet-ai.csv \
      bench/measurement/out-val/g1-packet.xlsx [--blind N]
"""
import argparse
import csv
import hashlib

from openpyxl import Workbook
from openpyxl.styles import Alignment, Font, PatternFill
from openpyxl.utils import get_column_letter
from openpyxl.worksheet.datavalidation import DataValidation

LABELS = ["correct", "correct-extra", "discovery-miss", "parser-artifact",
          "no-consumer", "spurious", "wrong-attribution"]
BLIND_SEED = "keyway-g1-blind"

# Label sheet columns, in review order: identity + evidence first, the two human
# columns next, then the AI reference pushed to the far right so it can be ignored
# during the blind pass.
COLS = [
    ("row_id", 7), ("blind_first", 12), ("repo", 30), ("field", 10),
    ("value", 30), ("side", 15), ("in_nonauth_context", 12),
    ("has_jwt_consumer", 12), ("human_label_blind", 18), ("human_label", 18),
    ("notes", 24), ("ai_label", 16), ("ai_confidence", 8), ("ai_rationale", 80),
    ("suggested_label", 16),
]
HUMAN_COLS = {"human_label_blind", "human_label"}


def blind_set(rows, n):
    """Deterministic pseudo-random subset of row indices for the blind pass."""
    ranked = sorted(
        range(len(rows)),
        key=lambda i: hashlib.sha256(
            f"{BLIND_SEED}|{rows[i]['repo']}|{rows[i]['field']}|{rows[i]['value']}"
            .encode()).hexdigest())
    return set(ranked[:min(n, len(rows))])


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("inp", nargs="?", default="bench/measurement/out-val/labeling-packet-ai.csv")
    ap.add_argument("out", nargs="?", default="bench/measurement/out-val/g1-packet.xlsx")
    ap.add_argument("--blind", type=int, default=25, help="size of the blind-first subset")
    args = ap.parse_args()

    rows = list(csv.DictReader(open(args.inp)))
    blind = blind_set(rows, args.blind)
    # Blind rows first (so the annotator meets them before any AI label), original
    # order preserved within each group for traceability.
    order = [i for i in range(len(rows)) if i in blind] + \
            [i for i in range(len(rows)) if i not in blind]

    wb = Workbook()
    _instructions(wb.active, len(rows), len(blind))
    _label_sheet(wb.create_sheet("Label"), rows, order, blind)
    wb.save(args.out)
    print(f"wrote {args.out}: {len(rows)} rows ({len(blind)} blind-first) + instructions")
    print("Fill human_label_blind on the BLIND rows FIRST (ignore the AI columns),")
    print("then human_label on every row using the AI draft as reference. Then:")
    print("  save the Label sheet as CSV and run grade_dual.py on it.")


def _instructions(ws, n, nblind):
    ws.title = "START HERE"
    ws.sheet_view.showGridLines = False
    big = Font(bold=True, size=14)
    hd = Font(bold=True, size=11)
    wrap = Alignment(wrap_text=True, vertical="top")
    ws.column_dimensions["A"].width = 22
    ws.column_dimensions["B"].width = 100

    def row(r, a, b="", font=None, w=False):
        ws[f"A{r}"] = a
        ws[f"B{r}"] = b
        if font:
            ws[f"A{r}"].font = font
        if w:
            ws[f"B{r}"].alignment = wrap

    row(1, "Keyway G1", "Human validation packet (gold-standard discovery precision / recall)", big)
    row(3, "What this is",
        f"{n} contested (repo, field, value) rows where Keyway's discovery and an "
        f"independent parse disagreed, plus a calibration sample. You adjudicate each "
        f"with one label from the taxonomy below. This is the gold standard the paper "
        f"cites; the AI draft is only a reference.", hd, w=True)
    row(6, "Do this, in order", "", hd)
    row(7, "  1. BLIND pass",
        f"The first {nblind} rows are marked BLIND. Fill ONLY their 'human_label_blind' "
        f"cell, using repo/field/value/side/flags and your own judgement. Do NOT scroll "
        f"right to the ai_label / ai_rationale columns yet. This gives an un-anchored "
        f"agreement number.", w=True)
    row(10, "  2. Full pass",
        "Now fill 'human_label' on EVERY row (including the blind ones). Here you may "
        "use the ai_label and ai_rationale columns (far right) as a reference and "
        "correct them. Both label cells are dropdowns.", w=True)
    row(13, "  3. Grade",
        "File > Save As > CSV (the 'Label' sheet) to e.g. labeling-packet-human.csv, then:", w=True)
    row(14, "", "python3 bench/measurement/grade_dual.py bench/measurement/out-val/labeling-packet-human.csv")
    row(15, "", "It reports v1 (AI) vs v2 (human) precision/recall, Cohen's kappa, and the "
        "blind-subset kappa (the honest, un-anchored one).", w=True)

    row(17, "Label taxonomy", "fill with exactly one", hd)
    tax = [
        ("correct", "real auth config, correctly captured/attributed -> TP"),
        ("correct-extra", "captured-only, but the value IS real (parser missed it) -> TP"),
        ("discovery-miss", "real auth value Keyway FAILED to capture -> hurts recall (FN)"),
        ("parser-artifact", "parser scraped a non-auth value (ConfigMap/annotation/comment) -> excluded"),
        ("no-consumer", "real value but its RequestAuthentication isn't in the corpus -> excluded"),
        ("spurious", "Keyway captured a value that is NOT real auth config -> hurts precision (FP)"),
        ("wrong-attribution", "real value attached to the WRONG consumer -> hurts precision (FP)"),
    ]
    for i, (lab, desc) in enumerate(tax):
        r = 18 + i
        ws[f"A{r}"] = lab
        ws[f"A{r}"].font = Font(bold=True)
        ws[f"B{r}"] = desc
    r = 18 + len(tax)
    row(r + 1, "Metrics", "recall = TP / (TP + discovery-miss);  "
        "precision = TP / (TP + spurious + wrong-attribution). "
        "parser-artifact and no-consumer are excluded from both.", hd, w=True)
    row(r + 3, "Two annotators",
        "For inter-rater agreement, a second co-author fills their own copy of this "
        "file; compare the two 'human_label' columns. Strongest on the blind subset.", w=True)


def _label_sheet(ws, rows, order, blind):
    ws.freeze_panes = "I2"  # keep header + identity/evidence columns while scrolling
    head_font = Font(bold=True, color="FFFFFF")
    head_fill = PatternFill("solid", fgColor="334155")
    blind_fill = PatternFill("solid", fgColor="FEF3C7")
    wrap = Alignment(wrap_text=True, vertical="top")

    for c, (name, width) in enumerate(COLS, start=1):
        cell = ws.cell(row=1, column=c, value=name)
        cell.font = head_font
        cell.fill = head_fill
        ws.column_dimensions[get_column_letter(c)].width = width

    for out_r, i in enumerate(order, start=2):
        src = rows[i]
        is_blind = i in blind
        rec = {
            "row_id": out_r - 1,
            "blind_first": "BLIND" if is_blind else "",
            "repo": src.get("repo", ""),
            "field": src.get("field", ""),
            "value": src.get("value", ""),
            "side": src.get("side", ""),
            "in_nonauth_context": src.get("in_nonauth_context", ""),
            "has_jwt_consumer": src.get("has_jwt_consumer", ""),
            "human_label_blind": "",
            "human_label": "",
            "notes": "",
            "ai_label": src.get("ai_label", ""),
            "ai_confidence": src.get("ai_confidence", ""),
            "ai_rationale": src.get("ai_rationale", ""),
            "suggested_label": src.get("suggested_label", ""),
        }
        for c, (name, _) in enumerate(COLS, start=1):
            cell = ws.cell(row=out_r, column=c, value=rec[name])
            if name in ("value", "ai_rationale"):
                cell.alignment = wrap
            if name == "blind_first" and is_blind:
                cell.fill = blind_fill
                cell.font = Font(bold=True, color="92400E")

    # Dropdown-validate the two human columns over the data rows.
    dv = DataValidation(type="list", formula1='"' + ",".join(LABELS) + '"', allow_blank=True)
    ws.add_data_validation(dv)
    last = len(order) + 1
    for name in HUMAN_COLS:
        col = get_column_letter(1 + [c[0] for c in COLS].index(name))
        dv.add(f"{col}2:{col}{last}")


if __name__ == "__main__":
    main()
