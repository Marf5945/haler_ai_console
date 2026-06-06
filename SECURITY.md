# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 3.5.x   | :white_check_mark: |
| < 3.5   | :x:                |

## Reporting a Vulnerability

If you discover a security vulnerability in AI Console, please report it
through **GitHub Issues**:

1. Go to the [Issues](../../issues) page of this repository.
2. Create a new issue with the title prefix `[SECURITY]`.
3. Describe the vulnerability, including steps to reproduce if possible.
4. Do **not** include exploit code or working PoC in public issues — describe
   the impact and we will coordinate privately if needed.

We aim to acknowledge reports within 72 hours and provide a fix or mitigation
plan within 14 days for confirmed vulnerabilities.

## Scope

The following areas are in scope for security reports:

- LLM Context Governance bypass (§11)
- Source Trust / GatewaySentinel bypass (§9)
- Memory pipeline injection or credential leakage
- Stop Recovery forbidden action bypass (§21)
- DAG Resume Guard hash collision or bypass
- Credential store plaintext exposure
- Project Lifecycle purge deleting protected files

## Out of Scope

- Vulnerabilities in upstream dependencies (report to the respective project)
- Social engineering attacks
- Denial of service on local-only components

## Contact

**GitHub Issues only.** We do not accept vulnerability reports via email.
