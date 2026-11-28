# 4. Access & Extensions (part of Cluster Architecture, 25%)

Official competencies: manage RBAC · use Helm and Kustomize to install cluster components ·
understand extension interfaces (CNI, CSI, CRI, etc.) · understand CRDs, install and configure operators.

---

## 4.1 RBAC and ServiceAccounts [`rbac`]

### Key ideas
- **Who**: users and groups (not API objects; they come from the client certificate CN = user, O = group, or from a token/OIDC) and **ServiceAccounts** (namespaced API objects, used by Pods).
- **What**: **Role** (namespaced) or **ClusterRole** (cluster-wide, or reusable) = a list of rules: `apiGroups`, `resources` (with subresources such as `pods/log` and `pods/exec`), `verbs` (get, list, watch, create, update, patch, delete, deletecollection), optional `resourceNames`.
- **Binding**: a **RoleBinding** grants a Role *or ClusterRole* inside **one namespace**. A **ClusterRoleBinding** grants a ClusterRole in **all namespaces** and for cluster-scoped resources.
- RBAC is **additive only**. There are no deny rules.
- Built-in ClusterRoles: `cluster-admin`, `admin`, `edit`, `view`. Bind them with a RoleBinding to limit them to a namespace.
- API groups: `""` = core (pods, services, secrets, configmaps, nodes, PVs…), `apps`, `batch`, `networking.k8s.io`, `rbac.authorization.k8s.io`, `storage.k8s.io`… Find with `k api-resources`.

| Want | Use |
|---|---|
| Access to Pods in namespace `dev` only | Role + RoleBinding in `dev` |
| The same read-only permissions in many namespaces | one ClusterRole + a RoleBinding per namespace |
| Access to nodes or PVs (cluster-scoped) | ClusterRole + ClusterRoleBinding |

### Commands

```bash
k create sa deployer -n dev
k create role deployer --verb=get,list,watch,create,update,patch,delete --resource=deployments.apps,pods -n dev
k create rolebinding deployer --role=deployer --serviceaccount=dev:deployer -n dev
k create clusterrole pv-reader --verb=get,list --resource=persistentvolumes
k create clusterrolebinding pv-reader --clusterrole=pv-reader --user=jane --group=ops
k create rolebinding view-dev --clusterrole=view --user=jane -n dev

# verify (this is how you check your work, and also how exam tasks get graded)
k auth can-i create deployments -n dev --as=system:serviceaccount:dev:deployer      # yes
k auth can-i delete nodes --as=jane                                                  # no
k auth can-i --list -n dev --as=system:serviceaccount:dev:deployer
k auth whoami
```

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata: {name: log-reader, namespace: dev}
rules:
  - apiGroups: [""]
    resources: [pods, pods/log]
    verbs: [get, list]
  - apiGroups: [""]
    resources: [configmaps]
    resourceNames: [app-cfg]          # only this object
    verbs: [get, update]
```

**ServiceAccounts in Pods**
- Every namespace has a `default` SA. A Pod uses `spec.serviceAccountName` (default `default`).
- The token is projected (short-lived, auto-rotated) at `/var/run/secrets/kubernetes.io/serviceaccount/token`. Disable with `automountServiceAccountToken: false`.
- `k create token <sa>` issues a short-lived token for testing.

**Create a user certificate with the CSR API** (a classic task):

```bash
openssl genrsa -out jane.key 2048
openssl req -new -key jane.key -out jane.csr -subj "/CN=jane/O=dev-team"
cat <<EOF | k apply -f -
apiVersion: certificates.k8s.io/v1
kind: CertificateSigningRequest
metadata: {name: jane}
spec:
  request: $(base64 -w0 < jane.csr)
  signerName: kubernetes.io/kube-apiserver-client
  expirationSeconds: 86400
  usages: [client auth]
