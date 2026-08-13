"use client";

import Link from "next/link";
import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";

const apiBase = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export default function RegisterPage() {
  const router = useRouter();
  const [fullName, setFullName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setMessage("");
    const res = await fetch(`${apiBase}/api/v1/auth/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ full_name: fullName, email, password }),
    });
    const data = await res.json();
    if (!res.ok) {
      setError(data.error || "Could not create account");
      return;
    }
    setMessage(data.message || "Account created. Check your email to verify.");
    setTimeout(() => router.push("/login"), 1200);
  }

  return (
    <AuthShell title="Create your HomeGauge account" subtitle="Start with eligibility built around your salary account.">
      <form onSubmit={onSubmit} className="space-y-4">
        <Input label="Full name" value={fullName} onChange={setFullName} />
        <Input label="Email" type="email" value={email} onChange={setEmail} />
        <Input label="Password (min 8 characters)" type="password" value={password} onChange={setPassword} />
        {error && <p className="text-sm text-[color:var(--danger)]">{error}</p>}
        {message && <p className="text-sm text-leaf">{message}</p>}
        <button type="submit" className="w-full rounded-md bg-leaf py-3 text-sm font-semibold text-white hover:bg-leaf-deep">
          Create account
        </button>
      </form>
      <p className="mt-6 text-sm text-muted">
        Already have an account? <Link href="/login" className="font-semibold text-leaf">Sign in</Link>
      </p>
    </AuthShell>
  );
}

function AuthShell({ title, subtitle, children }: { title: string; subtitle: string; children: React.ReactNode }) {
  return (
    <div className="mx-auto flex min-h-screen max-w-md flex-col justify-center px-5 py-12">
      <Link href="/" className="font-[family-name:var(--font-display)] text-2xl font-semibold">
        Home<span className="text-leaf">Gauge</span>
      </Link>
      <h1 className="mt-8 font-[family-name:var(--font-display)] text-3xl font-semibold">{title}</h1>
      <p className="mt-2 text-sm text-muted">{subtitle}</p>
      <div className="mt-8">{children}</div>
    </div>
  );
}

function Input({
  label,
  value,
  onChange,
  type = "text",
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  type?: string;
}) {
  return (
    <label className="block text-sm">
      <span className="mb-1.5 block font-medium">{label}</span>
      <input
        required
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full rounded-md border border-[color:var(--line)] bg-white px-3 py-2.5 outline-none ring-leaf focus:ring-2"
      />
    </label>
  );
}
