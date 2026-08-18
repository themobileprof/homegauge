CREATE INDEX IF NOT EXISTS idx_mortgage_applications_assigned_advisor
  ON mortgage_applications (assigned_advisor_id)
  WHERE assigned_advisor_id IS NOT NULL;