EOF
k get csr
k certificate approve jane
k get csr jane -o jsonpath='{.status.certificate}' | base64 -d > jane.crt
k config set-credentials jane --client-certificate=jane.crt --client-key=jane.key --embed-certs
k config set-context jane --cluster=$(k config view --minify -o jsonpath='{.clusters[0].name}') --user=jane
k --context jane get pods        # Forbidden until you bind a role
```

### Lab
1. Create SA `ci` in `dev` that can only `get,list` Pods and read their logs in `dev`. Verify with `auth can-i` for `get pods`, `get pods/log`, `delete pods` and `get pods -n default`.
2. Run a Pod with `serviceAccountName: ci` and image `bitnami/kubectl` (or `registry.k8s.io/kubectl`), and run `kubectl get pods` from inside it. Then try `kubectl get secrets`.
3. Create user `jane` via the CSR flow. Bind the built-in `view` ClusterRole to her in `dev` only. Check that she can list Pods in `dev` but not in `default`.
4. Make a ClusterRole `node-viewer` and bind it cluster-wide to group `ops`. Test with `--as=someone --as-group=ops`.

### Pitfalls
- A RoleBinding's `subjects[].namespace` is required for ServiceAccounts. The imperative `--serviceaccount=ns:name` handles it.
- Forgetting the API group (`deployments` lives in `apps`). Imperative `--resource=deployments` resolves it for you; in YAML you must write `apiGroups: [apps]`.
- `pods/exec` needs the `create` verb (and `get` on pods).

### Check questions

**Q1.** A ClusterRole `view` is bound with a **RoleBinding** in namespace `dev` to user `bob`. What can bob do?
- A. View resources in all namespaces
- B. View resources in `dev` only
- C. Nothing; a RoleBinding cannot reference a ClusterRole
- D. View cluster-scoped resources like nodes

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q2.** Which subject string does `kubectl auth can-i --as` use for SA `build` in namespace `ci`?
- A. `build`
- B. `ci:build`
- C. `system:serviceaccount:ci:build`
- D. `serviceaccount/ci/build`

<details><summary>▶ Answer</summary>

**C.**
</details>

**Q3.** A user must be able to read logs of Pods. Which rule is needed in addition to `get pods`?
- A. `get` on `pods/log`
- B. `watch` on `events`
- C. `get` on `logs`
- D. `exec` on `pods`

<details><summary>▶ Answer</summary>

**A.**
</details>

**Q4.** You need to stop one user from deleting Secrets while they keep `edit` elsewhere. How?
- A. Add a deny rule
- B. Remove the broad binding and grant a narrower Role; RBAC has no deny
- C. Use `resourceNames: ["!secrets"]`
- D. Use a NetworkPolicy

<details><summary>▶ Answer</summary>

**B.**
</details>

---

## 4.2 Helm [`helm`]

### Key ideas
- A **chart** is a package of templated manifests plus `values.yaml`. A **release** is an installed instance of a chart in a namespace. Each install, upgrade or rollback creates a new **revision** (stored as Secrets in the release namespace).
- The CKA expects you to **use** charts (add repo, install, set values, upgrade, roll back, inspect), not write them.
- Helm 4 was released in late 2025. The everyday commands below are the same in Helm 3 and 4.

### Commands

```bash
helm repo add podinfo https://stefanprodan.github.io/podinfo
helm repo update
helm search repo podinfo                       # add --versions to see all versions
helm show values podinfo/podinfo | less        # every configurable value
helm show chart podinfo/podinfo

helm install web podinfo/podinfo -n demo --create-namespace \
  --set replicaCount=2 --set service.type=NodePort --version 6.7.1
helm install web2 podinfo/podinfo -n demo -f my-values.yaml
helm list -n demo ; helm list -A
helm status web -n demo
helm get values web -n demo                    # only user-supplied values (add --all for everything)
helm get manifest web -n demo                  # rendered YAML that was applied

helm upgrade web podinfo/podinfo -n demo --reuse-values --set replicaCount=3
helm upgrade --install web podinfo/podinfo -n demo -f my-values.yaml   # idempotent
helm history web -n demo
helm rollback web 1 -n demo
helm uninstall web -n demo

