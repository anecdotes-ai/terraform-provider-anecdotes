terraform {
  required_providers {
    anecdotes = {
      source  = "anecdotes-ai/anecdotes"
      version = "1.0.0"
    }
  }
}

provider "anecdotes" {
  # Authenticates with the ANECDOTES_API_KEY environment variable.
  # Optionally set ANECDOTES_API_URL to override the default API base URL.
}
