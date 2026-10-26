---
trigger: glob
globs: **/*_test.go, **/*.spec.ts, **/*.stories.ts, test/**/*
---

# Test Data Sanitization and Dummy Value Standards

When creating or modifying tests or test fixtures, you **must** ensure that all test inputs, mock payloads, and raw log strings use standard dummy values. Never commit data copied from live environments or personal development projects without sanitizing all identifiers first.

These rules apply to all structured fields, labels, and raw text payloads, regardless of field names or data format.

## 1. IP Addresses (IPv4 and IPv6)

- **Allowed**:
  - Loopback and unspecified addresses: `127.0.0.1`, `0.0.0.0`, `::1`, `::`
  - RFC 5737 documentation IPv4 ranges: `192.0.2.0/24`, `198.51.100.0/24`, `203.0.113.0/24` (e.g., `192.0.2.1`, `198.51.100.1`)
  - RFC 3849 documentation IPv6 range: `2001:db8::/32` (e.g., `2001:db8::1`)
  - Simple textbook private IPs or standard Kubernetes CIDR addresses (e.g., `10.0.0.x`, `192.168.0.x`, `192.168.1.x`, `10.96.0.x`, `10.244.0.x`)
- **Prohibited**:
  - Any routable public IP address outside RFC 5737 / RFC 3849 ranges, including arbitrary digit sequences like `123.45.167.189` or real cloud provider IPs.

## 2. Project, Infrastructure, and Resource Identifiers

- **Scope**: All project IDs/numbers and resource names across cloud infrastructure, clusters, nodes, workloads, networking, and applications.
- **Allowed**:
  - Generic placeholders and standard Kubernetes namespaces (e.g., `test-*`, `sample-*`, `my-*`, `foo`, `bar`, `*-1`, `default`, `kube-system`, `123456789`, `test-project`, `test-cluster`, `gke-test-cluster-default-pool-12345678-abcd`)
  - Standard open-source or sample application names (e.g., `nginx`, `busybox`, `redis`, `productpage`, `details`, `ratings`, `reviews`, `bookinfo`, `sleep`, `httpbin`)
- **Prohibited**:
  - Real organization names, personal names/usernames, team prefixes, or actual project/cluster/resource names.

## 3. Domains, Hostnames, URLs, and Request Paths

- **Allowed**:
  - Public base domains: RFC 2606 reserved domains (`example.com`, `example.org`, `example.net`, `localhost`) or public Google API endpoints (`*.googleapis.com`).
  - Subdomains and Kubernetes internal DNS (`<service>.<namespace>.svc.cluster.local`): Every subdomain, `<service>`, and `<namespace>` segment must use the allowed resource placeholders above (e.g., `foo.default.svc.cluster.local`, `app.example.com`).
  - URL paths and query parameters: Generic paths and parameters (e.g., `/`, `/healthz`, `/readyz`, `/metrics`, `/api/v1/items`, `/productpage`).
- **Prohibited**:
  - Real-world organization or personal domains.
  - Internal hostnames (such as `<real-name>.svc.cluster.local`), subdomains, URL paths, or query parameters that contain real service names, organization names, personal usernames, or project identifiers in any segment.

## 4. User, Account, and Principal Identifiers

- **Allowed**:
  - Generic dummy usernames, emails, and service accounts (e.g., `test-user`, `alice`, `bob`, `test-user@example.com`, `test-sa@test-project.iam.gserviceaccount.com`) or standard Kubernetes system principals (e.g., `system:serviceaccount:default:default`, `system:node:test-node`).
  - Both the local part (before `@`) and the domain/project part (after `@`) must use allowed dummy values.
- **Prohibited**:
  - Real personal or organizational email addresses, usernames, or service accounts tied to actual projects or real names.