helm template web podinfo/podinfo --set replicaCount=2 > rendered.yaml  # render only, no install
helm pull podinfo/podinfo --untar               # inspect the chart files locally
```

### Lab
1. Install podinfo as release `web` in `demo` with 2 replicas. Confirm with `k get deploy -n demo`.
2. Upgrade to 3 replicas with `--reuse-values`, then `helm history`, then roll back to revision 1. Confirm 2 replicas.
3. Write `values.yaml` with `ui.message: "hello CKA"` and `service.type: NodePort`, and upgrade with `-f`. Check with `helm get values`.
4. Install **metrics-server** from its official chart (repo `https://kubernetes-sigs.github.io/metrics-server/`) into kube-system with `--set args={--kubelet-insecure-tls}`, a typical "install a cluster component with Helm" task.
5. `helm template` a chart and find the Deployment it would create.

### Pitfalls
- Forgetting `-n <ns>` on `helm list/upgrade/rollback`: Helm looks in the current namespace and reports "release not found".
- `helm upgrade` **without** `--reuse-values` or `-f` resets the values you set earlier to the chart defaults.
- A task that says "install version X" needs `--version X`.

### Check questions

**Q1.** Which command shows only the values you supplied to release `api` in namespace `prod`?
- A. `helm show values api`
- B. `helm get values api -n prod`
- C. `helm status api`
- D. `helm template api`

<details><summary>▶ Answer</summary>

**B.** `show values` shows the chart's defaults.
</details>

**Q2.** Release `api` is at revision 4 and broken. Return to revision 2:
- A. `helm upgrade api --revision 2`
- B. `helm rollback api 2 -n prod`
- C. `helm undo api`
- D. `kubectl rollout undo`

<details><summary>▶ Answer</summary>

**B.** This creates revision 5 with revision 2's content.
</details>

**Q3.** You ran `helm upgrade api repo/chart -n prod --set image.tag=2.0`, and the replica count you had set at install time reverted. Why?
- A. Bug in Helm
- B. Without `--reuse-values` (or `-f` with the full values), unspecified values return to chart defaults
- C. The HPA changed it
- D. `--set` always resets replicas

<details><summary>▶ Answer</summary>

**B.**
</details>

---

## 4.3 Kustomize [`kustomize`]

### Key ideas
- Kustomize is template-free: a **base** of plain YAML plus **overlays** that patch it. It is built into kubectl (`kubectl kustomize`, `kubectl apply -k`).
- `kustomization.yaml` fields: `resources`, `namespace`, `namePrefix`/`nameSuffix`, `labels` (newer; `commonLabels` is deprecated), `commonAnnotations`, `images` (change image name/tag), `replicas`, `patches` (strategic merge or JSON6902, with an optional `target`), `configMapGenerator`/`secretGenerator` (add a content hash suffix, so a config change rolls the Pods).

```
app/
├── base/
│   ├── deployment.yaml
│   ├── service.yaml
│   └── kustomization.yaml
└── overlays/
    └── prod/
        ├── kustomization.yaml
        └── replicas-patch.yaml
```

```yaml
# base/kustomization.yaml
resources: [deployment.yaml, service.yaml]
labels:
  - pairs: {app: web}
    includeSelectors: true
---
# overlays/prod/kustomization.yaml
resources: [../../base]
namespace: prod
namePrefix: prod-
images:
  - {name: nginx, newTag: "1.27"}
patches:
  - path: replicas-patch.yaml
  - target: {kind: Deployment, name: web}
    patch: |-
      - op: add
        path: /spec/template/spec/containers/0/env
        value: [{name: ENV, value: prod}]
configMapGenerator:
  - name: web-cfg
    literals: [MODE=prod]
---
# overlays/prod/replicas-patch.yaml   (strategic merge: only the fields you change, plus identity)
apiVersion: apps/v1
kind: Deployment
metadata: {name: web}
spec: {replicas: 4}
```

```bash
kubectl kustomize overlays/prod            # preview the rendered output
kubectl apply -k overlays/prod
kubectl diff -k overlays/prod
kubectl delete -k overlays/prod
```

