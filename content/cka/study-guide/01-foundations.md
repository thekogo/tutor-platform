# 1. Foundations: Kubernetes and kubectl

Foundations is not an exam domain, but every task in every domain is done with kubectl.
Speed and accuracy here decide whether you finish in 2 hours.

---

## 1.1 Kubernetes architecture [`k8s-architecture`]

### Key ideas

| Component | Runs where | How it runs on a kubeadm cluster | Job |
|---|---|---|---|
| **kube-apiserver** | control plane | static Pod | The only component that talks to etcd. Every client (kubectl, kubelet, controllers) goes through it. Handles authentication → authorization (RBAC) → admission → persistence. |
| **etcd** | control plane (or external) | static Pod | Key-value store holding all cluster state. Lose it without a backup = lose the cluster. |
| **kube-scheduler** | control plane | static Pod | Watches for Pods with no `nodeName` and picks a node (filter → score). |
| **kube-controller-manager** | control plane | static Pod | Runs the control loops: Deployment, ReplicaSet, Node, Job, EndpointSlice, ServiceAccount… Each loop moves *actual* state toward *desired* state. |
| **cloud-controller-manager** | control plane | optional | Cloud integrations (LoadBalancer Services, node lifecycle). Not present on bare kubeadm. |
| **kubelet** | every node | **systemd service** | Registers the node, watches the API for Pods bound to it, runs them through the CRI, reports status, runs probes. Also starts static Pods from `/etc/kubernetes/manifests`. |
| **container runtime** (containerd) | every node | systemd service | Pulls images, runs containers. kubelet talks to it over the **CRI** socket. |
| **kube-proxy** | every node | DaemonSet | Programs iptables/IPVS/nftables rules so Service IPs reach Pod IPs. (Some CNIs like Cilium can replace it.) |
| **CoreDNS** | cluster | Deployment in kube-system | DNS for Services and Pods. |
| **CNI plugin** (Calico, Cilium, Flannel…) | every node | DaemonSet + binaries | Gives each Pod an IP and connects Pods across nodes. |

**What happens on `kubectl create deployment web --image=nginx`:**
1. kubectl sends the Deployment to the apiserver; it is authenticated, authorized, admitted and stored in etcd.
2. The Deployment controller creates a ReplicaSet; the ReplicaSet controller creates Pods (no node yet).
3. The scheduler assigns each Pod a node (writes `spec.nodeName`).
4. The kubelet on that node sees the Pod, asks containerd (via CRI) to pull and run it, asks the CNI for an IP.
5. kubelet reports status back; the Pod becomes `Running` / `Ready`.

**Static Pods** are defined by files on a node, not by the API. kubelet watches `staticPodPath` (default `/etc/kubernetes/manifests`) and runs whatever YAML is there. The apiserver shows a read-only "mirror Pod" named `<name>-<nodename>`. To change one, **edit the file**; deleting the mirror Pod with kubectl does nothing permanent.

### Commands

```bash
k get nodes -o wide
k get pods -n kube-system -o wide              # see static Pods: etcd-cp1, kube-apiserver-cp1, ...
k cluster-info
k api-resources                                 # every resource type, short name, namespaced or not, API group
k get --raw='/readyz?verbose'                   # apiserver health checks
# on a node:
sudo systemctl status kubelet
sudo cat /var/lib/kubelet/config.yaml | grep staticPodPath
sudo ls /etc/kubernetes/manifests
```

### Lab

1. On your kind cluster, run `k get pods -n kube-system -o wide`. Identify which Pods are static (name ends with the node name) and which are from a Deployment/DaemonSet (random suffix).
2. `docker exec -it cka-control-plane bash`, then `ls /etc/kubernetes/manifests`. Open `kube-scheduler.yaml` and find the `--kubeconfig` flag.
3. Move the scheduler manifest away: `mv /etc/kubernetes/manifests/kube-scheduler.yaml /tmp/`. Exit, create a Deployment, and observe that Pods stay `Pending`. Move the file back and watch them get scheduled.
4. Run `k api-resources --namespaced=false` and list five cluster-scoped resources.

