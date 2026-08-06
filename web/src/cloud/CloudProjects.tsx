import { useCallback, useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { cloud, isUnauthorized } from "./api";
import type { Project, Source } from "./api";
import { useCloudAuth } from "./CloudAuth";
import { Button, Card, Empty, Field, Input, Modal, Pill } from "../components/ui";
import { readManifestFiles } from "./manifests";

/** Dashboard: the user's projects, each a tracked auth surface (repo or upload). */
export default function CloudProjects() {
  const { refresh } = useCloudAuth();
  const [projects, setProjects] = useState<Project[] | null>(null);
  const [error, setError] = useState("");
  const [creating, setCreating] = useState(false);

  const load = useCallback(async () => {
    try {
      setProjects(await cloud.listProjects());
      setError("");
    } catch (e) {
      if (isUnauthorized(e)) {
        void refresh();
        return;
      }
      setError(e instanceof Error ? e.message : "Failed to load projects");
    }
  }, [refresh]);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <div>
      <div className="mb-6 flex flex-wrap items-end justify-between gap-3">
        <div>
          <div className="eyebrow">Workspace</div>
          <h1 className="mt-1 text-h1 font-semibold tracking-tight display">Projects</h1>
          <p className="mt-1 text-body text-muted">
            Each project tracks one auth surface. Analyze it to derive its token contract and watch for drift.
          </p>
        </div>
        <Button variant="primary" className="glow-accent" onClick={() => setCreating(true)}>
          + New project
        </Button>
      </div>

      {error && (
        <div className="mb-4 rounded-md border border-critical/30 bg-critical/8 px-3 py-2.5 text-caption text-critical">
          {error}
        </div>
      )}

      {projects === null ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {[0, 1, 2].map((i) => (
            <div key={i} className="h-32 animate-pulse rounded-xl border border-border bg-surface-2/60" />
          ))}
        </div>
      ) : projects.length === 0 ? (
        <Card>
          <Empty>
            <p className="text-body">No projects yet.</p>
            <p className="mt-1 text-caption">
              Create one from a GitHub repository or by uploading your Istio / Envoy / OIDC config.
            </p>
            <Button variant="primary" className="mt-4 glow-accent" onClick={() => setCreating(true)}>
              + New project
            </Button>
          </Empty>
        </Card>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {projects.map((p) => (
            <ProjectCard key={p.id} project={p} />
          ))}
        </div>
      )}

      {creating && (
        <NewProjectModal
          onClose={() => setCreating(false)}
          onCreated={() => {
            setCreating(false);
            void load();
          }}
        />
      )}
    </div>
  );
}

function ProjectCard({ project }: { project: Project }) {
  const src = project.source;
  return (
    <Link
      to={`/cloud/projects/${project.id}`}
      className="group flex flex-col justify-between rounded-xl border border-border bg-surface p-5 shadow-xs transition hover:border-border-strong hover:shadow-md"
    >
      <div>
        <div className="flex items-center justify-between gap-2">
          <h3 className="truncate text-[1.05rem] font-semibold tracking-tight">{project.name}</h3>
          <Pill className="shrink-0 capitalize">{src.kind}</Pill>
        </div>
        <p className="mt-1.5 truncate text-caption text-muted">
          {src.kind === "github" ? `${src.repo}${src.ref ? ` · ${src.ref}` : ""}` : "Uploaded configuration"}
        </p>
      </div>
      <div className="mt-4 flex items-center justify-between text-caption text-muted">
        <span>{project.latest_analysis_id ? "Analyzed" : "Not analyzed yet"}</span>
        <span className="font-medium text-accent transition group-hover:translate-x-0.5">Open →</span>
      </div>
    </Link>
  );
}

/* ── New project ────────────────────────────────────────────────────────── */