### Lab
1. Build the tree above (generate `deployment.yaml` with `k create deploy web --image=nginx $do`). Render the prod overlay and check the name `prod-web`, namespace `prod`, image tag 1.27, 4 replicas and the env var.
2. Apply it (create the `prod` namespace first, or add a Namespace resource to the overlay).
3. Change a literal in the configMapGenerator, re-apply, and watch the ConfigMap name hash change and the Deployment roll.
4. Add a `dev` overlay with 1 replica and `namePrefix: dev-`.

### Pitfalls
- `kubectl apply -f dir/` ignores kustomization.yaml. Use `-k`.
- A strategic-merge patch must keep `apiVersion`, `kind` and `metadata.name` so Kustomize can find the target.

### Check questions

**Q1.** Which command applies a Kustomize overlay in `./overlays/dev`?
- A. `kubectl apply -f overlays/dev`
- B. `kubectl apply -k overlays/dev`
- C. `kustomize install overlays/dev`
- D. `helm install -k overlays/dev`

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q2.** Why does a ConfigMap from `configMapGenerator` get a name like `web-cfg-7g8k2m`?
- A. Random collision avoidance
- B. A content hash, so a content change creates a new name and triggers a rollout of the Pods that reference it
- C. To mark it immutable
- D. It is the namespace hash

<details><summary>▶ Answer</summary>

**B.**
</details>

---

## 4.4 Extension interfaces: CRI, CNI, CSI [`extension-interfaces`]

### Key ideas
Kubernetes defines interfaces and lets plugins implement them:

| Interface | Who calls it | Implementations | Where to look on a node |
|---|---|---|---|
| **CRI** (Container Runtime Interface, gRPC) | kubelet | containerd, CRI-O | socket `unix:///run/containerd/containerd.sock`; kubelet flag/config `containerRuntimeEndpoint`; `systemctl status containerd`; `crictl` |
| **CNI** (Container Network Interface) | the runtime, when a Pod sandbox is created | Calico, Cilium, Flannel, Weave… | config in `/etc/cni/net.d/`, binaries in `/opt/cni/bin/`, plus the plugin's DaemonSet in kube-system or its own namespace |
| **CSI** (Container Storage Interface, gRPC) | kubelet + external sidecars (provisioner, attacher, resizer) | cloud disks, Ceph, NFS, local-path… | `k get csidrivers`, `k get csinodes`, the driver's controller Deployment + node DaemonSet |

Other extension points to recognise: **CRDs** and the **aggregation layer** (APIService), **admission webhooks**, **device plugins** (GPUs), **scheduler plugins**.

### crictl: kubectl for the container runtime (on the node)

```bash
sudo crictl ps                 # running containers  (-a includes exited)
sudo crictl pods               # Pod sandboxes
sudo crictl logs <container-id>
sudo crictl inspect <container-id> | less
sudo crictl images ; sudo crictl pull nginx:alpine ; sudo crictl rmi <img>
sudo crictl info               # runtime config, including the CNI status
```

### Commands

```bash
ls /etc/cni/net.d/ ; cat /etc/cni/net.d/*.conflist      # which CNI is configured?
ls /opt/cni/bin/
k get pods -A -o wide | grep -Ei 'calico|cilium|flannel|weave'
k get csidrivers ; k get csinodes
sudo cat /var/lib/kubelet/config.yaml | grep -i containerRuntimeEndpoint
```

### Lab
1. On a node, use `crictl ps` to find the kube-proxy container and read its logs with `crictl logs`.
2. Identify the CNI on your kind and kubeadm clusters from `/etc/cni/net.d`.
3. On lab B, temporarily move the `/etc/cni/net.d/*` files away on a worker and restart kubelet. Create a Pod on that node and read the event (`network plugin not ready` / `NetworkPluginNotReady`). Restore.
4. On a node, `sudo systemctl stop containerd`. Watch the node turn `NotReady` (after ~40 s) and read `journalctl -u kubelet`. Start it again.
5. Run `k get csidrivers` on kind (none) and read the local-path provisioner Deployment in `local-path-storage` to see how dynamic provisioning works without CSI.

