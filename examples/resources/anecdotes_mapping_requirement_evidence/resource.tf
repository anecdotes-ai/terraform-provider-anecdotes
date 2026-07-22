# Link an existing evidence to a requirement.
# evidence_id references an evidence that already exists in the Anecdotes platform.
resource "anecdotes_mapping_requirement_evidence" "link_s3_acl" {
  requirement_id = anecdotes_requirement.data_protection.requirement_id
  evidence_id    = "evidence_12345"
}
