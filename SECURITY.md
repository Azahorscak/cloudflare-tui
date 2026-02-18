# Security

## Security Model

cloudflare-tui lists Cloudflare DNS zones and records and allows **editing** existing DNS records. It does not create or delete resources, but it does issue PUT requests to the Cloudflare API to update record values.

Credentials are loaded exclusively from a Kubernetes secret at startup. The API token is held in memory for the lifetime of the process and is never written to disk, logged, or transmitted to any destination other than the Cloudflare API.

## Cloudflare API Token Scoping

Follow the principle of least privilege when creating the API token stored in your Kubernetes secret. The application requires:

| Permission   | Access Level |
|---|---|
| Zone / Zone  | Read         |
| Zone / DNS   | Edit         |

`Zone / DNS Edit` is required because the application can update existing DNS records. If you only need read-only inspection and do not require the edit feature, scope the token to `Zone / DNS Read` instead and the edit form will return an API error when a save is attempted.

To create a properly scoped token:

1. Go to the [Cloudflare API Tokens](https://dash.cloudflare.com/profile/api-tokens) page.
2. Click **Create Token**.
3. Use the **Custom token** template.
4. Add only the two permissions above.
5. Under **Zone Resources**, restrict to the specific zones the operator needs to manage (or "All zones" if that's appropriate).
6. Set a reasonable TTL and enable token rotation if your workflow supports it.

**Do not** use a Global API Key. It grants full account access and cannot be scoped.

## Kubernetes RBAC

The application (or the user/service account running it) needs only `get` access to the single Secret containing the API token. A minimal RBAC policy:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: cloudflare-tui-reader
  namespace: <namespace>
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    resourceNames: ["<secret-name>"]
    verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: cloudflare-tui-reader-binding
  namespace: <namespace>
subjects:
  - kind: ServiceAccount
    name: <service-account>
    namespace: <namespace>
roleRef:
  kind: Role
  name: cloudflare-tui-reader
  apiGroup: rbac.authorization.k8s.io
```

Replace `<namespace>`, `<secret-name>`, and `<service-account>` with your values. The `resourceNames` field ensures the role can only read the specific secret it needs.

## Reporting a Vulnerability

If you discover a security issue, please report it privately by opening a [GitHub Security Advisory](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing-information-about-vulnerabilities/privately-reporting-a-security-vulnerability) on this repository. Do not open a public issue for security vulnerabilities.
