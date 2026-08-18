"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { api, apiBase } from "@/lib/api";
import { buyerStatusLabel, documentStatusLabel } from "@/lib/cases";

type Item = {
  document_type_code: string;
  label: string;
  category: string;
  required: boolean;
  instructions: string;
  status: string;
  document_id?: string;
  review_notes?: string;
};

type Application = {
  id: string;
  status: string;
  next_action_text: string;
  assigned_advisor_name?: string;
};

export default function DocumentsPage() {
  const [items, setItems] = useState<Item[]>([]);
  const [appID, setAppID] = useState("");
  const [file, setFile] = useState<Application | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState<string | null>(null);
  const [message, setMessage] = useState("");

  async function load() {
    const [data, mine] = await Promise.all([
      api<{ application_id: string; items: Item[] }>("/api/v1/documents/checklist"),
      api<{ application: Application }>("/api/v1/applications/me").catch(() => null),
    ]);
    setAppID(data.application_id);
    setItems(data.items || []);
    setFile(mine?.application || null);
  }

  useEffect(() => {
    load().catch((e) => setError(e.message));
  }, []);

  async function onUpload(code: string, fileObj: File | null) {
    if (!fileObj || !appID) return;
    setBusy(code);
    setError("");
    setMessage("");
    try {
      const fd = new FormData();
      fd.append("application_id", appID);
      fd.append("document_type_code", code);
      fd.append("file", fileObj);
      const res = await fetch(`${apiBase}/api/v1/documents/upload`, {
        method: "POST",
        credentials: "include",
        body: fd,
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "Upload failed");
      setMessage(`${fileObj.name} uploaded.`);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Upload failed");
    } finally {
      setBusy(null);
    }
  }

  const required = items.filter((i) => i.required);
  const accepted = required.filter((i) => i.status === "accepted").length;
  const sentBack = items.filter((i) => i.status === "requires_replacement" || i.status === "rejected");

  return (
    <div className="mx-auto max-w-3xl px-5 py-10">
      <Link href="/app" className="text-sm font-semibold text-leaf">
        ← Journey
      </Link>
      <div className="mt-4 flex items-start justify-between gap-4">
        <div>
          <h1 className="font-[family-name:var(--font-display)] text-3xl font-semibold">Your documents</h1>
          <p className="mt-2 text-sm text-muted">
            Upload PDF, JPG, or PNG (max 10MB). If your advisor sends something back, the reason appears on that item.
          </p>
        </div>
        <button
          type="button"
          onClick={() =>
            api("/api/v1/applications/request-advisor", { method: "POST" })
              .then(() => setMessage("Advisor requested. They will review your file."))
              .catch((e) => setError(e.message))
          }
          className="shrink-0 rounded-md border border-[color:var(--line)] px-3 py-2 text-sm font-semibold"
        >
          Request advisor
        </button>
      </div>

      {file && (
        <p className="mt-4 text-sm">
          <span className="font-semibold text-leaf">{buyerStatusLabel(file.status)}</span>
          {file.next_action_text ? <span className="text-muted"> — {file.next_action_text}</span> : null}
        </p>
      )}
      <p className="mt-2 text-sm font-semibold text-leaf">
        {accepted} of {required.length || items.length} required accepted
        {sentBack.length ? ` · ${sentBack.length} to replace` : ""}
      </p>

      {error && <p className="mt-4 text-sm text-[color:var(--danger)]">{error}</p>}
      {message && <p className="mt-4 text-sm text-leaf">{message}</p>}

      <ul className="mt-8 space-y-4">
        {items.map((item) => (
          <li key={item.document_type_code} className="rounded-xl border border-[color:var(--line)] bg-white/70 p-5">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <p className="text-xs font-semibold uppercase tracking-wide text-muted">{item.category}</p>
                <h2 className="text-lg font-semibold">
                  {item.label}
                  {item.required ? "" : " (optional)"}
                </h2>
                {item.instructions && <p className="mt-1 text-sm text-muted">{item.instructions}</p>}
                <p className="mt-2 text-xs font-semibold">{documentStatusLabel(item.status)}</p>
                {item.review_notes && (item.status === "requires_replacement" || item.status === "rejected") && (
                  <p className="mt-2 rounded-md bg-[color:var(--danger)]/8 px-3 py-2 text-sm">
                    {item.review_notes}
                  </p>
                )}
              </div>
              <label className="cursor-pointer rounded-md bg-leaf px-3 py-2 text-sm font-semibold text-white hover:bg-leaf-deep">
                {busy === item.document_type_code ? "Uploading…" : item.status === "not_started" ? "Upload" : "Replace"}
                <input
                  type="file"
                  accept=".pdf,.png,.jpg,.jpeg,application/pdf,image/png,image/jpeg"
                  className="hidden"
                  disabled={busy === item.document_type_code}
                  onChange={(e) => onUpload(item.document_type_code, e.target.files?.[0] || null)}
                />
              </label>
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}
