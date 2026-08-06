package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Keyway-AI/keyway/bench/mutations"
	"github.com/Keyway-AI/keyway/internal/model"
)

// This file scales the *realistic* corpus (KI-18). Unlike mutations.Generate,
// which mutates model structs directly, the scenarios here are rendered as real
// Istio / Envoy / Kubernetes YAML and scored through the actual discovery
// pipeline (L1) into diff (L3). Ground truth comes from the mutation applied to
// the manifest, independent of the diff's own field semantics — so a perfect
// score is evidence the end-to-end derive→classify path is correct at breadth,
// not just self-consistent. Generation is deterministic (index-driven, no RNG)
// so the corpus is reproducible in CI.

// genScenario is a rendered before/after pair of manifest files with its ground
// truth. Keys in the file maps are manifest filenames.
type genScenario struct {
	name   string
	before map[string]string
	after  map[string]string
	exp    mutations.Expected
}

// parameter pools give each generated scenario genuine variety (different names,
// namespaces, issuers, audiences) so the corpus is not overfit to one shape.
var (
	genNamespaces = []string{"prod", "staging", "payments", "identity", "edge"}
	genIssuers    = []string{
		"https://kc/realms/main",
		"https://kc/realms/partner",
		"https://idp.corp/realms/apps",
		"https://auth.internal/realms/svc",
	}
	genClaims    = []string{"dept", "scope", "tier", "role", "tenant"}
	genClaimVals = []string{"finance", "orders:write", "gold", "admin", "acme"}
)

// factory renders one scenario given a unique index. The full catalog below is
// cycled to produce as many scenarios as requested.
type factory func(i int) genScenario

// --- manifest renderers ------------------------------------------------------

type jwtRule struct {
	issuer string
	auds   []string
}

func istioRA(ns, name string, rules []jwtRule) string {
	var b strings.Builder
	fmt.Fprintf(&b, "apiVersion: security.istio.io/v1\nkind: RequestAuthentication\nmetadata:\n  name: %s\n  namespace: %s\nspec:\n  selector:\n    matchLabels:\n      app: %s\n  jwtRules:\n", name, ns, name)
	for _, r := range rules {
		fmt.Fprintf(&b, "    - issuer: %q\n      audiences: [%s]\n", r.issuer, quoteJoin(r.auds))
	}
	return b.String()
}

func istioAuthPolicy(ns, name, claim string, values []string) string {
	return fmt.Sprintf(`apiVersion: security.istio.io/v1
kind: AuthorizationPolicy
metadata:
  name: %s-require-%s
  namespace: %s
spec:
  selector:
    matchLabels:
      app: %s
  rules:
    - when:
        - key: request.auth.claims[%s]
          values: [%s]
`, name, claim, ns, name, claim, quoteJoin(values))
}

func envoyJWT(provider, issuer string, auds []string, ttlSec int, extra string) string {
	return fmt.Sprintf(`http_filters:
  - name: envoy.filters.http.jwt_authn
    typed_config:
      providers:
        %s:
          issuer: %q
          audiences: [%s]
%s          remote_jwks:
            http_uri:
              uri: "%s/protocol/openid-connect/certs"
            cache_duration: "%ds"
`, provider, issuer, quoteJoin(auds), extra, issuer, ttlSec)
}

func quoteJoin(ss []string) string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = fmt.Sprintf("%q", s)
	}
	return strings.Join(out, ", ")
}

// pick indexes a pool deterministically.
func pick(pool []string, i int) string { return pool[i%len(pool)] }

func istioConsumerID(ns, name string) string { return "k8s://local/" + ns + "/" + name }

// --- true-positive factories -------------------------------------------------

func fAudienceAdded(i int) genScenario {
	ns, name := pick(genNamespaces, i), fmt.Sprintf("svc-aa-%d", i)
	iss := pick(genIssuers, i)
	before := istioRA(ns, name, []jwtRule{{iss, []string{name}}})
	after := istioRA(ns, name, []jwtRule{{iss, []string{name, name + "-admin"}}})
	return genScenario{"audience-added", f1(before), f1(after),
		mutations.Expected{Detected: true, Class: model.ChangeWidened, Severity: model.SeverityMedium, Consumer: istioConsumerID(ns, name)}}
}

func fAudienceRemoved(i int) genScenario {
	ns, name := pick(genNamespaces, i), fmt.Sprintf("svc-ar-%d", i)
	iss := pick(genIssuers, i)
	before := istioRA(ns, name, []jwtRule{{iss, []string{name, name + "-legacy"}}})
	after := istioRA(ns, name, []jwtRule{{iss, []string{name}}})
	return genScenario{"audience-removed", f1(before), f1(after),
		mutations.Expected{Detected: true, Class: model.ChangeNarrowed, Severity: model.SeverityLow, Consumer: istioConsumerID(ns, name)}}
}