### Check questions

**Q1.** Which interface does kubelet use to start containers?
- A. CNI
- B. CSI
- C. CRI
- D. OCI directly

<details><summary>▶ Answer</summary>

**C.**
</details>

**Q2.** New Pods on one node are stuck in `ContainerCreating` with "failed to set up sandbox: plugin type=calico failed". Which layer do you investigate?
- A. CSI
- B. CNI (the plugin config and binaries, plus the plugin Pod on that node)
- C. CRI socket path
- D. RBAC

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q3.** The apiserver is down, so kubectl does not work. Which tool lists containers on the control-plane node?
- A. `docker ps`
- B. `crictl ps`
- C. `kubectl get pods --local`
- D. `ctr k8s list`

<details><summary>▶ Answer</summary>

**B.** Docker is not installed on modern nodes. `ctr -n k8s.io containers list` also works but is less friendly.
</details>

---

## 4.5 CRDs and operators [`crds-operators`]

### Key ideas
- A **CustomResourceDefinition** adds a new resource type (`kind`) to the API. Instances are **custom resources** (CRs). They are stored in etcd and work with kubectl like built-ins.
- An **operator** = CRDs + a controller (usually a Deployment) that watches those CRs and acts on them (creates Deployments, certificates, databases…). Examples: cert-manager, Prometheus Operator, CloudNativePG.
- Installing an operator usually means applying its CRDs + RBAC + controller Deployment, through **Helm** or a manifest URL.
- CRD key fields: `group`, `names` (kind, plural, singular, shortNames), `scope` (Namespaced/Cluster), `versions[]` (each with `served`, `storage`, and an `openAPIV3Schema`).

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: backups.ops.example.com          # <plural>.<group>
spec:
  group: ops.example.com
  scope: Namespaced
  names: {kind: Backup, plural: backups, singular: backup, shortNames: [bk]}
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              required: [schedule]
              properties:
                schedule: {type: string}
                retention: {type: integer, minimum: 1}
      additionalPrinterColumns:
        - {name: Schedule, type: string, jsonPath: .spec.schedule}
---
apiVersion: ops.example.com/v1
kind: Backup
metadata: {name: nightly}
spec: {schedule: "0 2 * * *", retention: 7}
```

### Commands

```bash
k get crd
k get crd backups.ops.example.com -o yaml | less
k api-resources --api-group=ops.example.com
k explain backup.spec                 # works because the CRD has a schema
k get bk ; k describe backup nightly

# install a real operator: cert-manager
helm repo add jetstack https://charts.jetstack.io && helm repo update
helm install cert-manager jetstack/cert-manager -n cert-manager --create-namespace --set crds.enabled=true
k get crd | grep cert-manager
k get pods -n cert-manager
```

Then use the operator through its CRs:

```yaml
apiVersion: cert-manager.io/v1
kind: Issuer
metadata: {name: selfsigned, namespace: default}
spec: {selfSigned: {}}
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata: {name: demo-cert, namespace: default}
spec:
  secretName: demo-tls
  dnsNames: [demo.example.com]
  issuerRef: {name: selfsigned, kind: Issuer}
