# Security Policy

This document outlines how to report security vulnerabilities in `actup` and what to expect during our response process.

## Supported Versions

We actively support and patch security issues in the following versions:

| Version | Supported |
| ------- | --------- |
| All versions | :white_check_mark: Yes |

Please upgrade to the latest stable release to ensure you have the latest security patches.

## Reporting a Vulnerability

If you discover a security vulnerability in `actup`, **do not open a public issue.** Public disclosure puts users at risk before a patch is available.

Instead, email your report privately to **[me@lynicis.dev](mailto:me@lynicis.dev)**.

Please include:
- A description of the vulnerability and its potential impact.
- Steps to reproduce the issue, including relevant workflow files.
- Any tools or scripts required to demonstrate the exploit.

## Response Process

When you submit a report, here is how we handle it:

1. **Acknowledge:** We will confirm receipt within 48 hours.
2. **Assess:** We will verify the vulnerability and follow up if we need more details.
3. **Patch:** We aim to develop and test a fix within 14 days.
4. **Publish:** We will release a patched version and document the security advisory.
5. **Credit:** We will credit your discovery in our release notes unless you request anonymity.

## Release Integrity & Package Verification

- **Release archives & checksums:** Every GitHub Release publishes `actup_<version>_checksums.txt` (SHA256) and GitHub build provenance attestations (`actions/attest`) for `.tar.gz` and `.zip` archives. Verify an archive with `gh attestation verify <archive> --repo lynicis/actup`.
- **Container images:** Images published to `ghcr.io/lynicis/actup` are scanned with Trivy before release and signed with Cosign.
- **Chocolatey package:** The `actup` Chocolatey package (`packaging/chocolatey/`) does not embed binaries. Its install script downloads `gh-actup-windows-amd64.exe` over HTTPS from GitHub Releases as `actup.exe` and verifies its SHA256 checksum against `actup_<version>_checksums.txt`.

Thank you for helping protect `actup` users.