function NewProjectModal({ onClose, onCreated }: { onClose: () => void; onCreated: () => void }) {
  const [kind, setKind] = useState<"upload" | "github">("upload");
  const [name, setName] = useState("");
  const [repo, setRepo] = useState("");
  const [ref, setRef] = useState("");
  const [path, setPath] = useState("");
  const [files, setFiles] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");
  const fileInput = useRef<HTMLInputElement>(null);

  const fileNames = Object.keys(files);

  async function onPick(list: FileList | null) {
    if (!list) return;
    try {
      setFiles(await readManifestFiles(list));
      setErr("");
    } catch (e) {
      setErr(e instanceof Error ? e.message : "Could not read files");
    }
  }

  async function submit() {
    const trimmed = name.trim();
    if (!trimmed) return setErr("Name is required.");
    let source: Source;
    if (kind === "github") {
      if (!repo.trim().includes("/")) return setErr('Repository must be "owner/name".');
      source = { kind: "github", repo: repo.trim(), ref: ref.trim() || undefined, path: path.trim() || undefined };
    } else {
      if (fileNames.length === 0) return setErr("Add at least one config file.");
      source = { kind: "upload" };
    }
    setBusy(true);
    setErr("");
    try {
      const project = await cloud.createProject(trimmed, source);
      // For uploads, run the first analysis immediately so the project isn't empty.
      if (kind === "upload") {
        try {
          await cloud.analyze(project.id, files);
        } catch {
          /* project still created; detail page will prompt to analyze */
        }
      }
      onCreated();
    } catch (e) {
      setErr(e instanceof Error ? e.message : "Failed to create project");
      setBusy(false);
    }
  }

  return (
    <Modal
      title="New project"
      onClose={onClose}
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={busy}>
            Cancel
          </Button>
          <Button variant="primary" onClick={submit} loading={busy}>
            {kind === "upload" ? "Create & analyze" : "Create"}
          </Button>
        </>
      }
    >
      <div className="space-y-4">
        <div className="grid grid-cols-2 gap-2">
          <SourceTab active={kind === "upload"} onClick={() => setKind("upload")} title="Upload config" hint="Istio / Envoy / OIDC YAML" />
          <SourceTab active={kind === "github"} onClick={() => setKind("github")} title="Connect GitHub" hint="Sync from a repository" />
        </div>

        <Field label="Project name">
          <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="payments-mesh" autoFocus />
        </Field>

        {kind === "github" ? (
          <>
            <Field label="Repository" hint="owner/name — Keyway reads its config with your GitHub access.">
              <Input value={repo} onChange={(e) => setRepo(e.target.value)} placeholder="acme/payments" />
            </Field>
            <div className="grid grid-cols-2 gap-3">
              <Field label="Branch / tag" hint="default main">
                <Input value={ref} onChange={(e) => setRef(e.target.value)} placeholder="main" />
              </Field>
              <Field label="Subdirectory" hint="optional">
                <Input value={path} onChange={(e) => setPath(e.target.value)} placeholder="deploy/" />
              </Field>
            </div>
          </>
        ) : (
          <Field label="Config files" hint="YAML manifests — AuthorizationPolicy, RequestAuthentication, Envoy JWT filters, OIDC clients.">
            <div className="rounded-md border border-dashed border-border-strong bg-surface-2 p-4 text-center">
              <input
                ref={fileInput}
                type="file"
                multiple
                accept=".yaml,.yml"
                className="hidden"
                onChange={(e) => onPick(e.target.files)}
              />
              <Button variant="secondary" size="sm" onClick={() => fileInput.current?.click()}>
                Choose files
              </Button>
              {fileNames.length > 0 && (
                <div className="mt-3 flex flex-wrap justify-center gap-1.5">
                  {fileNames.slice(0, 8).map((n) => (
                    <Pill key={n}>{n}</Pill>
                  ))}
                  {fileNames.length > 8 && <Pill>+{fileNames.length - 8} more</Pill>}
                </div>
              )}
            </div>
          </Field>
        )}

        {err && <p className="text-caption text-critical">{err}</p>}
      </div>
    </Modal>
  );
}

function SourceTab({ active, onClick, title, hint }: { active: boolean; onClick: () => void; title: string; hint: string }) {
  return (
    <button
      onClick={onClick}
      className={`rounded-lg border px-3 py-2.5 text-left transition ${
        active ? "border-accent bg-accent-soft" : "border-border bg-surface hover:bg-surface-2"
      }`}
    >
      <div className="text-caption font-semibold text-text">{title}</div>
      <div className="text-micro text-muted">{hint}</div>
    </button>
  );
}
