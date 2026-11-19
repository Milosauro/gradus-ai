# Security Policy

## 1. Supported Versions

During the initial development phase, security updates are applied to the active development trunk (`main`) and the latest pre-release versions.

| Version | Supported |
| --- | --- |
| `0.1.x` (pre-release) | :white_check_mark: |
| `< 0.1.0` | :x: |

## 2. Reporting a Vulnerability

If you discover a security issue or vulnerability in **Gradus**, please do **NOT** open a public issue on GitHub.

Instead, please send a private report to:
- **Email:** `security@gradus.ai` (or via private GitHub Security Advisory on `github.com/Milosauro/gradus-ai/security/advisories/new`)

Please include in your report:
- A description of the vulnerability.
- Steps to reproduce or a minimal proof of concept (PoC).
- Potential impact and affected components (`gradusctl`, skills, compiler, or resolution engine).

We will acknowledge receipt within 48 hours and work with you on coordinated disclosure.

## 3. Security Perimeter & Principles

- **No Secrets in Config:** Gradus configurations (`gradus.yaml`, `level.yaml`, rule files) must NEVER contain cleartext credentials, tokens, or private keys. Existing Git credentials and environment variables are reused.
- **Supply Chain Assurance:** All release binaries are compiled with `CGO_ENABLED=0`, signed using keyless Cosign (Sigstore), accompanied by an automated SBOM, and attested with GitHub Build Provenance.
- **Redaction:** The learning engine (`gradus-learn`) automatically redacts credentials, access tokens, and personal email addresses before writing local lesson logs.
