terraform {
  required_providers {
    anecdotes = {
      source  = "anecdotes-ai/anecdotes"
      version = "1.3.0"
    }
  }
}

provider "anecdotes" {
  # Uses ANECDOTES_API_KEY environment variable
}
