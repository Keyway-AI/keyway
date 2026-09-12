//go:build js && wasm

// Command playground exposes Keyway's agent-auth analyzer to the browser via
// WebAssembly. It is the exact same analyzer the CLI uses (internal/agentauth) —
// no reimplementation — compiled to run entirely client-side, so a pasted token
// never leaves the page. Build it with `make playground`.
package main

import (
	"encoding/json"
	"syscall/js"
	"time"

	"github.com/Keyway-AI/keyway/internal/agentauth"
)

// inspect is exposed to JS as window.keywayInspect(token, options?) and returns a
// JSON string: {count, findings:[{threat_id,severity,message}], error?}.
func inspect(_ js.Value, args []js.Value) any {
	if len(args) < 1 || args[0].Type() != js.TypeString {
		return marshal(nil, "no token provided")
	}
	pol := agentauth.Policy{Now: time.Now()}
	if len(args) >= 2 && args[1].Type() == js.TypeObject {
		o := args[1]
		if v := o.Get("audience"); v.Type() == js.TypeString {
			pol.Audience = v.String()
		}
		if v := o.Get("requireDelegation"); v.Type() == js.TypeBoolean {
			pol.RequireDelegation = v.Bool()
		}
	}
	findings, err := agentauth.Analyze(args[0].String(), pol)
	if err != nil {
		return marshal(nil, err.Error())
	}
	return marshal(findings, "")
}

func marshal(findings []agentauth.Finding, errMsg string) string {
	m := map[string]any{"count": len(findings), "findings": findings}
	if errMsg != "" {
		m["error"] = errMsg
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func main() {
	js.Global().Set("keywayInspect", js.FuncOf(inspect))
	select {} // keep the Go runtime alive so the exported function stays callable
}
