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
  DOCUMENTS_PENDING: "Documents pending",
  DOCUMENTS_UNDER_REVIEW: "Documents under review",
  READY_FOR_SUBMISSION: "Ready for submission",
  SUBMITTED_TO_LENDER: "Submitted to lender",
  LENDER_REVIEW: "Lender review",
  ADDITIONAL_INFORMATION_REQUIRED: "More information needed",
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