```

```bash
k get certificate demo-cert            # READY True
k get secret demo-tls                  # created by the operator
k logs -n cert-manager deploy/cert-manager
```

### Lab
1. Apply the `Backup` CRD and CR. Try a CR without `schedule` and one with `retention: 0`, and read the validation errors.
2. Use `k get bk` (short name) and confirm the printer column shows.
3. Install cert-manager with Helm, create the Issuer and Certificate, and confirm the Secret appears.
4. Delete the CRD and observe that all `Backup` objects are deleted with it.

### Pitfalls
- Creating a CR before its CRD is "Established": `no matches for kind`. Wait or check `k get crd <name>`.
- Deleting a CRD deletes **all** its custom resources.

### Check questions

**Q1.** What is the correct `metadata.name` for a CRD with group `ops.example.com` and plural `backups`?
- A. `Backup`
- B. `backups.ops.example.com`
- C. `ops.example.com/backups`
- D. `backup-crd`

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q2.** You applied a Certificate CR, but no Secret appears. The CRD exists. What do you check next?
- A. The kubelet logs
- B. That the operator controller Pod is running, and its logs; also the CR's status and events
- C. etcd health
- D. The CNI

<details><summary>▶ Answer</summary>

**B.** A CRD alone does nothing; the controller does the work.
</details>

**Q3.** `kubectl apply -f backup.yaml` fails with `no matches for kind "Backup" in version "ops.example.com/v1"`. Cause?
- A. RBAC denies it
- B. The CRD is not installed (or that version is not served)
- C. The namespace is missing
- D. Wrong kubectl version

<details><summary>▶ Answer</summary>

**B.**
</details>

---

## Chapter 4: exam-style tasks

**T4.1 (5 min)** Create ServiceAccount `pipeline` in namespace `cicd`. It must be able to create, update and delete Deployments and StatefulSets in `cicd` only, and list Pods there. Verify.

<details><summary>▶ Solution</summary>

```bash
k create ns cicd ; k create sa pipeline -n cicd
k create role pipeline -n cicd --verb=create,update,delete --resource=deployments.apps,statefulsets.apps
k create role pod-list -n cicd --verb=list --resource=pods
k create rolebinding pipeline -n cicd --role=pipeline --serviceaccount=cicd:pipeline
k create rolebinding pod-list -n cicd --role=pod-list --serviceaccount=cicd:pipeline
k auth can-i delete statefulsets -n cicd --as=system:serviceaccount:cicd:pipeline   # yes
k auth can-i list pods -n default --as=system:serviceaccount:cicd:pipeline          # no
```
(One Role with two rules in YAML is equally valid.)
</details>

**T4.2 (5 min)** Using Helm, install chart `podinfo/podinfo` version `6.7.1` as release `shop` in namespace `web` with `replicaCount=3`. Then upgrade it so `ui.color` is `#ff0000` while keeping 3 replicas. Write the release's revision count to `/tmp/rev.txt`.

<details><summary>▶ Solution</summary>

```bash
helm repo add podinfo https://stefanprodan.github.io/podinfo && helm repo update
helm install shop podinfo/podinfo --version 6.7.1 -n web --create-namespace --set replicaCount=3
helm upgrade shop podinfo/podinfo --version 6.7.1 -n web --reuse-values --set ui.color='#ff0000'
helm history shop -n web
echo 2 > /tmp/rev.txt
```
</details>

**T4.3 (6 min)** Create user `dev1` with a client certificate signed by the cluster CA (CSR name `dev1`, group `developers`). Give the group `developers` the `edit` ClusterRole in namespace `app` only.

<details><summary>▶ Solution</summary>

Follow the CSR flow in 4.1 with `-subj "/CN=dev1/O=developers"`, then:

```bash
k create rolebinding developers-edit -n app --clusterrole=edit --group=developers
k auth can-i create deploy -n app --as=dev1 --as-group=developers       # yes
```
</details>

**T4.4 (4 min)** A Kustomize base is in `/opt/kustom/base`. Create an overlay in `/opt/kustom/staging` that puts everything in namespace `staging`, sets image `nginx` to tag `1.27-alpine`, and apply it.

<details><summary>▶ Solution</summary>

```bash
mkdir -p /opt/kustom/staging && cat > /opt/kustom/staging/kustomization.yaml <<'EOF'
resources: [../base]
namespace: staging
images:
  - {name: nginx, newTag: 1.27-alpine}
EOF
k create ns staging
k apply -k /opt/kustom/staging
```
</details>

**T4.5 (3 min)** Write the names of all CRDs in the cluster whose group contains `cert-manager.io` to `/tmp/crds.txt`, and the installed CNI's config file name to `/tmp/cni.txt` (node `cp1`).

<details><summary>▶ Solution</summary>

```bash
k get crd -o name | grep cert-manager.io > /tmp/crds.txt
ssh cp1 'ls /etc/cni/net.d/'      # then write the file name into /tmp/cni.txt
```
</details>