func fIssuerAdded(i int) genScenario {
	ns, name := pick(genNamespaces, i), fmt.Sprintf("svc-ia-%d", i)
	iss1, iss2 := pick(genIssuers, i), pick(genIssuers, i+1)
	before := istioRA(ns, name, []jwtRule{{iss1, []string{name}}})
	after := istioRA(ns, name, []jwtRule{{iss1, []string{name}}, {iss2, []string{name}}})
	return genScenario{"issuer-added", f1(before), f1(after),
		mutations.Expected{Detected: true, Class: model.ChangeWidened, Severity: model.SeverityHigh, Consumer: istioConsumerID(ns, name)}}
}

func fIssuerRemoved(i int) genScenario {
	ns, name := pick(genNamespaces, i), fmt.Sprintf("svc-ir-%d", i)
	iss1, iss2 := pick(genIssuers, i), pick(genIssuers, i+1)
	before := istioRA(ns, name, []jwtRule{{iss1, []string{name}}, {iss2, []string{name}}})
	after := istioRA(ns, name, []jwtRule{{iss1, []string{name}}})
	return genScenario{"issuer-removed", f1(before), f1(after),
		mutations.Expected{Detected: true, Class: model.ChangeNarrowed, Severity: model.SeverityLow, Consumer: istioConsumerID(ns, name)}}
}

func fClaimAdded(i int) genScenario {
	ns, name := pick(genNamespaces, i), fmt.Sprintf("svc-ca-%d", i)
	iss := pick(genIssuers, i)
	claim, val := pick(genClaims, i), pick(genClaimVals, i)
	ra := istioRA(ns, name, []jwtRule{{iss, []string{name}}})
	after := ra + "---\n" + istioAuthPolicy(ns, name, claim, []string{val})
	return genScenario{"required-claim-added", f1(ra), f1(after),
		mutations.Expected{Detected: true, Class: model.ChangeNarrowed, Severity: model.SeverityLow, Consumer: istioConsumerID(ns, name)}}
}

func fClaimRemoved(i int) genScenario {
	ns, name := pick(genNamespaces, i), fmt.Sprintf("svc-cr-%d", i)
	iss := pick(genIssuers, i)
	claim, val := pick(genClaims, i), pick(genClaimVals, i)
	ra := istioRA(ns, name, []jwtRule{{iss, []string{name}}})
	before := ra + "---\n" + istioAuthPolicy(ns, name, claim, []string{val})
	return genScenario{"required-claim-removed", f1(before), f1(ra),
		mutations.Expected{Detected: true, Class: model.ChangeWidened, Severity: model.SeverityCritical, Consumer: istioConsumerID(ns, name)}}
}

func fEnvoyAudienceAdded(i int) genScenario {
	prov := fmt.Sprintf("prov-ea-%d", i)
	iss := pick(genIssuers, i)
	before := envoyJWT(prov, iss, []string{prov}, 300, "")
	after := envoyJWT(prov, iss, []string{prov, prov + "-admin"}, 300, "")
	return genScenario{"envoy-audience-added", map[string]string{"envoy.yaml": before}, map[string]string{"envoy.yaml": after},
		mutations.Expected{Detected: true, Class: model.ChangeWidened, Severity: model.SeverityMedium, Consumer: "route://envoy/" + prov}}
}

func fEnvoyCacheUp(i int) genScenario {
	prov := fmt.Sprintf("prov-cu-%d", i)
	iss := pick(genIssuers, i)
	before := envoyJWT(prov, iss, []string{prov}, 300, "")
	after := envoyJWT(prov, iss, []string{prov}, 3600, "")
	return genScenario{"envoy-cache-up", map[string]string{"envoy.yaml": before}, map[string]string{"envoy.yaml": after},
		mutations.Expected{Detected: true, Class: model.ChangeNarrowed, Severity: model.SeverityMedium, Consumer: "route://envoy/" + prov}}
}

func fEnvoyCacheDown(i int) genScenario {
	prov := fmt.Sprintf("prov-cd-%d", i)
	iss := pick(genIssuers, i)
	before := envoyJWT(prov, iss, []string{prov}, 3600, "")
	after := envoyJWT(prov, iss, []string{prov}, 300, "")
	return genScenario{"envoy-cache-down", map[string]string{"envoy.yaml": before}, map[string]string{"envoy.yaml": after},
		mutations.Expected{Detected: true, Class: model.ChangeNarrowed, Severity: model.SeverityLow, Consumer: "route://envoy/" + prov}}
}

