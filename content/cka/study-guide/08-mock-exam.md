# 8. Mock exam (2 hours, 17 tasks, pass = 66%)

Rules for yourself:
- Use lab A for the Kubernetes tasks and lab B for tasks marked **[kubeadm]**. Pretend each "context" is a separate cluster.
- Only one browser tab, on kubernetes.io/docs, helm.sh/docs and gateway-api.sigs.k8s.io.
- Set a 120-minute timer. Skim everything first. Flag tasks that take more than ~8 minutes.
- Grade yourself with the checklist under each solution. Partial credit = the fraction of check items you met.

The weights follow the official domain split: Troubleshooting 30, Cluster Architecture 25, Networking 20, Workloads 15, Storage 10.

| # | Domain | Weight | Topic |
|---|---|---|---|
| 1 | Cluster Arch | 7% | Upgrade control plane [kubeadm] |
| 2 | Cluster Arch | 6% | etcd backup + restore [kubeadm] |
| 3 | Cluster Arch | 4% | RBAC for a ServiceAccount |
| 4 | Cluster Arch | 4% | Helm install + upgrade |
| 5 | Cluster Arch | 4% | CRD + operator custom resource |
| 6 | Workloads | 5% | Deployment rollout + rollback |
| 7 | Workloads | 5% | Scheduling: taints + affinity |
| 8 | Workloads | 5% | HPA + resources |
| 9 | Networking | 7% | NetworkPolicy |
| 10 | Networking | 7% | Gateway API route |
| 11 | Networking | 6% | Service + DNS |
| 12 | Storage | 5% | PV/PVC/Pod |
| 13 | Storage | 5% | StorageClass + expansion |
| 14 | Troubleshooting | 8% | Broken node [kubeadm] |
| 15 | Troubleshooting | 8% | Broken control plane component [kubeadm] |
| 16 | Troubleshooting | 7% | Broken application |
| 17 | Troubleshooting | 7% | Logs, metrics, sidecar |

**Setup script** (run it in lab A first; it plants the scenarios for tasks 6, 11, 16 and 17):

```bash
k create ns mock
k -n mock create deploy shop --image=nginx:1.26 --replicas=3
k -n mock create deploy api --image=nginx --replicas=2
k -n mock label deploy api tier=api
k -n mock expose deploy api --port=80 --name=api-svc
k -n mock patch svc api-svc -p '{"spec":{"selector":{"app":"apii"}}}'        # planted bug for task 11
k -n mock create cm app-config --from-literal=COLOR=blue
cat <<'EOF' | k apply -f -
apiVersion: apps/v1
kind: Deployment
metadata: {name: broken, namespace: mock}
spec:
  replicas: 2
  selector: {matchLabels: {app: broken}}
  template:
    metadata: {labels: {app: broken}}
    spec:
      containers:
        - name: app
          image: nginx:1.27
          env: [{name: COLOR, valueFrom: {configMapKeyRef: {name: app-config, key: colour}}}]
          readinessProbe: {httpGet: {path: /, port: 8080}}
          resources: {limits: {memory: 8Mi}}
EOF
k -n mock run legacy --image=busybox:1.36 -- sh -c 'while true; do echo "$(date) tick" >> /var/log/legacy.log; sleep 2; done'
```

---

### Task 1 (7%) [kubeadm]: upgrade the control plane
Context `lab-b`. The cluster runs v1.36.x. Upgrade the control plane node `cp1` (kubeadm, control-plane components, kubelet and kubectl) to the latest v1.37 patch release. Do not upgrade the workers. Make sure `cp1` is schedulable afterwards.

<details><summary>▶ Solution + checklist</summary>

Chapter 3.3, steps a–d, on cp1.
- [ ] `kubectl version` server = v1.37.x
- [ ] `kubectl get nodes`: cp1 is v1.37.x, workers are still v1.36.x
- [ ] cp1 is not `SchedulingDisabled`
</details>

### Task 2 (6%) [kubeadm]: etcd backup and restore
Save a snapshot of etcd on `cp1` to `/opt/mock/etcd-before.db`. Then create namespace `after-backup`. Restore the snapshot into `/var/lib/etcd-mock` and make the cluster use it.

<details><summary>▶ Solution + checklist</summary>

Chapter 3.4.
- [ ] `/opt/mock/etcd-before.db` exists and `etcdutl snapshot status` works on it
- [ ] etcd.yaml's `etcd-data` hostPath is `/var/lib/etcd-mock`
- [ ] `kubectl get ns after-backup` → NotFound
- [ ] all kube-system Pods Running
</details>

### Task 3 (4%): RBAC
Create ServiceAccount `reporter` in namespace `mock`. It must be able to `get`, `list` and `watch` Pods and Deployments in **all** namespaces, and nothing else.

<details><summary>▶ Solution + checklist</summary>

