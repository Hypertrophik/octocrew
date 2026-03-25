# 🐙 Octocrew: GitOps Onboarding Generator

An interactive CLI tool that generates Crossplane-compatible YAML manifests for onboarding users to a GitHub organization via GitOps.

## What it does

Given one or more users (GitHub username + Okta ID) and a target GitHub org, it creates a directory named after the org and writes one YAML file per user containing:

- A **`Membership`** resource (`user.github.upbound.io/v1alpha1`)
- One **`TeamMembership`** resource (`team.github.upbound.io/v1alpha1`) per team (optional)

## Usage

```bash
go run main.go
```

Or build first:

```bash
go build -o octocrew .
./octocrew
```

The tool walks you through a short interactive form:

1. **Target Organization** — the GitHub org slug (e.g. `sinister-six`)
2. **Users** — GitHub username and Okta ID for each person; repeat as many times as needed
3. **Teams** (optional) — comma-separated team slugs to add all users to

Files are written to `./<org>/<username>.yaml`.

## Example

For user **Doctor Octopus** (`doc-ock`) onboarded to org `sinister-six` with team `villains`:

```
sinister-six/
└── doc-ock.yaml
```

**`sinister-six/doc-ock.yaml`**:

```yaml
apiVersion: user.github.upbound.io/v1alpha1
kind: Membership
metadata:
  labels:
    suse.okta.com/user-id: '<okta-id>'
  name: sinister-six--doc-ock
spec:
  deletionPolicy: Delete
  forProvider:
    downgradeOnDestroy: false
    role: member
    username: doc-ock
  providerConfigRef:
    name: sinister-six
---
apiVersion: team.github.upbound.io/v1alpha1
kind: TeamMembership
metadata:
  name: sinister-six--doc-ock--villains
spec:
  forProvider:
    org: sinister-six
    teamSlug: villains
    username: doc-ock
    role: member
  providerConfigRef:
    name: sinister-six
```

## Resource naming

Membership `metadata.name` always follows the pattern `<org>--<username>`.

Team membership names follow the pattern `<org>--<username>--<team-slug>`.

## Requirements

- Go 1.21+

```bash
go mod download
```