func fEnvoyIssuerMigration(i int) genScenario {
	prov := fmt.Sprintf("prov-im-%d", i)
	iss1, iss2 := pick(genIssuers, i), pick(genIssuers, i+2)
	before := envoyJWT(prov, iss1, []string{prov}, 300, "")
	after := envoyJWT(prov, iss2, []string{prov}, 300, "")
	return genScenario{"envoy-issuer-migration", map[string]string{"envoy.yaml": before}, map[string]string{"envoy.yaml": after},
		mutations.Expected{Detected: true, Class: model.ChangeWidened, Severity: model.SeverityHigh, Consumer: "route://envoy/" + prov}}
}

func fConsumerAdded(i int) genScenario {
	ns := pick(genNamespaces, i)
	a, b := fmt.Sprintf("svc-ex-%d", i), fmt.Sprintf("svc-new-%d", i)
	iss := pick(genIssuers, i)
	before := istioRA(ns, a, []jwtRule{{iss, []string{a}}})
	after := before + "---\n" + istioRA(ns, b, []jwtRule{{iss, []string{b}}})
	return genScenario{"consumer-added", f1(before), f1(after),
		mutations.Expected{Detected: true, Class: model.ChangeNeutral, Severity: model.SeverityLow, Consumer: istioConsumerID(ns, b)}}
}

func fConsumerRemoved(i int) genScenario {
	ns := pick(genNamespaces, i)
	a, b := fmt.Sprintf("svc-keep-%d", i), fmt.Sprintf("svc-gone-%d", i)
	iss := pick(genIssuers, i)
	keep := istioRA(ns, a, []jwtRule{{iss, []string{a}}})
	before := keep + "---\n" + istioRA(ns, b, []jwtRule{{iss, []string{b}}})
	return genScenario{"consumer-removed", f1(before), f1(keep),
		mutations.Expected{Detected: true, Class: model.ChangeNeutral, Severity: model.SeverityLow, Consumer: istioConsumerID(ns, b)}}
}

// --- true-negative (no-op) factories -----------------------------------------

func fReorderAudiences(i int) genScenario {
	ns, name := pick(genNamespaces, i), fmt.Sprintf("svc-ro-%d", i)
	iss := pick(genIssuers, i)
	before := istioRA(ns, name, []jwtRule{{iss, []string{name, name + "-b", name + "-c"}}})
	after := istioRA(ns, name, []jwtRule{{iss, []string{name + "-c", name, name + "-b"}}})
	return genScenario{"reorder-audiences", f1(before), f1(after), noopExp(istioConsumerID(ns, name))}
}

func fReorderJwtRules(i int) genScenario {
	ns, name := pick(genNamespaces, i), fmt.Sprintf("svc-rj-%d", i)
	iss1, iss2 := pick(genIssuers, i), pick(genIssuers, i+1)
	before := istioRA(ns, name, []jwtRule{{iss1, []string{name}}, {iss2, []string{name}}})
	after := istioRA(ns, name, []jwtRule{{iss2, []string{name}}, {iss1, []string{name}}})
	return genScenario{"reorder-jwtrules", f1(before), f1(after), noopExp(istioConsumerID(ns, name))}
}

func fAddAnnotation(i int) genScenario {
	ns, name := pick(genNamespaces, i), fmt.Sprintf("svc-an-%d", i)
	iss := pick(genIssuers, i)
	before := istioRA(ns, name, []jwtRule{{iss, []string{name}}})
	after := strings.Replace(before,
		fmt.Sprintf("  namespace: %s\n", ns),
		fmt.Sprintf("  namespace: %s\n  annotations:\n    argocd.argoproj.io/sync-wave: \"2\"\n", ns), 1)
	return genScenario{"add-annotation", f1(before), f1(after), noopExp(istioConsumerID(ns, name))}
}

func fRelabelSelector(i int) genScenario {
	ns, name := pick(genNamespaces, i), fmt.Sprintf("svc-rl-%d", i)
	iss := pick(genIssuers, i)
	before := istioRA(ns, name, []jwtRule{{iss, []string{name}}})
	// The selector labels are cosmetic to the token contract (identity is
	// name/namespace); adding one must not change the consumer.
	after := strings.Replace(before,
		fmt.Sprintf("      app: %s\n", name),
		fmt.Sprintf("      app: %s\n      version: v2\n", name), 1)
	return genScenario{"relabel-selector", f1(before), f1(after), noopExp(istioConsumerID(ns, name))}
}

func fUnrelatedResource(i int) genScenario {
	ns, name := pick(genNamespaces, i), fmt.Sprintf("svc-ur-%d", i)
	iss := pick(genIssuers, i)
	before := istioRA(ns, name, []jwtRule{{iss, []string{name}}})
	after := before + fmt.Sprintf("---\napiVersion: v1\nkind: ConfigMap\nmetadata: { name: %s-cfg, namespace: %s }\ndata:\n  MAX_CONNS: \"256\"\n", name, ns)
	return genScenario{"unrelated-resource", f1(before), f1(after), noopExp(istioConsumerID(ns, name))}
}

