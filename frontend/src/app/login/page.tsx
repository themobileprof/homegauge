"use client";

import Link from "next/link";
import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";

const apiBase = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    const res = await fetch(`${apiBase}/api/v1/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ email, password }),
    });
    const data = await res.json();
    if (!res.ok) {
      setError(data.error || "Could not sign in");
      return;
    }
    router.push("/app");
  }

  return (
    <div className="mx-auto flex min-h-screen max-w-md flex-col justify-center px-5 py-12">
      <Link href="/" className="font-[family-name:var(--font-display)] text-2xl font-semibold">
        Home<span className="text-leaf">Gauge</span>
      </Link>
      <h1 className="mt-8 font-[family-name:var(--font-display)] text-3xl font-semibold">Sign in</h1>
      <p className="mt-2 text-sm text-muted">Continue your mortgage journey.</p>
      <form onSubmit={onSubmit} className="mt-8 space-y-4">
        <label className="block text-sm">
          <span className="mb-1.5 block font-medium">Email</span>
          <input required type="email" value={email} onChange={(e) => setEmail(e.target.value)} className="w-full rounded-md border border-[color:var(--line)] bg-white px-3 py-2.5 outline-none ring-leaf focus:ring-2" />
        </label>
        <label className="block text-sm">
          <span className="mb-1.5 block font-medium">Password</span>
          <input required type="password" value={password} onChange={(e) => setPassword(e.target.value)} className="w-full rounded-md border border-[color:var(--line)] bg-white px-3 py-2.5 outline-none ring-leaf focus:ring-2" />
        </label>
        {error && <p className="text-sm text-[color:var(--danger)]">{error}</p>}
        <button type="submit" className="w-full rounded-md bg-leaf py-3 text-sm font-semibold text-white hover:bg-leaf-deep">
          Sign in
        </button>
      </form>
      <p className="mt-6 text-sm text-muted">
        New here? <Link href="/register" className="font-semibold text-leaf">Create an account</Link>
      </p>
    </div>
  );
}
