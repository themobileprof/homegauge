"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { api, apiBase } from "@/lib/api";

type Item = {
  document_type_code: string;
  label: string;
  category: string;
  required: boolean;
  instructions: string;
  status: string;
  document_id?: string;
};

export default function DocumentsPage() {
  const [items, setItems] = useState<Item[]>([]);
  const [appID, setAppID] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState<string | null>(null);
  const [message, setMessage] = useState("");

  async function load() {
    const data = await api<{ application_id: string; items: Item[] }>("/api/v1/documents/checklist");
    setAppID(data.application_id);
    setItems(data.items || []);
  }

  useEffect(() => {
    load().catch((e) => setError(e.message));
  }, []);

  async function onUpload(code: string, file: File | null) {
    if (!file || !appID) return;
    setBusy(code);
    setError("");
    setMessage("");
    try {
      const fd = new FormData();
      fd.append("application_id", appID);
      fd.append("document_type_code", code);
      fd.append("file", file);
      const res = await fetch(`${apiBase}/api/v1/documents/upload`, {
        method: "POST",
        credentials: "include",
        body: fd,
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "Upload failed");
      setMessage(`${file.name} uploaded.`);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Upload failed");
    } finally {
      setBusy(null);
    }
  }

  const done = items.filter((i) => ["uploaded", "under_review", "accepted"].includes(i.status)).length;

  return (
    <div className="mx-auto min-h-screen max-w-3xl px-5 py-10">
      <div className="flex items-center justify-between gap-4">
        <Link href="/app" className="font-[family-name:var(--font-display)] text-2xl font-semibold">
          Home<span className="text-leaf">Gauge</span>
        </Link>
        <button
          type="button"
          onClick={() =>
            api("/api/v1/applications/request-advisor", { method: "POST" })
              .then(() => setMessage("Advisor requested. They will review your case."))
              .catch((e) => setError(e.message))
          }
          className="rounded-md border border-[color:var(--line)] px-3 py-2 text-sm font-semibold"
        >
          Request advisor
        </button>
      </div>

      <h1 className="mt-8 font-[family-name:var(--font-display)] text-3xl font-semibold">Your documents</h1>
      <p className="mt-2 text-sm text-muted">
        Upload PDF, JPG, or PNG (max 10MB). Files stay private — only short-lived signed links can open them.
      </p>
      <p className="mt-4 text-sm font-semibold text-leaf">{done} of {items.length} started</p>

      {error && <p className="mt-4 text-sm text-[color:var(--danger)]">{error}</p>}
      {message && <p className="mt-4 text-sm text-leaf">{message}</p>}

      <ul className="mt-8 space-y-4">
        {items.map((item) => (
          <li key={item.document_type_code} className="rounded-xl border border-[color:var(--line)] bg-white/70 p-5">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <p className="text-xs font-semibold uppercase tracking-wide text-muted">{item.category}</p>
                <h2 className="text-lg font-semibold">{item.label}{item.required ? "" : " (optional)"}</h2>
                {item.instructions && <p className="mt-1 text-sm text-muted">{item.instructions}</p>}
                <p className="mt-2 text-xs font-semibold">Status: {item.status.replaceAll("_", " ")}</p>
              </div>
              <label className="cursor-pointer rounded-md bg-leaf px-3 py-2 text-sm font-semibold text-white hover:bg-leaf-deep">
                {busy === item.document_type_code ? "Uploading…" : "Upload"}
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