### Pitfalls
- kubelet is **not** a Pod. If kubelet is down, nothing on that node is managed, including static Pods.
- Deleting a static mirror Pod with kubectl: it comes right back. Fix static Pods by editing the manifest.

### Check questions

**Q1.** You moved `kube-scheduler.yaml` out of `/etc/kubernetes/manifests`. A new Deployment is created. What is the state of its Pods?
- A. ContainerCreating
- B. Pending, with no node assigned
- C. Running on the control plane node
- D. The Deployment is rejected by the apiserver

<details><summary>▶ Answer</summary>

**B.** The apiserver accepts the Deployment and the controllers still create the ReplicaSet and Pods, but nothing sets `spec.nodeName`, so the Pods stay Pending. D is wrong: the scheduler plays no part in admission.
</details>

**Q2.** Which component is the only one that reads and writes etcd directly?
- A. kube-controller-manager
- B. kubelet
- C. kube-apiserver
- D. kube-scheduler

<details><summary>▶ Answer</summary>

**C.** Everything else talks to the apiserver.
</details>

**Q3.** `kubectl delete pod kube-apiserver-cp1 -n kube-system` returns success. A few seconds later the Pod is listed again. Why?
- A. A ReplicaSet recreated it
- B. It is a static Pod; kubelet recreates it from the manifest on disk
- C. The deletion was blocked by a finalizer
- D. The DaemonSet controller recreated it

<details><summary>▶ Answer</summary>

**B.** The name ends in the node name (mirror Pod). Only removing or editing the file in `/etc/kubernetes/manifests` changes it.
</details>

---

## 1.2 Imperative kubectl and fast YAML [`kubectl-imperative`]

### Key ideas
The exam gives ~6–7 minutes per task. Writing YAML from memory is slow and error-prone. Instead:
1. **Generate** a skeleton with an imperative command + `$do` (`--dry-run=client -o yaml`).
2. **Edit** the few fields the generator cannot set.
3. **Apply** and **verify**.

### The generators you must know by heart

```bash
k run nginx --image=nginx --port=80 --labels=app=web $do > pod.yaml
k run busy --image=busybox --restart=Never $do -- sh -c 'sleep 3600'   # args after --
k run busy --image=busybox --restart=Never --command $do -- sleep 3600 # --command: sets command, not args
k create deployment web --image=nginx --replicas=3 --port=80 $do > deploy.yaml
k create service clusterip web --tcp=80:8080 $do
k expose deployment web --port=80 --target-port=8080 --type=NodePort $do
k create configmap app-cfg --from-literal=MODE=prod --from-file=app.conf $do
k create secret generic db --from-literal=PASSWORD=s3cret $do
k create secret tls web-tls --cert=tls.crt --key=tls.key
k create job pi --image=perl -- perl -Mbignum=bpi -wle 'print bpi(200)'
k create cronjob ping --image=busybox --schedule="*/5 * * * *" -- sh -c 'date'
k create namespace dev
k create serviceaccount app -n dev
k create role pod-reader --verb=get,list,watch --resource=pods -n dev
k create rolebinding rb --role=pod-reader --serviceaccount=dev:app -n dev
k create clusterrole node-reader --verb=get,list --resource=nodes
k create clusterrolebinding crb --clusterrole=node-reader --user=jane
k create ingress web --class=nginx --rule="app.example.com/=web:80"
k create quota q1 --hard=pods=10,requests.cpu=2 -n dev
k create priorityclass high --value=1000 --description="high"
k create token app -n dev                         # short-lived SA token
```

**Change live objects quickly**

```bash
k set image deploy/web nginx=nginx:1.27
k set resources deploy/web -c nginx --requests=cpu=100m,memory=128Mi --limits=memory=256Mi
k set env deploy/web MODE=prod
k scale deploy/web --replicas=5
k label node w1 disk=ssd ; k label node w1 disk-            # add / remove
k annotate deploy/web kubernetes.io/change-cause="bump image"
k taint node w1 dedicated=gpu:NoSchedule ; k taint node w1 dedicated-
k edit deploy/web                                           # opens $EDITOR on the live object
k patch deploy/web -p '{"spec":{"replicas":2}}'
k replace --force -f pod.yaml                               # delete + recreate (for fields you cannot edit on a Pod)
```