func fEnvoyNoisy(i int) genScenario {
	prov := fmt.Sprintf("prov-nz-%d", i)
	iss := pick(genIssuers, i)
	before := envoyJWT(prov, iss, []string{prov}, 300, "          forward: true\n")
	after := envoyJWT(prov, iss, []string{prov}, 300, "          forward: false\n          payload_in_metadata: \"jwt_payload\"\n")
	return genScenario{"envoy-noisy", map[string]string{"envoy.yaml": before}, map[string]string{"envoy.yaml": after},
		noopExp("route://envoy/" + prov)}
}

func fReformat(i int) genScenario {
	ns, name := pick(genNamespaces, i), fmt.Sprintf("svc-rf-%d", i)
	iss := pick(genIssuers, i)
	before := istioRA(ns, name, []jwtRule{{iss, []string{name, name + "-b"}}})
	// Block-style audiences instead of flow-style; same set.
	after := fmt.Sprintf("apiVersion: security.istio.io/v1\nkind: RequestAuthentication\nmetadata:\n  name: %s\n  namespace: %s\nspec:\n  selector:\n    matchLabels:\n      app: %s\n  jwtRules:\n    - issuer: %q\n      audiences:\n        - %s\n        - %s\n",
		name, ns, name, iss, name, name+"-b")
	return genScenario{"reformat", f1(before), f1(after), noopExp(istioConsumerID(ns, name))}
}

func fCommentOnly(i int) genScenario {
	ns, name := pick(genNamespaces, i), fmt.Sprintf("svc-co-%d", i)
	iss := pick(genIssuers, i)
	before := istioRA(ns, name, []jwtRule{{iss, []string{name}}})
	after := "# managed by platform-team\n" + before + "# end\n"
	return genScenario{"comment-only", f1(before), f1(after), noopExp(istioConsumerID(ns, name))}
}

// helpers ---------------------------------------------------------------------

// f1 wraps a single istio manifest under a stable filename.
func f1(content string) map[string]string { return map[string]string{"istio.yaml": content} }

func noopExp(consumer string) mutations.Expected {
	return mutations.Expected{Detected: false, Consumer: consumer}
}

// catalog is the full set of factories, cycled to produce the corpus. It leans
// slightly true-positive (12 TP : 8 TN); the generated struct corpus supplies
// the exact 50/50 split, so this set prioritizes risk-class breadth.
var catalog = []factory{
	fAudienceAdded, fAudienceRemoved, fIssuerAdded, fIssuerRemoved,
	fClaimAdded, fClaimRemoved, fEnvoyAudienceAdded, fEnvoyCacheUp,
	fEnvoyCacheDown, fEnvoyIssuerMigration, fConsumerAdded, fConsumerRemoved,
	fReorderAudiences, fReorderJwtRules, fAddAnnotation, fRelabelSelector,
	fUnrelatedResource, fEnvoyNoisy, fReformat, fCommentOnly,
}

// genRealistic produces n rendered scenarios by cycling the factory catalog.
func genRealistic(n int) []genScenario {
	out := make([]genScenario, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, catalog[i%len(catalog)](i))
	}
	return out
}

// loadGeneratedRealistic renders n scenarios to a temp tree and runs them
// through the real discovery pipeline, returning scored-ready Scenarios plus a
// cleanup func. Scenarios whose discovery errors are skipped.
func loadGeneratedRealistic(n int) ([]mutations.Scenario, func(), error) {
	root, err := os.MkdirTemp("", "keyway-realistic-*")
	if err != nil {
		return nil, func() {}, err
	}
	cleanup := func() { _ = os.RemoveAll(root) }

	gens := genRealistic(n)
	out := make([]mutations.Scenario, 0, len(gens))
	for i, g := range gens {
		dir := filepath.Join(root, fmt.Sprintf("s%05d", i))
		if err := writeManifests(filepath.Join(dir, "before"), g.before); err != nil {
			cleanup()
			return nil, func() {}, err
		}
		if err := writeManifests(filepath.Join(dir, "after"), g.after); err != nil {
			cleanup()
			return nil, func() {}, err
		}
		before, e1 := discoverDir(filepath.Join(dir, "before"))
		after, e2 := discoverDir(filepath.Join(dir, "after"))
		if e1 != nil || e2 != nil {
			continue
		}
		out = append(out, mutations.Scenario{
			ID: fmt.Sprintf("gen-%05d-%s", i, g.name), Name: g.name,
			Before: before, After: after, Expected: g.exp,
		})
	}
	return out, cleanup, nil
}

func writeManifests(dir string, files map[string]string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}
