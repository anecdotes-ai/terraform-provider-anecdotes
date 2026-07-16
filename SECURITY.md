<!--
Copyright (c) Anecdotes AI
SPDX-License-Identifier: MPL-2.0
-->

# Security Policy

## Supported versions

The latest released `1.x` version of the Anecdotes Terraform Provider receives
security updates. Please upgrade to the most recent release before reporting an issue.

## Reporting a vulnerability

**Please do not open public GitHub issues for security vulnerabilities.**

Report suspected vulnerabilities privately using GitHub's
[private vulnerability reporting](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing/privately-reporting-a-security-vulnerability):
open the repository's **Security** tab and click **Report a vulnerability**.

Please include:

- A description of the vulnerability and its impact.
- Steps to reproduce (proof-of-concept, affected version, configuration).
- Any suggested remediation, if known.

We aim to acknowledge reports within **3 business days** and to provide a remediation
timeline after triage.

## Handling secrets

This provider never bundles credentials. Each user supplies their own API key via the
`ANECDOTES_API_KEY` environment variable or the `api_key` provider attribute (marked
`Sensitive`). If you believe a credential was exposed, **rotate it immediately** in the
Anecdotes UI (**Administration > API Tokens**).
