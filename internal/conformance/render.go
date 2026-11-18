package conformance

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
)

// WriteJSON writes the report as indented JSON.
func (r Report) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// WriteTable writes a human-readable conformance report: a per-verifier summary,
// the findings (adversarial tokens wrongly accepted), and per-threat prevalence.
func (r Report) WriteTable(w io.Writer) {
	fmt.Fprintln(w, "Verifier conformance report")
	fmt.Fprintln(w, "===========================")
	fmt.Fprintf(w, "%d verifier(s), %d trusted the test issuer; %d clean, %d with findings (%d finding(s) total).\n\n",
		r.Summary.Verifiers, r.Summary.Verified, r.Summary.Clean, r.Summary.WithFindings, r.Summary.TotalFindings)

	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "VERIFIER\tTRUSTS ISSUER\tATTACKS\tACCEPTED\tVERDICT")
	for _, v := range r.Verifiers {
		verdict := "clean"
		switch {
		case !v.Verified:
			verdict = "UNVERIFIED (rejected control)"
		case v.Accepted > 0:
			verdict = fmt.Sprintf("%d FINDING(S)", v.Accepted)
		}
		fmt.Fprintf(tw, "%s\t%t\t%d\t%d\t%s\n", v.ConsumerID, v.Verified, v.Attacks, v.Accepted, verdict)
	}
	_ = tw.Flush()

	if r.Summary.TotalFindings > 0 {
		fmt.Fprintln(w, "\nFindings (an adversarial token a correct verifier MUST reject, but this one accepted):")
		ft := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
		fmt.Fprintln(ft, "  VERIFIER\tTHREAT\tSEVERITY\tTOKEN\tSTATUS")
		for _, v := range r.Verifiers {
			if !v.Verified {
				continue
			}
			for _, f := range v.Findings {
				title := f.Title
				if title == "" {
					title = f.ThreatID
				}
				fmt.Fprintf(ft, "  %s\t%s (%s)\t%s\t%s\t%d\n",
					v.ConsumerID, f.ThreatID, title, f.Severity, f.Name, f.StatusCode)
			}
		}
		_ = ft.Flush()
	}

	if len(r.Prevalence) > 0 {
		fmt.Fprintln(w, "\nPer-threat prevalence (across verifiers that trusted the issuer):")
		pt := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
		fmt.Fprintln(pt, "  THREAT\tSEVERITY\tACCEPTING")
		for _, p := range r.Prevalence {
			if p.Accepting == 0 {
				continue // only show weaknesses at least one verifier exhibited
			}
			title := p.Title
			if title == "" {
				title = p.Name
			}
			fmt.Fprintf(pt, "  %s %s\t%s\t%d/%d\n", p.ThreatID, title, p.Severity, p.Accepting, p.Evaluated)
		}
		_ = pt.Flush()
	}
}
