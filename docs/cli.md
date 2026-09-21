# CLI JSON examples

These examples use the bundled reference manifests and threat catalog. The
`jq` filters keep each result short; remove the filter to inspect the full JSON
response. `keyway discover` selects JSON with `--output json`, while commands
such as `threats coverage` use the global `--json` flag.

## Discover consumers

From the repository root, scan the bundled Istio, Kubernetes, and Envoy
manifests. The discovery command reads these files and does not need a running
database:

```sh
keyway discover --path testdata/discovery/reference --output json \
  | jq '[.[] | {name, kind, namespace, probeable}]'
```

Example output:

```json
[
  {
    "name": "payments-api",
    "kind": "service",
    "namespace": "prod",
    "probeable": true
  },
  {
    "name": "orders-api",
    "kind": "service",
    "namespace": "prod",
    "probeable": true
  },
  {
    "name": "legacy-reporting",
    "kind": "service",
    "namespace": "data",
    "probeable": true
  },
  {
    "name": "inventory-worker",
    "kind": "service",
    "namespace": "prod",
    "probeable": true
  },
  {
    "name": "edge-gateway",
    "kind": "gateway_route",
    "namespace": null,
    "probeable": true
  }
]
```

Each item represents a discovered consumer. The full response also includes
stable IDs, expected issuers and audiences, provenance, and confidence data.

## Check agent threat coverage

Filter the threat catalog to agent-auth risks and keep the summary fields from
the JSON report:

```sh
keyway threats coverage --domain agent --json \
  | jq '{total, covered, gaps, percent, domains}'
```

Example output:

```json
{
  "total": 15,
  "covered": 6,
  "gaps": 9,
  "percent": 40,
  "domains": [
    {"covered": 6, "domain": "agent", "percent": 40, "total": 15}
  ]
}
```

The counts come from the checked-in threat catalog and detection mappings. Run
`keyway threats coverage --json` without `--domain agent` to include both JWT
and agent-auth risks.
