# 7. Troubleshooting (30%)

Official competencies: troubleshoot clusters and nodes · troubleshoot cluster components · monitor cluster and application resource usage ·
manage and evaluate container output streams · troubleshoot services and networking.

This is the biggest domain. The skill is a **systematic method**, not memorised fixes. For every problem:

1. **Observe**: `get` (status, READY, RESTARTS), `describe` (Events at the bottom), `logs`.
2. **Locate the layer**: app → Pod spec → scheduling → node (kubelet/runtime/CNI) → control plane → network/DNS.
3. **Fix the cause**, not the symptom (do not just delete Pods).
4. **Verify** the end state that the task describes.

---

## 7.1 Container output streams, logs and events [`logs-and-events`]

### Key ideas
- Containers write to **stdout/stderr**. The runtime stores this per container on the node under `/var/log/pods/<ns>_<pod>_<uid>/<container>/N.log` (symlinked from `/var/log/containers/`). `kubectl logs` reads it through the kubelet.
- Logs of a **previous** (crashed) instance: `--previous` / `-p`. That is where a CrashLoopBackOff explains itself.
- Apps that log to a file inside the container are invisible to `kubectl logs`. The fix is a sidecar that tails the file to stdout (chapter 2.1).
- **Events** are short-lived objects (kept about 1 h by default) that explain scheduling, image pulls, probe failures, OOM kills and volume mounts.
- Node-level services (kubelet, containerd) log to **journald**.

```bash
k logs web                         # single-container Pod
k logs web -c sidecar              # pick the container
k logs web -p                      # previous instance (after a crash)
k logs web -f --tail=50 --since=10m --timestamps
k logs deploy/web                  # one Pod of the Deployment
k logs -l app=web --all-containers --prefix --max-log-requests=10
k logs job/backup

k get events -n dev --sort-by=.lastTimestamp
k get events -A --field-selector type=Warning
k get events --field-selector involvedObject.name=web-7d9c
k describe pod web | tail -20      # the Events section

# on the node
sudo journalctl -u kubelet -e --no-pager | tail -50
sudo journalctl -u containerd --since "10 min ago"
sudo ls /var/log/pods/ ; sudo tail /var/log/pods/kube-system_kube-apiserver-*/kube-apiserver/*.log
sudo crictl logs <container-id>
```

### Lab
1. Run a Pod that prints a line and exits with code 1 (`sh -c 'echo boom; exit 1'`). Read the logs with and without `-p` once it is in CrashLoopBackOff.
2. Write the last 20 log lines of all Pods labelled `app=web` (all containers, prefixed with the Pod name) to `/tmp/web.log`.
3. Find the log file for a Pod directly under `/var/log/pods/` on its node.
4. List only Warning events in all namespaces, sorted by time.

### Check questions

**Q1.** A Pod is in CrashLoopBackOff. `k logs pod` shows nothing useful. What next?
- A. `k logs pod --previous`
- B. `k delete pod`
- C. `k top pod`
- D. Restart kubelet

<details><summary>▶ Answer</summary>

**A.** The current instance may not have logged anything yet.
</details>

**Q2.** The apiserver is down. Where can you read its logs on a kubeadm control plane?
- A. `kubectl logs -n kube-system kube-apiserver-cp1`
- B. `crictl logs <id>` or `/var/log/pods/kube-system_kube-apiserver-*/`
- C. `journalctl -u kube-apiserver`
- D. `/etc/kubernetes/manifests/kube-apiserver.log`

<details><summary>▶ Answer</summary>

**B.** It is a static Pod, not a systemd unit, so C does not exist. A needs the apiserver.
</details>

**Q3.** An app writes logs only to `/var/log/app.log` inside its container. How do you get them into `kubectl logs` without changing the app?
- A. Mount a hostPath
- B. Add a sidecar sharing an emptyDir that tails the file to stdout
- C. Enable kubelet debug logging
- D. Use `k logs --file`

<details><summary>▶ Answer</summary>

**B.**
</details>

---

## 7.2 Troubleshoot applications and Pods [`troubleshoot-applications`]

### Status → cause cheat sheet

