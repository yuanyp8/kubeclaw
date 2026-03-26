# Documentation Index

Current documents:

- `docs/backend-design.md`
- `docs/backend-development.md`
- `docs/frontend-development.md`
- `docs/frontend-deployment.md`
- `docs/api-use-cases.md`
- `docs/cluster-kubeconfig-servername.md`
- `docs/agent-architecture.md`
- `docs/agent-api.md`
- `docs/platform-logs.md`

Recent capability additions reflected in the docs:

- rebuilt the workspace shell toward a KubeSphere-like top header plus left navigation layout
- new `/dashboard` route for cluster overview
- split cluster runtime workspace and cluster settings into separate pages
- new cluster workspace with integrated resource browsing and actions
- pod log streaming and keyword highlighting
- cluster overview API for namespace, node, pod, deployment, and service health
- direct cluster action APIs for delete, scale, restart, and apply
- model creation tests before save, while model edit avoids forced retest hangs
- platform log viewing and client log upload
- teams page now refreshes immediately after removing members and user pages fail more gracefully

Documentation maintenance rules:

- update `docs/backend-development.md` whenever backend flow or module boundaries change
- update `docs/frontend-development.md` whenever route structure, layout, or state flow changes
- update `docs/api-use-cases.md` whenever request or response contracts change
- add focused topic documents for new subsystems instead of growing one giant overview file