```bash
k create sa reporter -n mock
k create clusterrole reporter --verb=get,list,watch --resource=pods,deployments.apps
k create clusterrolebinding reporter --clusterrole=reporter --serviceaccount=mock:reporter
```
- [ ] `k auth can-i list deploy -A --as=system:serviceaccount:mock:reporter` → yes
- [ ] `k auth can-i delete pods -n default --as=...` → no
- [ ] `k auth can-i list secrets -n mock --as=...` → no
</details>

### Task 4 (4%): Helm
Add the repo `https://stefanprodan.github.io/podinfo` as `podinfo`. Install chart `podinfo/podinfo` as release `catalog` in namespace `helm-mock` with 2 replicas. Then upgrade the release so `ui.message=mock` while keeping 2 replicas.

<details><summary>▶ Solution + checklist</summary>

Chapter 4.2.
- [ ] `helm list -n helm-mock` shows `catalog` at revision 2
- [ ] `helm get values catalog -n helm-mock` shows both `replicaCount: 2` and `ui.message: mock`
- [ ] Deployment has 2 ready replicas
</details>

### Task 5 (4%): CRD
Apply the `Backup` CRD from chapter 4.5. Create a `Backup` named `weekly` in namespace `mock` with schedule `0 3 * * 0` and retention `4`. Write the output of `kubectl get backups -n mock` to `/opt/mock/backups.txt`.

<details><summary>▶ Solution + checklist</summary>

- [ ] `k get crd backups.ops.example.com`
- [ ] `k get bk weekly -n mock -o jsonpath='{.spec.retention}'` → 4
- [ ] the file contains the table with the SCHEDULE column
</details>

### Task 6 (5%): rollout
Deployment `shop` in `mock`: update the image to `nginx:1.27` with at most 1 Pod unavailable and at most 1 extra Pod during the rollout. Set the change cause to `bump 1.27`. Then update to `nginx:1.27-bad` and roll back to the `nginx:1.27` revision.

<details><summary>▶ Solution + checklist</summary>

Chapter 2.2.
- [ ] strategy maxSurge 1 / maxUnavailable 1
- [ ] the current image is `nginx:1.27` and all 3 replicas are available
- [ ] `k rollout history` shows `bump 1.27`
</details>

### Task 7 (5%): scheduling
Taint node `cka-worker2` with `workload=batch:NoSchedule`. Create Deployment `batch` (image `busybox:1.36`, command `sleep 3600`, 2 replicas) in `mock` that runs **only** on `cka-worker2`.

<details><summary>▶ Solution + checklist</summary>

Add a toleration for `workload=batch:NoSchedule` plus `nodeSelector: {kubernetes.io/hostname: cka-worker2}` (or node affinity).
- [ ] both Pods are on cka-worker2
- [ ] the taint is present on the node
- [ ] no other Pods without the toleration were scheduled there afterwards
</details>

### Task 8 (5%): autoscaling
Make Deployment `api` in `mock` autoscale between 2 and 8 replicas at 70% average CPU. Containers must request `100m` CPU and `64Mi` memory.

<details><summary>▶ Solution + checklist</summary>

```bash
k -n mock set resources deploy/api --requests=cpu=100m,memory=64Mi
k -n mock autoscale deploy api --min=2 --max=8 --cpu-percent=70
```
- [ ] the HPA exists with min 2, max 8, target 70%
- [ ] the HPA TARGETS column shows a number, not `<unknown>`
</details>

### Task 9 (7%): NetworkPolicy
In namespace `mock`, Pods of Deployment `api` (label `app=api`) must accept ingress **only** from Pods labelled `app=shop` in the same namespace, and from any Pod in namespaces labelled `team=monitoring`, on TCP 80. All other ingress to `api` Pods is denied. Other Pods are unaffected.

<details><summary>▶ Solution + checklist</summary>

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata: {name: api-ingress, namespace: mock}
spec:
  podSelector: {matchLabels: {app: api}}
  policyTypes: [Ingress]
  ingress:
    - from:
        - podSelector: {matchLabels: {app: shop}}
        - namespaceSelector: {matchLabels: {team: monitoring}}
      ports: [{protocol: TCP, port: 80}]