| What you see | Usual causes | Where to look |
|---|---|---|
| `Pending` | insufficient CPU/memory; taints without tolerations; nodeSelector/affinity matches nothing; unbound PVC; quota | `describe pod` Events ("0/3 nodes are available: …") |
| `ContainerCreating` (stuck) | volume mount failing (missing ConfigMap/Secret/PVC); CNI not ready; image still pulling | `describe pod` Events |
| `ImagePullBackOff` / `ErrImagePull` | typo in image or tag; private registry without `imagePullSecrets`; no network | Events: "not found", "unauthorized" |
| `CreateContainerConfigError` | missing ConfigMap/Secret key referenced in env | Events |
| `CrashLoopBackOff` | app exits: bad command/args, missing config, failing liveness probe, OOMKilled | `logs -p`, `describe` (Last State, Exit Code, Reason) |
| `Running` but `0/1 READY` | readiness probe failing | `describe` (Readiness probe failed) |
| `OOMKilled` (exit 137) | memory limit too low | `describe` Last State |
| `Completed` repeatedly in a Deployment | the command finishes; Deployments need long-running processes | container command |
| `Error` (exit 1/2/126/127) | 126 = not executable, 127 = command not found | `logs`, command |
| `Terminating` (stuck) | finalizers; node gone | `k get pod -o yaml` finalizers; last resort `--force --grace-period=0` |

### Tools

```bash
k describe pod web
k get pod web -o yaml | less                        # compare spec vs status
k get pod web -o jsonpath='{.status.containerStatuses[0].lastState}'
k exec -it web -- sh                                # if the image has a shell
k debug -it web --image=busybox:1.36 --target=web   # ephemeral debug container sharing the target's process namespace
k debug web -it --copy-to=web-dbg --container=web -- sh   # debug a copy with a changed command
k debug node/w1 -it --image=busybox:1.36            # shell on a node (host fs at /host)
k port-forward pod/web 8080:80                      # test the app directly
k get pod web -o yaml > fix.yaml && vi fix.yaml && k replace --force -f fix.yaml
```

### Lab: break-and-fix drill (do each, then describe the cause in one sentence)

```bash
k create deploy b1 --image=nginx:1.277                          # ImagePullBackOff → fix the tag
k run b2 --image=busybox -- sh -c 'exit 3'                       # CrashLoopBackOff → logs -p, exit code
k run b3 --image=nginx $do > b3.yaml   # add resources.requests.cpu: "100" → Pending (Insufficient cpu)
k create deploy b4 --image=nginx $do > b4.yaml   # add nodeSelector disk=nvme → Pending (no matching node)
k run b5 --image=nginx $do > b5.yaml   # add env from configMapKeyRef name=nope → CreateContainerConfigError
k run b6 --image=nginx $do > b6.yaml   # add a livenessProbe httpGet port 8080 → restarts every ~30 s
k run b7 --image=nginx $do > b7.yaml   # add a PVC volume claimName=missing → Pending
k run b8 --image=polinux/stress -- stress --vm 1 --vm-bytes 300M   # with limits.memory 100Mi → OOMKilled
```

Then ask a friend (or write a script) to break things without telling you what they changed, and fix each within 5 minutes.

### Check questions

**Q1.** Event: `0/3 nodes are available: 1 node(s) had untolerated taint {node-role.kubernetes.io/control-plane: }, 2 Insufficient memory.` Fix?
- A. Add a toleration for the control-plane taint
- B. Reduce the Pod's memory request or free/add capacity on the workers
- C. Restart the scheduler
- D. Raise the memory limit

<details><summary>▶ Answer</summary>

**B.** The workers are the intended targets; they lack memory for the request. A works technically but is rarely what the task wants.
</details>

**Q2.** Container status shows `Last State: Terminated, Reason: Error, Exit Code: 127`. Likely cause?
- A. OOM
- B. The command or binary was not found in the image
- C. Liveness probe
- D. Image pull failure

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q3.** The image has no shell (distroless). You need to inspect the running process's environment. Best tool?
- A. `k exec -- sh`
- B. `k debug -it pod --image=busybox --target=<container>`
- C. `k logs`
- D. ssh to the node and docker exec

<details><summary>▶ Answer</summary>

