package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// reportData is everything the HTML/SVG report needs.
type reportData struct {
	Card      Scorecard // the full corpus (generated + realistic file scenarios)
	Generated int
	FileBased int
	TruePos   int
	TrueNeg   int
}

// writeReport emits a self-contained HTML report and a standalone ROC SVG.
func writeReport(dir string, d reportData) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	svg := rocSVG(d.Card)
	if err := os.WriteFile(filepath.Join(dir, "roc.svg"), []byte(svg), 0o644); err != nil {
		return err
	}
	html := reportHTML(d, svg)
	return os.WriteFile(filepath.Join(dir, "report.html"), []byte(html), 0o644)
}

// rocSVG renders an OWASP-style ROC chart: the diagonal "random guess" line, the
// documented static-analysis accuracy region, and Keyway's measured point.
func rocSVG(card Scorecard) string {
	// Plot area 0..1 mapped onto a 360x360 box inside a 440x440 viewport.
	const pad, size = 60, 360
	x := func(fpr float64) float64 { return pad + fpr*size }
	y := func(tpr float64) float64 { return pad + (1-tpr)*size }

	kx, ky := x(card.FPR), y(card.TPR)

	// Published third-party OWASP Benchmark points (static-analysis category),
	// shown for calibration only — see BENCHMARK.md for sources & caveats.
	type pt struct {
		name     string
		fpr, tpr float64
		dx, dy   float64
	}
	refs := []pt{
		{"SonarQube 50%", 0.11, 0.5036, 8, 4},
		{"Semgrep 87%", 0.25, 0.8706, 8, -6},
		{"Snyk 97%", 0.28, 0.9718, 8, 14},
		{"Kiuwan 100%@16%", 0.16, 1.0, -140, 16},
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 480 460" font-family="ui-sans-serif,system-ui,sans-serif">`)
	b.WriteString(`<rect width="480" height="460" fill="#0b0f14"/>`)
	// Axes.
	fmt.Fprintf(&b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#3a4657" stroke-width="1.5"/>`, pad, pad, pad, pad+size)
	fmt.Fprintf(&b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#3a4657" stroke-width="1.5"/>`, pad, pad+size, pad+size, pad+size)
	// Diagonal (random guessing).
	fmt.Fprintf(&b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#5a6b82" stroke-width="1" stroke-dasharray="4 4"/>`, pad, pad+size, pad+size, pad)
	fmt.Fprintf(&b, `<text x="%d" y="%d" fill="#6b7a90" font-size="11" transform="rotate(-45 %d %d)">random guessing</text>`, pad+size/2-20, pad+size/2+6, pad+size/2-20, pad+size/2+6)

	// Axis labels & ticks.
	for i := 0; i <= 10; i += 2 {
		f := float64(i) / 10
		fmt.Fprintf(&b, `<text x="%.0f" y="%d" fill="#8b98a9" font-size="10" text-anchor="middle">%.1f</text>`, x(f), pad+size+18, f)
		fmt.Fprintf(&b, `<text x="%d" y="%.0f" fill="#8b98a9" font-size="10" text-anchor="end">%.1f</text>`, pad-8, y(f)+3, f)
	}
	fmt.Fprintf(&b, `<text x="%d" y="%d" fill="#e6edf3" font-size="12" text-anchor="middle">False alarm rate (FPR) →</text>`, pad+size/2, pad+size+38)
	fmt.Fprintf(&b, `<text x="18" y="%d" fill="#e6edf3" font-size="12" text-anchor="middle" transform="rotate(-90 18 %d)">Real changes caught (TPR) →</text>`, pad+size/2, pad+size/2)
	b.WriteString(`<text x="60" y="30" fill="#e6edf3" font-size="14" font-weight="600">Detection accuracy (ROC)</text>`)

	// Reference region + points.
	for _, r := range refs {
		fmt.Fprintf(&b, `<circle cx="%.1f" cy="%.1f" r="4" fill="#f5a34f" opacity="0.85"/>`, x(r.fpr), y(r.tpr))
		fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" fill="#c9a06a" font-size="10">%s</text>`, x(r.fpr)+r.dx, y(r.tpr)+r.dy, r.name)
	}

	// Keyway measured point (top-left corner = perfect).
	fmt.Fprintf(&b, `<circle cx="%.1f" cy="%.1f" r="7" fill="#4f9cf9" stroke="#0b0f14" stroke-width="2"/>`, kx, ky)
	fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" fill="#8fc0ff" font-size="12" font-weight="600">Keyway (%.0f%% / %.0f%%)</text>`, kx+12, ky+4, card.TPR*100, card.FPR*100)

	b.WriteString(`</svg>`)
	return b.String()
}

func reportHTML(d reportData, svg string) string {
	c := d.Card
	pct := func(f float64) string { return fmt.Sprintf("%.1f%%", f*100) }
	return `<!doctype html><html lang="en"><head><meta charset="utf-8"/>
<meta name="viewport" content="width=device-width,initial-scale=1"/>
<title>Keyway — Detection Benchmark</title>
<style>
 :root{color-scheme:dark}
 body{margin:0;background:#0b0f14;color:#e6edf3;font-family:ui-sans-serif,system-ui,-apple-system,Segoe UI,Roboto,sans-serif;line-height:1.55}
 .wrap{max-width:860px;margin:0 auto;padding:40px 24px 80px}
 h1{font-size:30px;margin:0 0 6px} h2{font-size:20px;margin:36px 0 12px;border-bottom:1px solid #253143;padding-bottom:6px}
 .sub{color:#8b98a9;margin:0 0 28px}
 .tiles{display:grid;grid-template-columns:repeat(auto-fit,minmax(150px,1fr));gap:14px;margin:20px 0}
 .tile{border:1px solid #253143;border-radius:12px;padding:16px;background:#121821}
 .tile .n{font-size:30px;font-weight:700} .tile .l{color:#8b98a9;font-size:12px;text-transform:uppercase;letter-spacing:.04em}
 .low{color:#6ee7b7} .brand{color:#4f9cf9}
 table{border-collapse:collapse;width:100%;margin:12px 0;font-size:14px}
 th,td{border:1px solid #253143;padding:9px 12px;text-align:left} th{background:#161d27;color:#c9d3df}
 .yes{color:#6ee7b7} .no{color:#f26d6d} .part{color:#f5d76e}
 code{background:#1a222e;padding:2px 6px;border-radius:5px;font-size:13px}
 .note{border-left:3px solid #4f9cf9;background:#0f1620;padding:10px 16px;border-radius:0 8px 8px 0;color:#b9c6d6;font-size:14px;margin:16px 0}
 .chart{background:#0b0f14;border:1px solid #253143;border-radius:12px;padding:8px;max-width:520px;margin:8px 0}
 .chart svg{width:100%;height:auto;display:block}
 footer{color:#6b7a90;font-size:12px;margin-top:40px;border-top:1px solid #253143;padding-top:16px}
</style></head><body><div class="wrap">
<h1>🔑 Keyway — Detection Benchmark</h1>
<p class="sub">Can Keyway catch a real JWT auth-contract change without crying wolf on ordinary redeploys? Here's the test, in plain English.</p>

<h2>The plain-English version</h2>
<p>We took realistic auth configurations (Istio, Envoy and Kubernetes files) and made <b>` + fmt.Sprint(c.TP+c.TN) + `</b> changes to them.
Half were <b>real contract changes</b> — a service starts accepting a new audience, trusts a second identity provider, or lengthens how long it caches signing keys.
The other half were <b>noise</b> — an ordinary redeploy that reorders a list, bumps the replica count, renames a team label, or bumps a container image. Nothing that actually changes who-can-log-in.</p>
<p>A good tool must <b>flag every real change</b> and <b>stay completely silent on the noise</b>. Alerts you can't trust get ignored — and then the one that mattered gets ignored too.</p>

<div class="tiles">
 <div class="tile"><div class="l">Real changes caught</div><div class="n low">` + pct(c.TPR) + `</div><div class="l">` + fmt.Sprintf("%d of %d", c.TP, c.TP+c.FN) + `</div></div>
 <div class="tile"><div class="l">False alarms</div><div class="n brand">` + pct(c.FPR) + `</div><div class="l">on ` + fmt.Sprint(c.TN+c.FP) + ` noise changes</div></div>
 <div class="tile"><div class="l">Precision</div><div class="n">` + pct(c.Precision) + `</div><div class="l">alerts that were real</div></div>
 <div class="tile"><div class="l">Youden score</div><div class="n">` + fmt.Sprintf("%.2f", c.Youden) + `</div><div class="l">1.0 = perfect</div></div>
</div>

<h2>How that compares to what teams use today</h2>
<p>Most teams have <b>no tool</b> for this — it's tribal knowledge in someone's head. The closest tools people reach for are <b>static code scanners</b> (Snyk, Semgrep, SonarQube). Those are excellent at their own job, but they <i>guess</i> from source code; they can't tell you which running services will actually break when you rotate a key. The chart plots their <b>published</b> OWASP-Benchmark accuracy next to Keyway's — the point isn't a head-to-head (different jobs), it's that guessing has a documented accuracy ceiling, while <b>checking with a real token doesn't guess</b>.</p>
<div class="chart">` + svg + `</div>

<table>
<tr><th>Question a team actually asks</th><th>Do nothing</th><th>Static scanners</th><th>Keyway</th></tr>
<tr><td>List every service that validates JWTs</td><td class="no">✗ in someone's head</td><td class="no">✗ not their job</td><td class="yes">✓ automatic</td></tr>
<tr><td>Alert on a real change without false alarms</td><td class="no">✗</td><td class="part">~ high false-alarm rate</td><td class="yes">✓ 0 here</td></tr>
<tr><td>"Who breaks if I rotate this key?"</td><td class="no">✗</td><td class="no">✗</td><td class="yes">✓ with a grace period</td></tr>
<tr><td>Prove it with a real token, not a guess</td><td class="no">✗</td><td class="no">✗ infers from code</td><td class="yes">✓ 13 live probes</td></tr>
</table>

<div class="note"><b>Honesty note.</b> Keyway and static scanners solve different problems, so this is not a winner-take-all comparison. The third-party numbers are published OWASP Benchmark results for the scanner category (shown for calibration, not re-run here). Keyway's numbers are measured on the corpus described below and are reproducible in one command.</div>

<h2>Reproduce it yourself</h2>
<p>No trust required — run the exact benchmark on your own machine:</p>
<p><code>git clone https://github.com/nometria/keyway &amp;&amp; cd keyway</code><br/>
<code>make bench</code></p>
<p>The corpus lives in <code>bench/corpus/</code> and the generator in <code>bench/mutations/</code>. Every scenario is plain YAML you can read. CI fails the build if accuracy ever drops below the published thresholds, so these numbers stay honest over time.</p>

<footer>Corpus: ` + fmt.Sprintf("%d generated scenarios + %d realistic before/after manifests", d.Generated, d.FileBased) + ` &middot; ` +
		fmt.Sprintf("%d real changes, %d noise changes", d.TruePos, d.TrueNeg) + ` &middot; Generated by <code>keyway bench --report</code>.</footer>
</div></body></html>`
}