**Look things up without the browser**

```bash
k explain pod.spec.containers.livenessProbe
k explain deploy.spec.strategy --recursive | less
k api-resources | grep -i gateway
k <cmd> -h | less                                           # examples are at the top
```

**Output tricks**

```bash
k get pods -o wide --show-labels
k get pods -l app=web,tier!=db
k get pods -A --field-selector status.phase!=Running
k get pod web -o jsonpath='{.status.podIP}'
k get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.nodeInfo.kubeletVersion}{"\n"}{end}'
k get pods -o custom-columns=NAME:.metadata.name,NODE:.spec.nodeName,IMAGE:.spec.containers[*].image
k get pods --sort-by=.metadata.creationTimestamp
k get pods -o name > /opt/answer.txt
```

### Lab
1. Generate a Pod named `web` (image `nginx:alpine`, label `tier=front`, port 80) to `web.yaml` **without** opening an editor, then apply it.
2. Generate a Deployment `api` with 3 replicas, then add a CPU request of `50m` using `k set resources`.
3. Make a Pod that runs `sleep 3600` with `--command`, then `k get pod <name> -o yaml | grep -A3 command` and compare with the args version.
4. Write the IPs of all Pods labelled `tier=front` into `/tmp/ips.txt`, one per line, using jsonpath.
5. Time yourself creating these five objects imperatively: namespace, ServiceAccount, Role, RoleBinding, ConfigMap. Target: under 2 minutes.

### Pitfalls
- `k run x --image=busybox -- sleep 3600` sets **args**; with `--command` it sets **command**. With busybox both work, but for images with an ENTRYPOINT they differ.
- Most Pod spec fields are immutable. `k edit pod` saves a copy in `/tmp/kubectl-edit-*.yaml`; then use `k replace --force -f <that file>`.
- `--dry-run=server` validates against the apiserver (including admission) without saving. Use it to check a manifest.

### Check questions

**Q1.** Which command creates YAML for a Pod running `busybox` with **command** `sleep 1d`, without creating it?
- A. `k run b --image=busybox -- sleep 1d -o yaml`
- B. `k run b --image=busybox --command --dry-run=client -o yaml -- sleep 1d`
- C. `k create pod b --image=busybox --command sleep 1d`
- D. `k run b --image=busybox --dry-run=server -- sleep 1d`

<details><summary>▶ Answer</summary>

**B.** Flags must come before `--`, and `--command` makes the trailing words the command. C: there is no `create pod` generator. D creates nothing either, but it gives no YAML and sets args, not command.
</details>

**Q2.** You must add a liveness probe to a running Pod `web`. The fastest correct approach is:
- A. `k edit pod web` and save; the change applies in place
- B. `k get pod web -o yaml > w.yaml`, add the probe, then `k replace --force -f w.yaml`
- C. `k patch pod web` with the probe
- D. `k set probe pod/web`

<details><summary>▶ Answer</summary>

**B.** Probes are immutable on a running Pod, so A and C are rejected. D does not exist. (Equivalent: `k edit`, let it fail, then `k replace --force -f /tmp/kubectl-edit-xxx.yaml`.)
</details>

**Q3.** What does this print? `k get nodes -o jsonpath='{.items[*].metadata.name}'`
- A. One node name per line
- B. All node names on one line, separated by spaces
- C. A JSON array of names
- D. An error, because jsonpath needs `range`

<details><summary>▶ Answer</summary>

**B.** Use `{range .items[*]}{.metadata.name}{"\n"}{end}` to get one name per line.
</details>

---

## 1.3 YAML manifests, labels and selectors [`yaml-manifests`]

### Key ideas
Every object has four top-level parts:

```yaml
apiVersion: apps/v1          # group/version — find with `k api-resources` or `k explain deploy`
kind: Deployment
metadata:
  name: web
  namespace: dev
  labels: {app: web}         # identifying, selectable
  annotations: {owner: team-a}   # non-identifying metadata, not selectable
spec:                        # desired state (status is written by the system)
  replicas: 3
  selector:
    matchLabels: {app: web}  # MUST match template labels
  template:
    metadata:
      labels: {app: web}
    spec:
      containers:
        - name: nginx
          image: nginx:1.27
```

- **Labels + selectors** are how Kubernetes objects find each other: a Deployment finds its Pods, a Service finds its endpoints, a NetworkPolicy picks Pods, a PDB protects Pods.
- **Equality** selectors: `app=web`, `app!=web`. **Set** selectors: `env in (prod,stage)`, `!canary`. In YAML: `matchLabels` and/or `matchExpressions` (operators `In`, `NotIn`, `Exists`, `DoesNotExist`). Services only support `selector:` with equality (a plain map).
- A Deployment's `spec.selector` is **immutable** after creation.
- Common apiVersions: `v1` (Pod, Service, ConfigMap, Secret, PV, PVC, Namespace, ServiceAccount), `apps/v1` (Deployment, StatefulSet, DaemonSet, ReplicaSet), `batch/v1` (Job, CronJob), `networking.k8s.io/v1` (Ingress, NetworkPolicy, IngressClass), `rbac.authorization.k8s.io/v1`, `storage.k8s.io/v1` (StorageClass), `autoscaling/v2` (HPA), `policy/v1` (PodDisruptionBudget), `gateway.networking.k8s.io/v1` (Gateway, HTTPRoute).

### Lab
1. Create three Pods with labels `app=shop,env=prod`, `app=shop,env=dev`, `app=blog,env=prod`. List: all prod Pods; all shop Pods that are not dev; Pods whose `env` is in (dev, stage).
2. Create a Deployment, then relabel one of its Pods: `k label pod <pod> app=other --overwrite`. Watch the ReplicaSet create a replacement (the relabelled Pod is now "orphaned" and still running).
3. Write a Deployment whose selector does **not** match the template labels and apply it. Read the error.
4. Use `k apply --dry-run=server -f` on a manifest that has a typo in a field name, and compare with `--dry-run=client`.

### Pitfalls
- YAML indentation: lists under `containers:` need a `-`. Mixing tabs causes "found character that cannot start any token".
- `k apply` with a strict field check rejects unknown fields (`strict decoding error: unknown field`). Read the field path it prints.

### Check questions

**Q1.** A Service has `selector: {app: web, tier: front}`. Which Pod becomes an endpoint?
- A. labels `app=web`
- B. labels `app=web, tier=front, version=v2`
- C. labels `tier=front`
- D. labels `app=web, tier=back`

<details><summary>▶ Answer</summary>

**B.** The Pod must have *all* selector labels. Extra labels are fine.
</details>

**Q2.** Which `apiVersion` is correct for a NetworkPolicy?
- A. `v1`
- B. `apps/v1`
- C. `networking.k8s.io/v1`
- D. `policy/v1`

<details><summary>▶ Answer</summary>

**C.** `policy/v1` is for PodDisruptionBudget.
</details>

**Q3.** You relabel one Pod of a 3-replica Deployment so it no longer matches the selector. What happens?
- A. The Pod is deleted
- B. The ReplicaSet creates a new Pod; the relabelled Pod keeps running unmanaged
- C. Nothing; the ReplicaSet still owns it through ownerReferences
- D. The Deployment rolls out a new revision

<details><summary>▶ Answer</summary>

**B.** The ReplicaSet counts Pods by selector, so it sees 2 and makes a third. It also releases the Pod (removes its ownerReference). This is a common debugging trick.
</details>

---

## 1.4 kubeconfig, contexts and namespaces [`kubeconfig-contexts`]