**B.**
</details>

---

## 7.3 Monitor cluster and application resource usage [`resource-monitoring`]

### Key ideas
- `kubectl top` needs **metrics-server** (it scrapes the kubelets' resource metrics). It shows **current usage**, not requests or limits.
- Requests and limits per node: `kubectl describe node` → "Allocated resources".
- Tasks look like "find the Pod using the most CPU in namespace X and write its name to a file".

```bash
k top nodes
k top nodes --sort-by=memory
k top pods -A --sort-by=cpu | head
k top pods -n prod --containers
k top pods -l app=web --no-headers | sort -k3 -h | tail -1     # by memory

# highest-CPU Pod name only
k top pods -n prod --sort-by=cpu --no-headers | head -1 | awk '{print $1}' > /opt/top-pod.txt

k describe node w1 | sed -n '/Allocated resources/,/Events/p'
k get pods -A -o custom-columns=NS:.metadata.namespace,NAME:.metadata.name,CPU_REQ:.spec.containers[*].resources.requests.cpu
k get --raw /apis/metrics.k8s.io/v1beta1/nodes | head
```

### Lab
1. Run a CPU hog (`k run hog --image=busybox -- sh -c 'while true; do :; done'`). Find it with `k top pods -A --sort-by=cpu`.
2. Write the name of the node with the highest memory usage to `/tmp/node.txt`.
3. If `k top` says `Metrics API not available`, debug metrics-server: is the Pod running, what do its logs say (TLS to kubelets?), is the APIService healthy (`k get apiservice v1beta1.metrics.k8s.io`)?

### Check questions

**Q1.** `kubectl top pods` fails with `error: Metrics API not available`. Most likely cause:
- A. kubelet is down
- B. metrics-server is not installed or not healthy
- C. RBAC
- D. CoreDNS is down

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q2.** `kubectl top` shows a Pod using 50m CPU, and its request is 500m. What does `top` report?
- A. Requests
- B. Limits
- C. Actual current usage
- D. The average since creation

<details><summary>▶ Answer</summary>

**C.**
</details>

---

## 7.4 Troubleshoot control-plane components [`troubleshoot-control-plane`]

### Key ideas
- On kubeadm, the apiserver, etcd, scheduler and controller-manager are **static Pods**. When one is broken, the manifest in `/etc/kubernetes/manifests/` is almost always the cause: a typo in a flag, a wrong file path or port, a bad cert path, a YAML error, or a wrong image tag.
- Symptoms by component:

| Broken | Symptom |
|---|---|
| kube-apiserver | `kubectl`: "connection refused :6443" / timeouts. Nothing works. |
| etcd | apiserver crash-loops (cannot reach etcd) → same as above |
| kube-scheduler | new Pods stay `Pending` with **no events** from the scheduler |
| kube-controller-manager | Deployments do not create ReplicaSets/Pods; scaling does nothing; Service endpoints are not updated; nodes never marked NotReady |
| kubelet (on cp) | static Pods not (re)started; node NotReady |

### Procedure when kubectl works

```bash
k get pods -n kube-system
k describe pod kube-scheduler-cp1 -n kube-system
k logs kube-controller-manager-cp1 -n kube-system
```

### Procedure when the apiserver is down

```bash
ssh cp1 ; sudo -i
crictl ps -a | grep -E 'apiserver|etcd'           # Exited? restarting?
crictl logs <container-id> 2>&1 | tail -20        # the actual error
ls /var/log/pods/ | grep apiserver                # if no container exists at all (YAML error)
journalctl -u kubelet -e | grep -i apiserver      # kubelet reports manifest parse errors here
vi /etc/kubernetes/manifests/kube-apiserver.yaml  # fix it
watch crictl ps                                    # wait for it to come back (30–90 s)
```

Common planted errors:
- `--etcd-servers=https://127.0.0.1:2380` (peer port) instead of `2379`
- wrong `--client-ca-file` / `--tls-cert-file` path
- misspelled flag name (`--authorization-modes`)
- wrong `image:` tag
- YAML indentation broken → **no container at all**; the kubelet logs say "could not process manifest"
- scheduler/controller-manager `--kubeconfig=/etc/kubernetes/scheduler.conf` path typo
- a probe port or `command` changed

### Lab (lab B, or `docker exec` into the kind control plane)
Do each break, observe the symptom from kubectl's point of view, then fix it from the node:
1. In `kube-apiserver.yaml`, change `--etcd-servers` to port 2380.
2. In `kube-scheduler.yaml`, change `--kubeconfig` to `/etc/kubernetes/scheduler.conff`.
3. In `kube-controller-manager.yaml`, change the command to `kube-controller-manegar`. Then try scaling a Deployment.
4. Break the YAML indentation in `etcd.yaml`.
5. Move `kube-apiserver.yaml` to `/root/` and back.

### Check questions

**Q1.** A Deployment was scaled from 2 to 5, but `k get rs` still shows 2 desired. Pods are healthy. Which component is broken?
- A. kube-scheduler
- B. kube-controller-manager
- C. kubelet
- D. kube-proxy

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q2.** New Pods stay Pending with no events at all, and nodes have plenty of resources. Which component?
- A. kube-scheduler
- B. etcd
- C. CoreDNS
- D. kube-proxy

<details><summary>▶ Answer</summary>

**A.**
</details>

**Q3.** After you edited `kube-apiserver.yaml`, `crictl ps -a` shows no apiserver container at all. What is the most likely cause?
- A. A wrong flag value
- B. A YAML syntax error, so kubelet cannot parse the manifest (check `journalctl -u kubelet`)
- C. etcd down
- D. Cert expired

<details><summary>▶ Answer</summary>

**B.** A wrong flag value gives a container that exits repeatedly.
</details>

---

## 7.5 Troubleshoot clusters and nodes [`troubleshoot-nodes`]

### Key ideas
A node is `NotReady` when its kubelet stops posting status (after about 40 s), or reports a problem (container runtime down, CNI not ready, disk/memory/PID pressure).

### Checklist for a NotReady node

```bash
k get nodes ; k describe node w1           # Conditions: Ready, MemoryPressure, DiskPressure, PIDPressure; Events
ssh w1 ; sudo -i
systemctl status kubelet                   # inactive? failed? disabled?
journalctl -u kubelet -e --no-pager | tail -40
systemctl status containerd
crictl info | grep -i -A3 networkready     # CNI status
ls /etc/cni/net.d /opt/cni/bin
cat /var/lib/kubelet/config.yaml           # kubelet config (clusterDNS, staticPodPath, cgroupDriver...)
cat /etc/kubernetes/kubelet.conf           # kubelet's kubeconfig: correct server address?
cat /usr/lib/systemd/system/kubelet.service.d/10-kubeadm.conf    # how kubelet is started (some distros: /etc/systemd/system/kubelet.service.d/)
cat /var/lib/kubelet/kubeadm-flags.env     # extra flags
df -h ; free -m                            # disk and memory pressure
```

Common planted errors and fixes:

| Cause | Fix |
|---|---|
| kubelet stopped / disabled | `systemctl enable --now kubelet` |
| containerd stopped | `systemctl enable --now containerd` |
| wrong path or typo in `/var/lib/kubelet/config.yaml` or the systemd drop-in | fix the file, `systemctl daemon-reload && systemctl restart kubelet` |
| wrong apiserver address/port in `/etc/kubernetes/kubelet.conf` | fix the `server:` line, restart kubelet |
| kubelet client cert expired or missing | check `/var/lib/kubelet/pki/`; rejoin if needed |
| swap enabled after a reboot | `swapoff -a` and fix fstab |
| CNI config missing | restore `/etc/cni/net.d/*` or reinstall the CNI DaemonSet |
| disk full | clean up (`crictl rmi --prune`, old logs) |

### Lab (lab B)
For each one, break it, watch `k get nodes` change, diagnose from the node, and fix:
1. `systemctl stop kubelet` on w1.
2. `systemctl disable --now containerd` on w1.
3. In `/var/lib/kubelet/config.yaml`, change `clientCAFile` to a wrong path, then restart kubelet.
4. In `/etc/kubernetes/kubelet.conf`, change the server port to 6444.
5. Add a nonsense flag to `KUBELET_KUBEADM_ARGS` in `/var/lib/kubelet/kubeadm-flags.env`.
6. Fill the disk (`fallocate -l 15G /bigfile`) and read the node conditions and evictions.

### Check questions

**Q1.** Node w1 is NotReady. On w1, `systemctl status kubelet` shows `inactive (dead)`, and it is also disabled. Correct fix?
- A. `systemctl start kubelet`
- B. `systemctl enable --now kubelet`
- C. `kubeadm reset`
- D. `kubectl uncordon w1`

<details><summary>▶ Answer</summary>

**B.** A works until the next reboot; graders may check that the service is enabled.
</details>

**Q2.** The kubelet logs show `failed to run Kubelet: validate service connection: CRI v1 runtime API is not implemented for endpoint "unix:///run/containerd/containerd.sock"`, or a connection refused on that socket. What do you check?
- A. CNI config
- B. That containerd is running and the socket path is correct
- C. kube-proxy
- D. RBAC

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q3.** You changed `/var/lib/kubelet/config.yaml`. How do you apply it?
- A. `kubectl apply -f`
- B. `systemctl restart kubelet`
- C. kubelet picks it up automatically within 20 s
- D. Reboot the node

<details><summary>▶ Answer</summary>

**B.** Use `daemon-reload` as well when you change systemd unit files.
</details>

---

## 7.6 Troubleshoot services and networking [`troubleshoot-networking`]

### Method: work outward, one hop at a time

```text
client Pod
  │ 1. DNS: nslookup svc            → fails? CoreDNS (7.6-a)
  ▼
Service ClusterIP:port
  │ 2. endpoints exist?             → none? selector / readiness (7.6-b)
  ▼
Pod IP:targetPort
  │ 3. curl Pod IP directly         → fails? app / port / NetworkPolicy (7.6-c)
  ▼
container listening?               → k exec -- netstat -tlnp / ss -tlnp
```

```bash
# 1. DNS
k run t --image=busybox:1.36 --rm -it --restart=Never -- nslookup web.prod
k -n kube-system get pods -l k8s-app=kube-dns ; k -n kube-system logs -l k8s-app=kube-dns
k -n kube-system get endpointslices -l kubernetes.io/service-name=kube-dns

# 2. Service → endpoints
k get svc web -n prod -o wide                         # SELECTOR column
k get pods -n prod -l app=web --show-labels           # do labels match? READY?
k get endpointslices -n prod -l kubernetes.io/service-name=web
k describe svc web -n prod                            # TargetPort vs the container's port

# 3. direct to the Pod
k get pod -n prod -l app=web -o wide                  # Pod IP
k run t -n prod --image=busybox:1.36 --rm -it --restart=Never -- wget -qO- -T2 <podIP>:<port>
k get netpol -n prod ; k describe netpol -n prod

# 4. node level
k -n kube-system get ds kube-proxy ; k -n kube-system logs ds/kube-proxy
k get pods -A -o wide | grep -Ei 'calico|cilium|flannel'
```

| Symptom | Likely cause |
|---|---|
| `nslookup` times out for every name | CoreDNS down or broken Corefile; egress NetworkPolicy blocks 53; kube-dns Service has no endpoints |
| Name resolves, connection to Service IP hangs | no endpoints (selector mismatch, Pods not Ready); NetworkPolicy; kube-proxy broken |
| Connection refused | wrong targetPort; the app listens on another port or on 127.0.0.1 only |
| Works by Pod IP, not by Service IP | kube-proxy (DaemonSet down, wrong mode or config) |
| Works on the same node, not across nodes | CNI (overlay or routing between nodes, a CNI Pod down on one node) |
| NodePort works on one node only | `externalTrafficPolicy: Local` with Pods on a single node; firewall |
| Ingress/Gateway 404 | host/path mismatch, wrong class, route not attached (check status conditions) |
| Ingress/Gateway 503 | backend Service has no endpoints, or the port is wrong |

### Lab: networking break-and-fix
1. Deploy `web` (nginx) + Service in `prod`. Break each, then fix:
   - Service selector `app: webb`
   - Service `targetPort: 8080`
   - a readiness probe that always fails
   - default-deny ingress NetworkPolicy
   - scale CoreDNS to 0
   - delete the kube-proxy DaemonSet Pods with a bad config (edit the `kube-proxy` ConfigMap `mode` to `"foo"`, then `rollout restart ds/kube-proxy`)
2. For each, write the symptom you saw at steps 1–3 of the method.

### Check questions

**Q1.** `nslookup web` works and returns the ClusterIP, but `wget web` hangs. `k get endpointslices` for `web` shows no endpoints. Pods have label `app=web-app`; the Service selector is `app=web`. Fix?
- A. Restart CoreDNS
- B. Correct the Service selector (or the Pod labels) so they match
- C. Change the targetPort
- D. Add a NetworkPolicy

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q2.** Pods on node w1 can reach Pods on w1, but not Pods on w2. Where do you look?
- A. CoreDNS
- B. The CNI plugin (its Pod on each node, the overlay/routing between nodes)
- C. Service selectors
- D. RBAC

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q3.** `curl <podIP>:80` works, `curl <clusterIP>:80` fails, endpoints are correct. Which component?
- A. CoreDNS
- B. kube-proxy
- C. kube-scheduler
- D. etcd

<details><summary>▶ Answer</summary>

**B.**
</details>

---

## Chapter 7: exam-style tasks

**T7.1 (6 min)** Node `w2` is NotReady. Fix it so it is Ready, and make sure the fix survives a reboot.

<details><summary>▶ Solution</summary>

```bash
k describe node w2 | sed -n '/Conditions/,/Addresses/p'
ssh w2 ; sudo -i
systemctl status kubelet ; journalctl -u kubelet -e | tail -30
# typical: kubelet disabled/stopped, or a wrong path in config.yaml or the drop-in
systemctl daemon-reload ; systemctl enable --now kubelet
exit ; exit
k get nodes -w
```
</details>

**T7.2 (5 min)** In namespace `orders`, Deployment `api` has 0/3 Pods Ready. Find and fix the issue. Do not change the image.

<details><summary>▶ Solution</summary>

```bash
k get pods -n orders ; k describe pod -n orders <pod> | tail -20 ; k logs -n orders <pod> -p
```
Fix what the evidence shows: a missing ConfigMap key, a wrong command, a readiness probe on the wrong port, or a memory limit that causes OOMKilled. Edit the Deployment (`k edit deploy api -n orders`), then `k rollout status`.
</details>

**T7.3 (6 min)** `kubectl` returns `The connection to the server 10.0.0.10:6443 was refused`. Restore the cluster.

<details><summary>▶ Solution</summary>

```bash
ssh cp1 ; sudo -i
crictl ps -a | grep apiserver ; crictl logs <id> | tail
journalctl -u kubelet | grep -i manifest | tail
vi /etc/kubernetes/manifests/kube-apiserver.yaml       # fix the flag, path, port or YAML
watch crictl ps
```
</details>

**T7.4 (3 min)** Write the name of the Pod that uses the most memory in namespace `analytics` to `/opt/top-mem.txt` (name only).

<details><summary>▶ Solution</summary>

```bash
k top pods -n analytics --sort-by=memory --no-headers | head -1 | awk '{print $1}' > /opt/top-mem.txt
```
</details>

**T7.5 (6 min)** Service `frontend` in namespace `web` does not respond from other Pods. Fix it without recreating the Deployment.

<details><summary>▶ Solution</summary>

Walk through the method: DNS → endpoints (selector, readiness) → targetPort → NetworkPolicy. Then:

```bash
k get svc frontend -n web -o yaml | grep -A5 -E 'selector|ports'
k get pods -n web --show-labels
k get endpointslices -n web -l kubernetes.io/service-name=frontend
k get netpol -n web
```
Fix the mismatch with `k edit svc frontend -n web`, or adjust or add the NetworkPolicy, and verify with a temporary busybox Pod.
</details>

**T7.6 (3 min)** Write all Warning events from namespace `kube-system`, sorted by time, to `/opt/ks-warnings.txt`.

<details><summary>▶ Solution</summary>

```bash
k get events -n kube-system --field-selector type=Warning --sort-by=.lastTimestamp > /opt/ks-warnings.txt
```
</details>
