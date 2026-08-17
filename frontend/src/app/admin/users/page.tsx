"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";
import { api } from "@/lib/api";
import { roleLabel, useAuth, type Role } from "@/lib/auth";

type User = {
  id: string;
  email: string;
  role: Role;
  status: string;
  full_name: string;
  created_at: string;
};

const ROLES: Role[] = ["CUSTOMER", "ADVISOR", "ADMIN", "LENDER_USER"];

const emptyForm = {
  full_name: "",
  email: "",
  password: "",
  role: "CUSTOMER" as Role,
  status: "active",
};

export default function AdminUsersPage() {
  const { user: me } = useAuth();
  const [users, setUsers] = useState<User[]>([]);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [query, setQuery] = useState("");
  const [roleFilter, setRoleFilter] = useState<string>("all");
  const [editingId, setEditingId] = useState<string | "new" | null>(null);
  const [form, setForm] = useState(emptyForm);

  const editing = editingId && editingId !== "new" ? users.find((u) => u.id === editingId) : null;
  const creating = editingId === "new";

  async function load() {
    const d = await api<{ users: User[] }>("/api/v1/admin/users");
    setUsers(d.users || []);
  }

  useEffect(() => {
    load()
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  const visible = useMemo(() => {
    const q = query.trim().toLowerCase();
    return users.filter((u) => {
      if (roleFilter !== "all" && u.role !== roleFilter) return false;
      if (!q) return true;
      return u.email.toLowerCase().includes(q) || (u.full_name || "").toLowerCase().includes(q);
    });
  }, [users, query, roleFilter]);

  function openCreate() {
    setError("");
    setMessage("");
    setForm(emptyForm);
    setEditingId("new");
  }

  function openEdit(u: User) {
    setError("");
    setMessage("");
    setForm({
      full_name: u.full_name || "",
      email: u.email,
      password: "",
      role: u.role,
      status: u.status || "active",
    });
    setEditingId(u.id);
  }

  function closePanel() {
    setEditingId(null);
    setForm(emptyForm);
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setMessage("");
    setBusy(true);
    try {
      if (creating) {
        const d = await api<{ user: User }>("/api/v1/admin/users", {
          method: "POST",
          body: JSON.stringify({
            full_name: form.full_name.trim(),
            email: form.email.trim(),
            password: form.password,
            role: form.role,
          }),
        });
        setUsers((prev) => [d.user, ...prev.filter((u) => u.id !== d.user.id)]);
        setMessage(`${d.user.email} can sign in now.`);
        closePanel();
      } else if (editing) {
        const body: Record<string, string> = {
          full_name: form.full_name.trim(),
          role: form.role,
          status: form.status,
        };
        if (form.password.trim()) body.password = form.password;
        const d = await api<{ user: User }>(`/api/v1/admin/users/${editing.id}`, {
          method: "PATCH",
          body: JSON.stringify(body),
        });
        setUsers((prev) => prev.map((u) => (u.id === d.user.id ? d.user : u)));
        setMessage("Saved.");
        closePanel();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not save user.");
    } finally {
      setBusy(false);
    }
  }

  async function onRemove(u: User) {
    if (u.id === me?.id) return;
    if (!window.confirm(`Remove ${u.email}? They will no longer be able to sign in.`)) return;
    setError("");
    setMessage("");
    setBusy(true);
    try {
      await api(`/api/v1/admin/users/${u.id}`, { method: "DELETE" });
      setUsers((prev) => prev.filter((x) => x.id !== u.id));
      if (editingId === u.id) closePanel();
      setMessage(`${u.email} removed.`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not remove user.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto max-w-6xl px-5 py-10 md:px-8">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.16em] text-gold">Admin console</p>
          <h1 className="mt-2 font-[family-name:var(--font-display)] text-3xl font-semibold">Users</h1>
          <p className="mt-2 max-w-xl text-sm text-muted">
            Create staff and homebuyer accounts, change roles, disable access, or set a new password.
          </p>
        </div>
        <button
          type="button"
          onClick={openCreate}
          className="rounded-md bg-ink px-4 py-2.5 text-sm font-semibold text-paper hover:bg-ink-soft"
        >
          Add user
        </button>
      </div>

      {error && <p className="mt-4 text-sm text-[color:var(--danger)]">{error}</p>}
      {message && <p className="mt-4 text-sm text-leaf">{message}</p>}

      {editingId && (
        <form onSubmit={onSubmit} className="mt-8 rounded-xl border border-[color:var(--line)] bg-white/80 p-6">
          <div className="flex items-start justify-between gap-3">
            <h2 className="text-lg font-semibold">{creating ? "New user" : `Edit ${editing?.email}`}</h2>
            <button type="button" onClick={closePanel} className="text-sm font-semibold text-muted hover:text-ink">
              Close
            </button>
          </div>
          <div className="mt-5 grid gap-4 sm:grid-cols-2">
            <label className="block text-sm">
              <span className="mb-1.5 block font-medium">Full name</span>
              <input
                required
                minLength={2}
                value={form.full_name}
                onChange={(e) => setForm((f) => ({ ...f, full_name: e.target.value }))}
                className="w-full rounded-md border border-[color:var(--line)] bg-white px-3 py-2.5 outline-none ring-leaf focus:ring-2"
              />
            </label>
            <label className="block text-sm">
              <span className="mb-1.5 block font-medium">Email</span>
              <input
                required={creating}
                type="text"
                inputMode="email"
                autoComplete="off"
                disabled={!creating}
                value={form.email}
                onChange={(e) => setForm((f) => ({ ...f, email: e.target.value }))}
                className="w-full rounded-md border border-[color:var(--line)] bg-white px-3 py-2.5 outline-none ring-leaf focus:ring-2 disabled:bg-paper-2"
              />
            </label>
            <label className="block text-sm">
              <span className="mb-1.5 block font-medium">{creating ? "Password" : "New password (optional)"}</span>
              <input
                required={creating}
                minLength={creating ? 8 : undefined}
                type="password"
                autoComplete="new-password"
                value={form.password}
                onChange={(e) => setForm((f) => ({ ...f, password: e.target.value }))}
                className="w-full rounded-md border border-[color:var(--line)] bg-white px-3 py-2.5 outline-none ring-leaf focus:ring-2"
              />
            </label>
            <label className="block text-sm">
              <span className="mb-1.5 block font-medium">Role</span>
              <select
                value={form.role}
                onChange={(e) => setForm((f) => ({ ...f, role: e.target.value as Role }))}
                disabled={editing?.id === me?.id}
                className="w-full rounded-md border border-[color:var(--line)] bg-white px-3 py-2.5 outline-none ring-leaf focus:ring-2 disabled:bg-paper-2"
              >
                {ROLES.map((r) => (
                  <option key={r} value={r}>
                    {roleLabel(r)}
                  </option>
                ))}
              </select>
            </label>
            {!creating && (
              <label className="block text-sm">
                <span className="mb-1.5 block font-medium">Status</span>
                <select
                  value={form.status}
                  onChange={(e) => setForm((f) => ({ ...f, status: e.target.value }))}
                  disabled={editing?.id === me?.id}
                  className="w-full rounded-md border border-[color:var(--line)] bg-white px-3 py-2.5 outline-none ring-leaf focus:ring-2 disabled:bg-paper-2"
                >
                  <option value="active">Active</option>
                  <option value="disabled">Disabled</option>
                </select>
              </label>
            )}
          </div>
          <div className="mt-6 flex flex-wrap items-center gap-3">
            <button
              type="submit"
              disabled={busy}
              className="rounded-md bg-leaf px-4 py-2.5 text-sm font-semibold text-white hover:bg-leaf-deep disabled:opacity-60"
            >
              {busy ? "Saving…" : creating ? "Create user" : "Save changes"}
            </button>
            {!creating && editing && editing.id !== me?.id && (
              <button
                type="button"
                disabled={busy}
                onClick={() => onRemove(editing)}
                className="rounded-md border border-[color:var(--danger)]/30 px-4 py-2.5 text-sm font-semibold text-[color:var(--danger)] hover:bg-red-50 disabled:opacity-60"
              >
                Remove
              </button>
            )}
          </div>
        </form>
      )}

      <div className="mt-8 flex flex-wrap gap-3">
        <input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search name or email"
          className="min-w-[16rem] flex-1 rounded-md border border-[color:var(--line)] bg-white px-3 py-2 text-sm outline-none ring-leaf focus:ring-2"
        />
        <select
          value={roleFilter}
          onChange={(e) => setRoleFilter(e.target.value)}
          className="rounded-md border border-[color:var(--line)] bg-white px-3 py-2 text-sm"
        >
          <option value="all">All roles</option>
          {ROLES.map((r) => (
            <option key={r} value={r}>
              {roleLabel(r)}
            </option>
          ))}
        </select>
      </div>

      <div className="mt-4 overflow-x-auto rounded-xl border border-[color:var(--line)] bg-white/80">
        <table className="w-full min-w-[720px] text-left text-sm">
          <thead className="border-b border-[color:var(--line)] bg-paper-2/60">
            <tr>
              <th className="px-4 py-3 font-semibold">Name</th>
              <th className="px-4 py-3 font-semibold">Email</th>
              <th className="px-4 py-3 font-semibold">Role</th>
              <th className="px-4 py-3 font-semibold">Status</th>
              <th className="px-4 py-3 font-semibold">Added</th>
              <th className="px-4 py-3 font-semibold"> </th>
            </tr>
          </thead>
          <tbody>
            {visible.map((u) => (
              <tr key={u.id} className="border-b border-[color:var(--line)] last:border-0">
                <td className="px-4 py-3">
                  {u.full_name || "—"}
                  {u.id === me?.id && <span className="ml-2 text-xs font-semibold text-muted">you</span>}
                </td>
                <td className="px-4 py-3">{u.email}</td>
                <td className="px-4 py-3 font-medium">{roleLabel(u.role)}</td>
                <td className="px-4 py-3">
                  <span className={u.status === "active" ? "text-leaf" : "text-muted"}>{u.status}</span>
                </td>
                <td className="px-4 py-3 text-muted">
                  {u.created_at ? new Date(u.created_at).toLocaleDateString() : "—"}
                </td>
                <td className="px-4 py-3 text-right">
                  <button type="button" onClick={() => openEdit(u)} className="font-semibold text-leaf hover:underline">
                    Edit
                  </button>
                </td>
              </tr>
            ))}
            {!loading && visible.length === 0 && (
              <tr>
                <td colSpan={6} className="px-4 py-8 text-center text-muted">
                  No users match that filter.
                </td>
              </tr>
            )}
            {loading && (
              <tr>
                <td colSpan={6} className="px-4 py-8 text-center text-muted">
                  Loading…
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