### Key ideas
- A **kubeconfig** (`~/.kube/config` or `$KUBECONFIG`) has three lists: **clusters** (server URL + CA), **users** (client cert/key or token) and **contexts** (cluster + user + default namespace). `current-context` selects one.
- **Every exam task starts with a line like** `kubectl config use-context k8s-c2`. Run it, every time, even if you think you are already on that cluster.
- Namespaced vs cluster-scoped: Pods, Services, Deployments, Roles are namespaced. Nodes, PVs, StorageClasses, ClusterRoles, Namespaces, CRDs are cluster-scoped.

```bash
k config get-contexts
k config current-context
k config use-context k8s-c2
k config set-context --current --namespace=dev     # default ns for this context
k config view --minify                             # only the current context
k config view --raw -o jsonpath='{.users[0].user.client-certificate-data}' | base64 -d | openssl x509 -noout -subject
k --kubeconfig /etc/kubernetes/admin.conf get nodes
k get pods -n dev ; k get pods -A
```

**Add a user to a kubeconfig** (from certificate files, tested in exams):

```bash
k config set-credentials jane --client-certificate=jane.crt --client-key=jane.key --embed-certs=true
k config set-context jane@k8s --cluster=kubernetes --user=jane --namespace=dev
k --context jane@k8s get pods
```

### Lab
1. `k config view --minify` and identify the cluster server URL and the user.
2. Create namespace `dev`, set it as the default namespace for your current context, create a Pod with no `-n`, and confirm it landed in `dev`. Reset the default namespace to `default`.
3. Make a second kind cluster (`kind create cluster --name second`). Practise switching contexts and listing nodes in each cluster.
4. Print the CN of the client certificate in your kubeconfig with `openssl`.

### Pitfalls
- Setting the namespace on the context and then forgetting about it in the next task (which uses another cluster or namespace). Prefer explicit `-n` in the exam.
- `k config use-context` with a typo prints an error and leaves you on the old context. Always read the output (`Switched to context "..."`).

### Check questions

**Q1.** What does `kubectl config set-context --current --namespace=prod` change?
- A. Creates namespace prod
- B. Sets the default namespace of the current context in the kubeconfig
- C. Moves all Pods to prod
- D. Sets `$KUBECONFIG`

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q2.** Which of these is cluster-scoped?
- A. ServiceAccount
- B. PersistentVolumeClaim
- C. PersistentVolume
- D. RoleBinding

<details><summary>▶ Answer</summary>

**C.** Check with `k api-resources --namespaced=false`.
</details>

**Q3.** A task begins with "use context `k8s-c3`". You do the work and it is correct, but you get 0 points. The most likely cause:
- A. You used imperative commands
- B. You ran the commands against the previous task's cluster
- C. You did not save a YAML file
- D. Your kubectl version was wrong

<details><summary>▶ Answer</summary>

**B.**
</details>

---

## 1.5 Exam environment, docs and time management [`exam-environment-strategy`]

### Key ideas
- **Interface**: a remote Linux desktop (PSI) with a terminal and a browser limited to the allowed docs. Copy/paste is `Ctrl+Shift+C/V` in the terminal. Each task shows the context, the weight (%), and sometimes a "Quick Reference" link.
- **Partial credit**: tasks are graded by scripts that check the end state (object exists, field values, file contents). A task with 4 checks can give partial points, so finish what you can.
- **SSH**: some tasks say `ssh cka-node1`. Use `sudo -i` for root. **Type `exit` to return** before the next task; a later `kubectl` on the wrong host may have no kubeconfig.
- **Output files**: when asked to write output to `/opt/course/x/file`, write exactly that path and the format requested.

### Strategy
1. First 2 minutes: aliases and vim settings (see lab setup), skim all tasks, note weights.
2. Pass 1: do every task you can finish in under ~6 minutes. Flag the rest.
3. Pass 2: flagged tasks by weight.
4. Last 10 minutes: re-check contexts and namespaces on finished tasks.
5. Always verify: `get`, `describe`, `logs`, `auth can-i`, `curl` from a temporary Pod.

