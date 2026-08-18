export const ADVISOR_STATUSES = [
  "DOCUMENTS_PENDING",
  "DOCUMENTS_UNDER_REVIEW",
  "READY_FOR_SUBMISSION",
  "SUBMITTED_TO_LENDER",
  "LENDER_REVIEW",
  "ADDITIONAL_INFORMATION_REQUIRED",
] as const;

export const ADMIN_STATUSES = [
  "NEW",
  "ASSESSMENT_COMPLETED",
  ...ADVISOR_STATUSES,
  "APPROVED",
  "REJECTED",
  "COMPLETED",
  "CANCELLED",
] as const;

const labels: Record<string, string> = {
  NEW: "New",
  ASSESSMENT_COMPLETED: "Assessment completed",
  DOCUMENTS_PENDING: "Waiting on documents",
  DOCUMENTS_UNDER_REVIEW: "Documents to review",
  READY_FOR_SUBMISSION: "Ready for admin",
  SUBMITTED_TO_LENDER: "With lender",
  LENDER_REVIEW: "Lender reviewing",
  ADDITIONAL_INFORMATION_REQUIRED: "Waiting on buyer",
  APPROVED: "Approved (top-level)",
  REJECTED: "Rejected",
  COMPLETED: "Completed",
  CANCELLED: "Cancelled",
};

export function statusLabel(status?: string | null) {
  if (!status) return "—";
  return labels[status] || status.replaceAll("_", " ").toLowerCase();
}

export function advisorName(c: {
  assigned_advisor_name?: string | null;
  assigned_advisor_email?: string | null;
}) {
  return c.assigned_advisor_name || c.assigned_advisor_email || "Unassigned";
}

export type QueueBucket = "review" | "waiting" | "ready" | "other";

export function queueBucket(status: string): QueueBucket {
  if (status === "DOCUMENTS_UNDER_REVIEW") return "review";
  if (status === "READY_FOR_SUBMISSION") return "ready";
  if (
    status === "DOCUMENTS_PENDING" ||
    status === "ADDITIONAL_INFORMATION_REQUIRED" ||
    status === "NEW" ||
    status === "ASSESSMENT_COMPLETED"
  ) {
    return "waiting";
  }
  return "other";
}

export function documentStatusLabel(status?: string | null) {
  switch (status) {
    case "uploaded":
      return "Needs review";
    case "under_review":
      return "In review";
    case "accepted":
      return "Accepted";
    case "rejected":
      return "Sent back";
    case "requires_replacement":
      return "Replacement needed";
    case "not_started":
      return "Not uploaded";
    default:
      return status || "—";
  }
}

export function fileRef(id?: string | null) {
  if (!id) return "HG-————";
  return `HG-${id.replace(/-/g, "").slice(-6).toUpperCase()}`;
}

export function buyerStatusLabel(status?: string | null) {
  switch (status) {
    case "NEW":
    case "ASSESSMENT_COMPLETED":
      return "Preparing your file";
    case "DOCUMENTS_PENDING":
      return "Documents needed";
    case "DOCUMENTS_UNDER_REVIEW":
      return "Your advisor is reviewing documents";
    case "READY_FOR_SUBMISSION":
      return "Your file is being prepared for a lender";
    case "SUBMITTED_TO_LENDER":
      return "Your file is with the lender";
    case "LENDER_REVIEW":
      return "The lender is reviewing your file";
    case "ADDITIONAL_INFORMATION_REQUIRED":
      return "More information needed";
    case "APPROVED":
      return "A positive outcome was recorded on your file";
    case "REJECTED":
      return "This file was not taken forward";
    case "COMPLETED":
      return "This file is closed";
    case "CANCELLED":
      return "This file was cancelled";
    default:
      return statusLabel(status);
  }
}

export function relativeTime(iso?: string | null) {
  if (!iso) return "";
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return "";
  const s = Math.round((Date.now() - t) / 1000);
  if (s < 45) return "just now";
  if (s < 3600) return `${Math.max(1, Math.round(s / 60))}m ago`;
  if (s < 86400) return `${Math.round(s / 3600)}h ago`;
  if (s < 86400 * 7) return `${Math.round(s / 86400)}d ago`;
  return new Date(iso).toLocaleDateString("en-NG", { day: "numeric", month: "short" });
}

export const QUEUE_BUCKETS: { key: QueueBucket; title: string; hint: string }[] = [
  { key: "review", title: "Your review", hint: "Documents or checks waiting on you." },
  { key: "waiting", title: "Waiting on the buyer", hint: "Uploads, replacements, or extra information." },
  { key: "ready", title: "Ready for admin", hint: "File is complete — you have handed it up." },
  { key: "other", title: "With lender", hint: "Submitted or in lender review." },
];