```
- [ ] a shop Pod can reach an api Pod on 80
- [ ] a Pod with another label in `mock` cannot
- [ ] a Pod in a namespace labelled `team=monitoring` can
</details>

### Task 10 (7%): Gateway API
GatewayClass `eg` exists (install Envoy Gateway from chapter 5.6 first). In namespace `mock`, create Gateway `mock-gw` with an HTTP listener on port 80. Create HTTPRoute `shop-route` for host `shop.mock.local`: path prefix `/api` → Service `api-svc:80`, and everything else → a Service `shop:80` (create it by exposing Deployment `shop`).

<details><summary>▶ Solution + checklist</summary>

Chapter 5.6 / T5.4 pattern, with two rules (put `/api` first for readability; Gateway API picks the longest match anyway).
- [ ] the Gateway is `Programmed=True`
- [ ] the HTTPRoute is `Accepted=True` and `ResolvedRefs=True`
- [ ] `curl -H 'Host: shop.mock.local' <gw>/api` reaches api (after task 11 is fixed)
</details>

### Task 11 (6%): Service + DNS
From a Pod in namespace `default`, `wget api-svc.mock` hangs. Fix it. Then write the fully qualified DNS name of the Service to `/opt/mock/fqdn.txt`.

<details><summary>▶ Solution + checklist</summary>

The selector is `app=apii`. Patch it to `app=api`.
- [ ] the EndpointSlice lists 2+ addresses
- [ ] wget succeeds from `default`
- [ ] the file contains `api-svc.mock.svc.cluster.local`
</details>

### Task 12 (5%): PV/PVC
Create PV `mock-pv` (1Gi, RWO, Retain, class `manual`, hostPath `/mnt/mock`), PVC `mock-pvc` (500Mi, class `manual`) in `mock`, and Pod `writer` (busybox, `sh -c 'echo hello > /data/hello; sleep 3600'`) mounting it at `/data`.

<details><summary>▶ Solution + checklist</summary>

Chapter 6.2 trio.
- [ ] the PVC is Bound to mock-pv
- [ ] `k exec writer -n mock -- cat /data/hello` → hello
</details>

### Task 13 (5%): StorageClass
Create StorageClass `mock-sc` using the cluster's existing provisioner, with `WaitForFirstConsumer`, Retain and expansion allowed, and make it the **only** default class.

<details><summary>▶ Solution + checklist</summary>

Chapter 6.3.
- [ ] `k get sc` shows `(default)` only next to mock-sc
- [ ] reclaimPolicy Retain, bindingMode WaitForFirstConsumer, allowVolumeExpansion true
</details>

### Task 14 (8%) [kubeadm]: broken node
Have a partner (or a script) do **one** of these on `w1` without telling you which: stop and disable kubelet; set a bad `clientCAFile` in `/var/lib/kubelet/config.yaml`; stop containerd; or change the server port in `/etc/kubernetes/kubelet.conf`. Make `w1` Ready, permanently.

<details><summary>▶ Solution + checklist</summary>

Chapter 7.5 checklist.
- [ ] w1 Ready
- [ ] `systemctl is-enabled kubelet containerd` → enabled
- [ ] it survives a `reboot`
</details>

### Task 15 (8%) [kubeadm]: broken control plane
A partner breaks **one** of: the scheduler's `--kubeconfig` path, the controller-manager's binary name, or the apiserver's `--etcd-servers` port. Find and fix it. Prove the cluster works by creating Deployment `proof` with 2 replicas and seeing them Running.

<details><summary>▶ Solution + checklist</summary>

Chapter 7.4.
- [ ] all control-plane Pods Running with 0 recent restarts
- [ ] `proof` has 2/2 Pods Running
</details>

### Task 16 (7%): broken application
Deployment `broken` in `mock` has no Ready Pods. Make it run with 2 Ready replicas. The intended config key is `colour`; you may change the ConfigMap but not the key name in the Deployment. nginx listens on port 80.

<details><summary>▶ Solution + checklist</summary>

Three planted faults:
1. The ConfigMap has `COLOR` but the Deployment wants `colour` → `k -n mock patch cm app-config -p '{"data":{"colour":"blue"}}'`
2. The readiness probe is on port 8080 → change it to 80
3. A memory limit of 8Mi gives OOMKilled → raise it (for example to 128Mi)

- [ ] 2/2 Ready
- [ ] the env var COLOR=blue inside a Pod
</details>

### Task 17 (7%): logs, metrics, sidecar
a) Write the name of the Pod in **all namespaces** that uses the most CPU to `/opt/mock/top-cpu.txt`.
b) Pod `legacy` in `mock` writes to `/var/log/legacy.log` inside its container. Recreate it (same name and command) with a native sidecar named `log-tail` that streams the file to stdout. Write the last 5 lines of `k logs legacy -c log-tail` to `/opt/mock/legacy.txt`.

<details><summary>▶ Solution + checklist</summary>

a) `k top pods -A --sort-by=cpu --no-headers | head -1 | awk '{print $2}'` (column 2 = name when `-A` is used).
b) Export the Pod YAML, add an `emptyDir` at `/var/log` in both containers and an `initContainers` entry `log-tail` with `restartPolicy: Always` running `tail -F /var/log/legacy.log`, then `k replace --force -f`.
- [ ] the file has one Pod name
- [ ] `legacy` has a sidecar `log-tail` with restartPolicy Always
- [ ] `/opt/mock/legacy.txt` has 5 "tick" lines
</details>

---

## After the mock
- Score ≥ 80%: book the exam; do killer.sh session 1 now and session 2 three days before the exam.
- 66–80%: redo the chapters behind your lost points, then repeat the mock with different planted faults.
- < 66%: focus on speed (the chapter 1 drills) and chapters 3 and 7, which are 55% of the exam between them.