### Doc pages to bookmark mentally (find them fast with the docs search)

| Need | Search term |
|---|---|
| etcd backup/restore | "Operating etcd clusters" |
| kubeadm upgrade | "Upgrading kubeadm clusters" |
| kubeadm install | "Installing kubeadm", "Creating a cluster with kubeadm" |
| NetworkPolicy examples | "Network Policies" |
| PV/PVC/Pod example | "Configure a Pod to Use a PersistentVolume" |
| Ingress | "Ingress" |
| Gateway API | "Gateway API" (k8s docs) and gateway-api.sigs.k8s.io |
| Probes | "Configure Liveness, Readiness and Startup Probes" |
| Affinity/taints | "Assigning Pods to Nodes", "Taints and Tolerations" |
| kubectl cheat sheet | "kubectl Quick Reference" |
| Helm | helm.sh/docs → "Using Helm" |

### Lab
- Do a "speed drill" every study day: 10 random imperative objects in 10 minutes.
- Practise finding each page in the table above within 30 seconds using only the docs search box.

### Check questions

**Q1.** You have 25 minutes left and three tasks: A (4%, about 4 min), B (13%, about 15 min) and C (7%, about 6 min). What order maximises the expected score?
- A. B, C, A
- B. A, C, B
- C. B, A, C
- D. B only

<details><summary>▶ Answer</summary>

**B.** A and C are high value per minute and fit in 10 minutes, leaving 15 minutes for B, which may still earn partial credit. Starting with B risks running out of time before the two quick tasks.
</details>

**Q2.** After finishing a task on `ssh node01`, `kubectl get nodes` in the next task fails with `connection refused` to `localhost:8080`. Why?
- A. The apiserver crashed
- B. You are still on node01, which has no kubeconfig for your user
- C. The context is wrong
- D. kubectl needs an update

<details><summary>▶ Answer</summary>

**B.** `localhost:8080` is kubectl's default when it finds no kubeconfig. Type `exit`.
</details>

---

## Chapter 1: exam-style tasks

**T1.1 (3 min)** In namespace `apps` (create it), create a Pod `cache` with image `redis:7-alpine`, label `role=cache`, and CPU request `100m`. Write the Pod's node name into `/tmp/cache-node.txt`.

<details><summary>▶ Solution</summary>

```bash
k create ns apps
k run cache -n apps --image=redis:7-alpine --labels=role=cache $do > cache.yaml
# add under the container:  resources: {requests: {cpu: 100m}}
k apply -f cache.yaml
k get pod cache -n apps -o jsonpath='{.spec.nodeName}' > /tmp/cache-node.txt
```
</details>

**T1.2 (3 min)** List every Pod in all namespaces with its namespace, name and node, sorted by node name, and write it to `/tmp/pods-by-node.txt`.

<details><summary>▶ Solution</summary>

```bash
k get pods -A -o custom-columns=NS:.metadata.namespace,NAME:.metadata.name,NODE:.spec.nodeName \
  --sort-by=.spec.nodeName > /tmp/pods-by-node.txt
```
</details>

**T1.3 (2 min)** Find which API group the resource `cronjobs` belongs to and whether it is namespaced; write `group,namespaced` (for example `batch,true`) to `/tmp/cron.txt`.

<details><summary>▶ Solution</summary>

```bash
k api-resources | grep -i cronjob     # cronjobs  cj  batch/v1  true  CronJob
echo "batch,true" > /tmp/cron.txt
```
</details>

**T1.4 (4 min)** On the control plane node, find the static Pod path that kubelet uses and the `--service-cluster-ip-range` configured on the apiserver. Write both to `/tmp/cp.txt`, one per line.

<details><summary>▶ Solution</summary>

```bash
ssh cp1        # or docker exec -it cka-control-plane bash
sudo grep staticPodPath /var/lib/kubelet/config.yaml
sudo grep service-cluster-ip-range /etc/kubernetes/manifests/kube-apiserver.yaml
exit
# write the two values into /tmp/cp.txt on the machine the task specifies
```
</details>
