# Import a SCIM API key by its key_id. The full secret is not recoverable via
# import — only the platform's truncated (last 8 characters) value is available.
terraform import anecdotes_scim_api_key.example 0000000000000000000000000000000000000000000000000000000000000000
