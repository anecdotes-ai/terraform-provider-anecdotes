resource "anecdotes_requirement" "data_protection" {
  name        = "Data at rest is encrypted"
  description = "Storage volumes and buckets are encrypted with managed keys"
  category    = "Encryption"
}

# Evidence is collected by the Anecdotes platform and is read-only here, so
# evidence_id refers to an evidence that already exists. Find one with the
# anecdotes_evidences data source.
resource "anecdotes_mapping_requirement_evidence" "link_storage_encryption" {
  requirement_id = anecdotes_requirement.data_protection.requirement_id
  evidence_id    = "evidence_12345"
}
