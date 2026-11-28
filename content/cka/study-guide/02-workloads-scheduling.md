# 2. Workloads & Scheduling (15%)

Official competencies: rolling updates and rollbacks · ConfigMaps and Secrets · workload autoscaling ·
self-healing primitives · Pod admission and scheduling (limits, node affinity, etc.).

---

## 2.1 Pods, init containers and sidecars [`pods-multicontainer`]

### Key ideas
- A Pod is one or more containers that share a **network namespace** (same IP, talk over `localhost`), can share **volumes**, and are scheduled together.
- **Init containers** (`spec.initContainers`) run **one at a time, in order, each to completion**, before any app container starts. If one fails, kubelet retries it (per `restartPolicy`). Use them for "wait for DB", "download config", "fix permissions".
- **Native sidecars** (stable since v1.33): an init container with `restartPolicy: Always`. It starts before the app containers, keeps running alongside them, is restarted if it dies, and is stopped **after** the app containers. It does not block Job completion. Use for log shippers, proxies.
- Classic sidecar = just a second entry in `containers`; works, but has no ordering guarantees.
- Pod `restartPolicy`: `Always` (default, required for Deployments), `OnFailure`, `Never` (Jobs use one of the last two).
- Container `command` overrides the image ENTRYPOINT; `args` overrides CMD.

```yaml
apiVersion: v1
kind: Pod
metadata: {name: app, labels: {app: demo}}
spec:
  volumes:
    - name: logs
      emptyDir: {}
  initContainers:
    - name: wait-db
      image: busybox:1.36
      command: ['sh', '-c', 'until nslookup db.default.svc.cluster.local; do sleep 2; done']
    - name: log-shipper              # native sidecar
      image: busybox:1.36
      restartPolicy: Always
      command: ['sh', '-c', 'tail -F /var/log/app/app.log']
      volumeMounts: [{name: logs, mountPath: /var/log/app}]
  containers:
    - name: app
      image: busybox:1.36
      command: ['sh', '-c', 'while true; do date >> /var/log/app/app.log; sleep 5; done']
      volumeMounts: [{name: logs, mountPath: /var/log/app}]
```

### Commands

```bash
k get pod app                          # Init:0/2 → PodInitializing → Running (READY 2/2 counts the sidecar)
k logs app -c wait-db
k logs app -c log-shipper -f
k exec -it app -c app -- sh
k describe pod app | sed -n '/Init Containers/,/Containers:/p'
```

### Lab
1. Apply the Pod above **without** a `db` Service. Observe `Init:0/2`. Read the `wait-db` logs.
2. `k create svc clusterip db --tcp=5432` and watch the Pod move to `Running`.
3. Confirm the sidecar prints the app's log lines: `k logs app -c log-shipper`.
4. Kill the sidecar process (`k exec app -c log-shipper -- kill 1`) and check its RESTARTS count.
5. Make a two-container Pod where container `b` runs `wget -qO- localhost:80` against an nginx container `a`. Confirm they share the network.

### Pitfalls
- Exam tasks often say "add a sidecar container named X that does Y". Read whether they want a plain extra container or `restartPolicy: Always` in initContainers. If the wording is "sidecar" with no detail, either form is usually accepted, but match the container name exactly.
- `k logs pod` on a multi-container Pod without `-c` errors or uses the default container. Always pass `-c`.

### Check questions

**Q1.** A Pod has init containers `a` then `b`, and app container `c`. `b` exits with code 1. What happens?
- A. `c` starts anyway
- B. `b` is retried; `c` does not start until `b` succeeds (with restartPolicy Always/OnFailure)
- C. `a` is re-run first
- D. The Pod is deleted

<details><summary>▶ Answer</summary>

**B.** The status shows `Init:Error` / `Init:CrashLoopBackOff`.
</details>

**Q2.** Which spec makes a native sidecar?
- A. `containers[].sidecar: true`
- B. `initContainers[].restartPolicy: Always`
- C. `spec.restartPolicy: Sidecar`
- D. `containers[].lifecycle.type: sidecar`

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q3.** Two containers in one Pod need to exchange files. What is the simplest mechanism?
- A. A shared `emptyDir` volume mounted in both
- B. A hostPath on each node
- C. A PVC per container
- D. They share a filesystem by default

<details><summary>▶ Answer</summary>

**A.** D is wrong: containers share network, not filesystems.
</details>

---

## 2.2 Deployments, rolling updates and rollbacks [`deployments-rollouts`]

### Key ideas
- Deployment → manages **ReplicaSets** → manage **Pods**. Each change to the Pod **template** creates a new ReplicaSet (a new *revision*). Scaling does not create a revision.
- Strategies:
  - `RollingUpdate` (default): `maxSurge` (extra Pods above desired, default 25%) and `maxUnavailable` (Pods allowed down, default 25%). `maxSurge: 0, maxUnavailable: 1` = replace one at a time with no extra capacity. `maxSurge: 1, maxUnavailable: 0` = zero-downtime with one extra Pod.
  - `Recreate`: kill all old Pods, then start new ones (downtime).
- `minReadySeconds`: how long a new Pod must be Ready before it counts as available. `progressDeadlineSeconds` (default 600): after this the rollout is marked failed (`ProgressDeadlineExceeded`), but it is **not** rolled back automatically.
- `revisionHistoryLimit` (default 10) old ReplicaSets are kept for rollback.
- `--record` is deprecated. Set the `kubernetes.io/change-cause` annotation to fill the CHANGE-CAUSE column.

### Commands

```bash
k create deploy web --image=nginx:1.26 --replicas=4
k set image deploy/web nginx=nginx:1.27
k annotate deploy/web kubernetes.io/change-cause="nginx 1.27" --overwrite
k rollout status deploy/web
k rollout history deploy/web
k rollout history deploy/web --revision=2        # shows the template of revision 2
k rollout undo deploy/web                        # back to the previous revision
k rollout undo deploy/web --to-revision=1
k rollout pause deploy/web ; k rollout resume deploy/web   # batch several changes into one rollout
k rollout restart deploy/web                     # new Pods, same spec (adds a restartedAt annotation)
k get rs -l app=web
```

```yaml
spec:
  replicas: 4
  strategy:
    type: RollingUpdate
    rollingUpdate: {maxSurge: 1, maxUnavailable: 0}
  minReadySeconds: 5
```

### Lab
1. Create `web` with `nginx:1.26`, 4 replicas, strategy `maxSurge: 1, maxUnavailable: 0`.
2. In a second terminal: `k get rs -w`. Update to `nginx:1.27` and watch the old RS scale down one by one.
3. Update to a broken image `nginx:doesnotexist`. Run `k rollout status` and see it hang. Check that the old Pods are still serving (thanks to `maxUnavailable: 0`).
4. `k rollout history`, then roll back to the 1.27 revision with `--to-revision`.
5. Switch the strategy to `Recreate`, update the image, and compare what `k get pods -w` shows.

### Pitfalls
- `rollout undo` creates a **new** revision number (the old one disappears from the list).
- Changing `replicas` or labels outside the template does not trigger a rollout.
- An exam task may say "record the change". Do it with the annotation, since `--record` may be removed.

### Check questions

**Q1.** Deployment has 10 replicas, `maxSurge: 25%`, `maxUnavailable: 25%`. During a rollout, what is the maximum total number of Pods?
- A. 10
- B. 12
- C. 13
- D. 15

<details><summary>▶ Answer</summary>

**C.** maxSurge rounds **up**: 25% of 10 = 2.5, which becomes 3, so the maximum is 13. maxUnavailable rounds **down**, to 2.
</details>

**Q2.** Which change creates a new revision?
- A. `k scale deploy/web --replicas=6`
- B. `k set image deploy/web nginx=nginx:1.27`
- C. `k label deploy/web team=a`
- D. `k annotate deploy/web owner=b`

<details><summary>▶ Answer</summary>

**B.** Only changes to `spec.template` create a revision.
</details>

**Q3.** A rollout to a bad image has hung. You need the previous working version immediately. Which command?
- A. `k rollout restart deploy/web`
- B. `k rollout undo deploy/web`
- C. `k delete rs --all`
- D. `k rollout pause deploy/web`

<details><summary>▶ Answer</summary>

**B.**
</details>

---

## 2.3 ConfigMaps and Secrets [`configmaps-secrets`]

### Key ideas
- **ConfigMap** = non-secret key/value or file data. **Secret** = the same, but values are base64-encoded in the API, can be encrypted at rest (if the apiserver is configured), and are stored in tmpfs on nodes. Base64 is **not** encryption.
- Secret types: `Opaque` (generic), `kubernetes.io/tls`, `kubernetes.io/dockerconfigjson` (image pull), `kubernetes.io/service-account-token`, `kubernetes.io/basic-auth`.
- Three ways to consume:
  1. Single env var: `env[].valueFrom.configMapKeyRef / secretKeyRef`
  2. All keys as env vars: `envFrom[].configMapRef / secretRef` (optional `prefix`)
  3. As files: a `configMap` / `secret` **volume** (each key becomes a file; `items` picks keys and paths; `defaultMode` sets permissions)
- Updates: **env vars never update** in a running container. **Mounted volumes do update** (after the kubelet sync, up to about a minute), **except when mounted with `subPath`**.
- `immutable: true` protects a ConfigMap/Secret from changes (delete and recreate to change it).

```bash
k create cm app-cfg --from-literal=MODE=prod --from-literal=LOG=debug
k create cm nginx-conf --from-file=default.conf            # key = file name
k create cm env-cfg --from-env-file=app.env
k create secret generic db --from-literal=user=admin --from-literal=password='p@ss'
k create secret docker-registry regcred --docker-server=reg.example.com --docker-username=u --docker-password=p
k get secret db -o jsonpath='{.data.password}' | base64 -d
```

```yaml
spec:
  containers:
    - name: app
      image: busybox:1.36
      command: ['sh', '-c', 'env; cat /etc/cfg/MODE; sleep 3600']
      env:
        - name: DB_PASSWORD
          valueFrom: {secretKeyRef: {name: db, key: password}}
      envFrom:
        - configMapRef: {name: app-cfg}
          prefix: APP_
      volumeMounts:
        - {name: cfg, mountPath: /etc/cfg, readOnly: true}
        - {name: nginx, mountPath: /etc/nginx/conf.d/default.conf, subPath: default.conf}
  volumes:
    - name: cfg
      configMap: {name: app-cfg}
    - name: nginx
      configMap: {name: nginx-conf}
  imagePullSecrets: [{name: regcred}]
```

### Lab
1. Create `app-cfg` and `db` as above, and a Pod that uses all three consumption methods. Verify with `k exec -- env` and `k exec -- cat`.
2. Change `MODE` in the ConfigMap (`k edit cm app-cfg`). After about a minute, compare the file in `/etc/cfg/MODE` (updated) with the env var `APP_MODE` (not updated).
3. Reference a key that does not exist. Read the Pod status (`CreateContainerConfigError`) and the event message.
4. Decode a Secret value with jsonpath + `base64 -d`.

### Pitfalls
- `echo 'x' | base64` adds a newline. Use `echo -n` when you hand-write `data:`, or use `stringData:` (plain text, encoded by the API).
- A missing ConfigMap/Secret blocks the Pod from starting unless the reference is marked `optional: true`.

### Check questions

**Q1.** A ConfigMap mounted as a volume and also exposed through `envFrom` is updated. What does the running container see?
- A. Both are updated
- B. The file updates after a short delay; the env vars do not
- C. The env vars update; the file does not
- D. Neither, until the Pod restarts

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q2.** A Pod is stuck in `CreateContainerConfigError`. What is the most likely cause?
- A. The image does not exist
- B. A referenced ConfigMap/Secret or key is missing
- C. The node is out of CPU
- D. The liveness probe fails

<details><summary>▶ Answer</summary>

**B.** A gives ImagePullBackOff, C gives Pending, D gives restarts and CrashLoopBackOff.
</details>

**Q3.** Which statement about Secrets is true?
- A. Secret data is encrypted by default in etcd
- B. `data` values are base64-encoded; `stringData` accepts plain text
- C. Secrets cannot be mounted as files
- D. Any Pod in any namespace can reference any Secret

<details><summary>▶ Answer</summary>

**B.** Encryption at rest needs an `EncryptionConfiguration`. Pods can reference only Secrets in their own namespace.
</details>

---

## 2.4 Self-healing primitives: probes, DaemonSet, StatefulSet, Job, CronJob, PDB [`self-healing-primitives`]

### Probes

| Probe | On failure | Use for |
|---|---|---|
| `startupProbe` | container restarted; **other probes are disabled until it succeeds** | slow-starting apps |
| `livenessProbe` | container restarted | deadlocks, hung processes |
| `readinessProbe` | Pod removed from Service endpoints (no restart) | not ready to take traffic |

Handlers: `httpGet` (path, port), `tcpSocket` (port), `exec` (command), `grpc` (port).
Timing fields: `initialDelaySeconds` (0), `periodSeconds` (10), `timeoutSeconds` (1), `failureThreshold` (3), `successThreshold` (1).

```yaml
containers:
  - name: web
    image: nginx
    startupProbe:   {httpGet: {path: /, port: 80}, failureThreshold: 30, periodSeconds: 2}
    livenessProbe:  {httpGet: {path: /, port: 80}, periodSeconds: 10}
    readinessProbe: {exec: {command: ['test', '-f', '/usr/share/nginx/html/index.html']}, periodSeconds: 5}
```

### Controllers: pick by need

| Controller | Guarantees | Notes |
|---|---|---|
| **Deployment** | N interchangeable replicas | stateless apps |
| **StatefulSet** | stable names `web-0, web-1`, stable DNS, **one PVC per Pod** (`volumeClaimTemplates`), ordered start/stop | needs a **headless Service** (`clusterIP: None`) named in `serviceName` |
| **DaemonSet** | one Pod per (matching) node | node agents: CNI, kube-proxy, log collectors. Needs tolerations to run on tainted control-plane nodes. |
| **Job** | run to completion | `completions`, `parallelism`, `backoffLimit` (default 6), `activeDeadlineSeconds`, `ttlSecondsAfterFinished`; `restartPolicy` must be `Never` or `OnFailure` |
| **CronJob** | Jobs on a schedule | `schedule`, `concurrencyPolicy` (Allow/Forbid/Replace), `successfulJobsHistoryLimit` (3), `failedJobsHistoryLimit` (1), `suspend`, `timeZone` |

```bash
# There is no `kubectl create daemonset`: generate a Deployment and edit it
k create deploy agent --image=busybox $do -- sh -c 'sleep infinity' > ds.yaml
# change kind: DaemonSet; delete replicas, strategy and status
k create job hello --image=busybox -- echo hi
k create cronjob tick --image=busybox --schedule="*/1 * * * *" -- date
k create job manual-run --from=cronjob/tick       # trigger a CronJob now
```

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata: {name: db}
spec:
  serviceName: db               # headless Service: kubectl create svc clusterip db --clusterip=None --tcp=5432
  replicas: 3
  selector: {matchLabels: {app: db}}
  template:
    metadata: {labels: {app: db}}
    spec:
      containers:
        - name: pg
          image: postgres:17
          env: [{name: POSTGRES_PASSWORD, value: x}]
          volumeMounts: [{name: data, mountPath: /var/lib/postgresql/data}]
  volumeClaimTemplates:
    - metadata: {name: data}
      spec: {accessModes: [ReadWriteOnce], resources: {requests: {storage: 1Gi}}}
```

**PodDisruptionBudget** (`policy/v1`) limits *voluntary* disruptions (drain, eviction): `minAvailable` or `maxUnavailable` plus a `selector`. `kubectl drain` waits or fails while a PDB would be violated.

```bash
k create pdb web-pdb --selector=app=web --min-available=2
```

### Lab
1. Deploy nginx with a readiness probe on `/ready` (which returns 404). Check `READY 0/1` and that `k get endpointslices -l kubernetes.io/service-name=<svc>` lists no ready endpoints. Create the file (`k exec -- sh -c 'echo ok > /usr/share/nginx/html/ready'`) and watch it become ready.
2. Add a liveness probe that `exec`s `cat /tmp/healthy`, with a container command that creates the file and deletes it after 30 s. Watch RESTARTS increase.
3. Build a DaemonSet from a Deployment skeleton. Check that it does not run on the control plane, then add a toleration for `node-role.kubernetes.io/control-plane:NoSchedule`.
4. Create the StatefulSet above. Delete `db-1` and confirm it comes back with the same name and the same PVC.
5. Create a Job with `completions: 5, parallelism: 2` and watch the Pods.
6. Create a PDB `min-available=3` for a 3-replica Deployment and try `k drain` on a node hosting one of its Pods (lab B, or a kind worker).

### Pitfalls
- A liveness probe that is too aggressive restarts a healthy but slow app. Use a startupProbe instead of a large initialDelaySeconds.
- Readiness failures never restart the container. If the app "never gets traffic", look at readiness.
- Deleting a StatefulSet does **not** delete its PVCs.

### Check questions

**Q1.** Container restarts every 30 seconds; logs show the app is healthy but takes 60 s to start. Best fix?
- A. Remove the readiness probe
- B. Add a startupProbe allowing enough time (for example `failureThreshold: 30, periodSeconds: 5`)
- C. Set `restartPolicy: Never`
- D. Increase replicas

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q2.** Which workload gives each replica its own PersistentVolumeClaim automatically?
- A. Deployment with a PVC volume
- B. DaemonSet
- C. StatefulSet with volumeClaimTemplates
- D. ReplicaSet

<details><summary>▶ Answer</summary>

**C.** A Deployment with one PVC shares that single claim across all replicas.
</details>

**Q3.** A CronJob must never run two Jobs at the same time; if one is still running, the next run is skipped. Which setting?
- A. `concurrencyPolicy: Replace`
- B. `concurrencyPolicy: Forbid`
- C. `parallelism: 1`
- D. `suspend: true`

<details><summary>▶ Answer</summary>

**B.** Replace kills the running Job and starts the new one.
</details>

**Q4.** A Pod fails its readiness probe. What happens?
- A. It is restarted
- B. It is removed from Service endpoints until the probe passes
- C. It is evicted to another node
- D. The Deployment rolls back

<details><summary>▶ Answer</summary>

**B.**
</details>

---

## 2.5 Requests/limits, LimitRange and ResourceQuota [`resources-limits`]

### Key ideas
- **requests**: what the scheduler reserves. A Pod fits on a node only if the sum of requests fits the node's *allocatable*.
- **limits**: the runtime cap. CPU over the limit is **throttled**. Memory over the limit is **OOMKilled** (exit code 137, reason `OOMKilled`).
- If you set only limits, requests default to the limits.
- **QoS class** (shown in `k describe pod`), which decides eviction order under node pressure:
  - **Guaranteed**: every container has CPU and memory requests equal to limits.
  - **Burstable**: at least one request or limit is set, but not Guaranteed.
  - **BestEffort**: nothing is set. Evicted first.
- **LimitRange** (per namespace): `default` (default limits), `defaultRequest`, `min`, `max`, `maxLimitRequestRatio`, per container or Pod. Applied **at admission to new Pods only**.
- **ResourceQuota** (per namespace): caps totals such as `pods`, `requests.cpu`, `limits.memory`, `persistentvolumeclaims`, `services.nodeports`, `count/deployments.apps`. When a quota covers `requests.cpu`, every new Pod **must** set a CPU request (or get one from a LimitRange), or it is rejected.
- CPU units: `1` = 1 vCPU = `1000m`. Memory: `Mi`/`Gi` (binary) vs `M`/`G` (decimal).

```yaml
apiVersion: v1
kind: LimitRange
metadata: {name: defaults, namespace: dev}
spec:
  limits:
    - type: Container
      defaultRequest: {cpu: 100m, memory: 128Mi}
      default:        {cpu: 500m, memory: 256Mi}
      max:            {cpu: "1",  memory: 1Gi}
---
apiVersion: v1
kind: ResourceQuota
metadata: {name: q, namespace: dev}
spec:
  hard: {pods: "10", requests.cpu: "2", requests.memory: 2Gi, limits.memory: 4Gi}
```

```bash
k create quota q --hard=pods=10,requests.cpu=2,requests.memory=2Gi -n dev
k describe quota q -n dev            # Used vs Hard
k describe limitrange defaults -n dev
k describe node w1 | sed -n '/Allocated resources/,/Events/p'
k get pod x -o jsonpath='{.status.qosClass}'
```

### Lab
1. In `dev`, create the LimitRange, then a Pod with no resources. Inspect its spec: the defaults were injected.
2. Create the quota with `pods: 2`, then a Deployment with 4 replicas. Read the ReplicaSet events (`k describe rs`): "exceeded quota".
3. Run a Pod with `limits.memory: 50Mi` and command `sh -c 'dd if=/dev/zero of=/dev/shm/x bs=1M count=100; sleep 3600'`, or use `polinux/stress`. Observe `OOMKilled`.
4. Create a Pod requesting more CPU than any node has. Read the scheduler event: `0/3 nodes are available: 3 Insufficient cpu`.
5. Create one Pod of each QoS class and verify with jsonpath.

### Pitfalls
- A quota error appears on the **ReplicaSet**, not on the Deployment or a Pod. Look at `k describe rs` or `k get events`.
- A LimitRange `max` that is lower than an existing Pod's limit does not evict it. It only blocks new Pods.

### Check questions

**Q1.** A container has `requests: {cpu: 200m, memory: 256Mi}` and `limits: {cpu: 200m, memory: 256Mi}`, and it is the only container. QoS class?
- A. BestEffort
- B. Burstable
- C. Guaranteed
- D. Critical

<details><summary>▶ Answer</summary>

**C.**
</details>

**Q2.** Namespace `dev` has a ResourceQuota on `requests.memory`. A Deployment's Pods have no resources set and there is no LimitRange. What happens?
- A. Pods are created with no requests
- B. Pod creation is rejected; the ReplicaSet shows "must specify requests.memory"
- C. Pods get the quota as their request
- D. The Deployment is rejected

<details><summary>▶ Answer</summary>

**B.** Adding a LimitRange with `defaultRequest` fixes it.
</details>

**Q3.** A container repeatedly exits with code 137 and reason `OOMKilled`. What should you change?
- A. Raise the CPU limit
- B. Raise the memory limit (or fix the leak)
- C. Add a readiness probe
- D. Change the restartPolicy

<details><summary>▶ Answer</summary>

**B.**
</details>

---

## 2.6 Workload autoscaling [`workload-autoscaling`]

### Key ideas
- **HorizontalPodAutoscaler** (`autoscaling/v2`) changes `replicas` of a Deployment/StatefulSet/ReplicaSet based on metrics.
  - Needs **metrics-server** for CPU and memory, and **requests** on the target containers (utilization = usage ÷ request).
  - Formula: `desired = ceil(current × currentMetric / targetMetric)`.
  - `behavior.scaleDown.stabilizationWindowSeconds` defaults to 300 s, which is why scale-down is slow.
- **VerticalPodAutoscaler**: not built in (separate add-on). It recommends or sets requests.
- **In-place Pod resize** (stable in recent versions): change a running container's CPU/memory without recreating the Pod, through the `resize` subresource; `resizePolicy` controls whether a restart is needed.
- Manual scaling is also "workload scaling": `k scale`.

```bash
k create deploy php --image=registry.k8s.io/hpa-example --port=80
k set resources deploy/php --requests=cpu=200m
k expose deploy php --port=80
k autoscale deploy php --min=1 --max=10 --cpu-percent=50
#   newer kubectl versions also accept --cpu=50% and may print a deprecation note for --cpu-percent; check `k autoscale -h`
k get hpa -w
# generate load:
k run load --image=busybox --restart=Never -- sh -c 'while true; do wget -qO- http://php; done'
# in-place resize (if enabled):
k patch pod mypod --subresource resize -p '{"spec":{"containers":[{"name":"app","resources":{"requests":{"cpu":"300m"}}}]}}'
```

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata: {name: php}
spec:
  scaleTargetRef: {apiVersion: apps/v1, kind: Deployment, name: php}
  minReplicas: 1
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target: {type: Utilization, averageUtilization: 50}
    - type: Resource
      resource:
        name: memory
        target: {type: AverageValue, averageValue: 200Mi}
  behavior:
    scaleDown: {stabilizationWindowSeconds: 60}
```

### Lab
1. Do the php-apache walkthrough above. Watch `TARGETS` go from `<unknown>` to a percentage, then the replicas increase.
2. Delete the load Pod and time how long scale-down takes. Then set `stabilizationWindowSeconds: 30` and repeat.
3. Remove the CPU request from the Deployment and run `k describe hpa`: "failed to get cpu utilization: missing request for cpu".
4. Write the HPA in YAML (from the docs page "HorizontalPodAutoscaler Walkthrough") with both a CPU and a memory metric.

### Pitfalls
- `TARGETS <unknown>`: metrics-server is missing or not ready, or the containers have no requests.
- The HPA fights a manual `k scale`: it wins at the next sync.

### Check questions

**Q1.** HPA target is 50% CPU. There are 4 replicas averaging 100%. How many replicas does it want?
- A. 4
- B. 6
- C. 8
- D. 10

<details><summary>▶ Answer</summary>

**C.** ceil(4 × 100 / 50) = 8, capped at maxReplicas.
</details>

**Q2.** `k get hpa` shows `TARGETS: <unknown>/50%`. Two likely causes are:
- A. metrics-server is not installed, or the Pods have no CPU requests
- B. The Deployment has too many replicas
- C. The HPA is in the wrong API version
- D. The Service is missing

<details><summary>▶ Answer</summary>

**A.**
</details>

---

## 2.7 Scheduling and placement [`scheduling-placement`]

### Key ideas

| Mechanism | Field | Effect |
|---|---|---|
| **nodeName** | `spec.nodeName` | Skips the scheduler completely; kubelet on that node runs it. Ignores taints for NoSchedule. |
| **nodeSelector** | `spec.nodeSelector: {disk: ssd}` | Node must have all these labels (hard). |
| **Node affinity** | `affinity.nodeAffinity` | `requiredDuringSchedulingIgnoredDuringExecution` (hard) or `preferred…` (soft, `weight` 1–100). Operators: In, NotIn, Exists, DoesNotExist, Gt, Lt. |
| **Pod affinity / anti-affinity** | `affinity.podAffinity / podAntiAffinity` | Place near or away from Pods with given labels, within a `topologyKey` (`kubernetes.io/hostname` = same node; `topology.kubernetes.io/zone` = same zone). |
| **Taints** (on nodes) + **tolerations** (on Pods) | `k taint node`, `spec.tolerations` | A taint repels Pods that don't tolerate it. Effects: `NoSchedule`, `PreferNoSchedule`, `NoExecute` (also evicts running Pods; `tolerationSeconds`). A toleration **allows** but does not **attract**; combine with affinity or nodeSelector to pin. |
| **Topology spread** | `topologySpreadConstraints` | `maxSkew`, `topologyKey`, `whenUnsatisfiable: DoNotSchedule/ScheduleAnyway`, `labelSelector`. Spreads replicas evenly. |
| **PriorityClass** | `priorityClassName` | Higher priority Pods are scheduled first and can **preempt** (evict) lower ones. |
| **cordon** | `k cordon node` | Marks node unschedulable (`node.kubernetes.io/unschedulable` taint); existing Pods stay. |

Control plane nodes on kubeadm carry `node-role.kubernetes.io/control-plane:NoSchedule`.

```yaml
spec:
  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:                 # terms are ORed
          - matchExpressions:              # expressions inside a term are ANDed
              - {key: disk, operator: In, values: [ssd]}
      preferredDuringSchedulingIgnoredDuringExecution:
        - weight: 80
          preference:
            matchExpressions: [{key: zone, operator: In, values: [a]}]
    podAntiAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        - labelSelector: {matchLabels: {app: web}}
          topologyKey: kubernetes.io/hostname      # at most one web Pod per node
  tolerations:
    - {key: dedicated, operator: Equal, value: gpu, effect: NoSchedule}
    - {key: node-role.kubernetes.io/control-plane, operator: Exists, effect: NoSchedule}
  topologySpreadConstraints:
    - maxSkew: 1
      topologyKey: kubernetes.io/hostname
      whenUnsatisfiable: DoNotSchedule
      labelSelector: {matchLabels: {app: web}}
```

```bash
k label node w1 disk=ssd
k taint node w2 dedicated=gpu:NoSchedule
k taint node w2 dedicated=gpu:NoSchedule-        # remove
k describe node w2 | grep -i taint
k get pods -o wide                               # NODE column
k create priorityclass critical --value=100000
```

### Lab
1. Label `cka-worker` with `disk=ssd`. Create a Deployment with required node affinity for `disk=ssd` and 3 replicas. Confirm all Pods land on that worker.
2. Taint `cka-worker2` with `dedicated=gpu:NoSchedule`. Create a Pod with the toleration only and see that it may still land elsewhere. Add a nodeSelector or affinity to pin it.
3. Apply `NoExecute` to a node running Pods without a matching toleration and watch them get evicted.
4. Create a 3-replica Deployment with required pod anti-affinity on hostname, on a cluster with 2 workers (no control-plane toleration). One Pod stays Pending; read the event.
5. Schedule a Pod on the control plane with a toleration + `nodeSelector: {node-role.kubernetes.io/control-plane: ""}`.

### Pitfalls
- A toleration alone does not put a Pod on the tainted node.
- `IgnoredDuringExecution`: changing a node's labels later does not evict Pods already running there.
- nodeSelector expects the label to exist with that exact value. Check with `k get nodes --show-labels`.

### Check questions

**Q1.** You want Pods to run **only** on nodes tainted `team=a:NoSchedule`. What is the minimum required?
- A. A toleration for `team=a:NoSchedule`
- B. A toleration plus node affinity or a nodeSelector for a label on those nodes
- C. nodeName
- D. Pod anti-affinity

<details><summary>▶ Answer</summary>

**B.** C works for one node but skips scheduling and does not scale.
</details>

**Q2.** Which taint effect evicts running Pods that do not tolerate it?
- A. NoSchedule
- B. PreferNoSchedule
- C. NoExecute
- D. Evict

<details><summary>▶ Answer</summary>

**C.**
</details>

**Q3.** A Pod has `nodeName: w1`. Node w1 has a taint `x=y:NoSchedule`, and the Pod has no toleration. Result?
- A. Pending forever
- B. Runs on w1, because nodeName bypasses the scheduler's taint check
- C. Scheduled to another node
- D. Rejected by the apiserver

<details><summary>▶ Answer</summary>

**B.** A `NoExecute` taint would still evict it.
</details>

**Q4.** Two `nodeSelectorTerms` are listed under required node affinity. How are they combined?
- A. AND
- B. OR
- C. Only the first is used
- D. Error

<details><summary>▶ Answer</summary>

**B.** Terms are ORed; expressions inside one term are ANDed.
</details>

---

## 2.8 Admission control and Pod Security Admission [`admission-control`]

### Key ideas
- Request flow in the apiserver: **authentication → authorization → mutating admission → schema validation → validating admission → etcd**.
- Admission plugins are enabled with `--enable-admission-plugins=` in `/etc/kubernetes/manifests/kube-apiserver.yaml` (kubeadm sets `NodeRestriction`; many plugins such as `NamespaceLifecycle`, `LimitRanger`, `ServiceAccount`, `DefaultStorageClass`, `ResourceQuota`, `PodSecurity`, the webhook plugins and `ValidatingAdmissionPolicy` are on by default). Disable with `--disable-admission-plugins=`.
- **Dynamic admission**: `MutatingWebhookConfiguration` / `ValidatingWebhookConfiguration` (call an external service) and **ValidatingAdmissionPolicy** (CEL rules in-cluster, no webhook).
- **Pod Security Admission (PSA)** enforces the Pod Security Standards per namespace via labels:
  - levels: `privileged` (anything), `baseline` (blocks hostNetwork, privileged, hostPath and similar), `restricted` (also requires runAsNonRoot, `allowPrivilegeEscalation: false`, `capabilities.drop: [ALL]`, `seccompProfile: RuntimeDefault`)
  - modes: `enforce` (reject), `audit` (log), `warn` (warning to the client)

```bash
k label ns dev pod-security.kubernetes.io/enforce=restricted pod-security.kubernetes.io/warn=restricted
k label ns dev pod-security.kubernetes.io/enforce=baseline --overwrite --dry-run=server   # shows which existing Pods would violate
```

A Pod that passes `restricted`:

```yaml
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 1000
    seccompProfile: {type: RuntimeDefault}
  containers:
    - name: app
      image: busybox:1.36
      command: ['sleep', '3600']
      securityContext:
        allowPrivilegeEscalation: false
        capabilities: {drop: [ALL]}
```

ValidatingAdmissionPolicy example (deny Deployments with more than 5 replicas):

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicy
metadata: {name: max-replicas}
spec:
  failurePolicy: Fail
  matchConstraints:
    resourceRules:
      - {apiGroups: [apps], apiVersions: [v1], operations: [CREATE, UPDATE], resources: [deployments]}
  validations:
    - expression: "object.spec.replicas <= 5"
      message: "replicas must be <= 5"
---
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicyBinding
metadata: {name: max-replicas}
spec:
  policyName: max-replicas
  validationActions: [Deny]
```

### Lab
1. Label a namespace `enforce=restricted`. Try `k run nginx --image=nginx` there and read the rejection reasons. Then create the compliant Pod above.
2. Switch the namespace to `enforce=baseline` with `warn=restricted`, create nginx, and read the warning.
3. On lab B, add `--enable-admission-plugins=NodeRestriction,AlwaysPullImages` to the apiserver manifest. Wait for the apiserver to restart (`watch crictl ps`) and confirm new Pods get `imagePullPolicy: Always`. Revert.
4. Apply the ValidatingAdmissionPolicy and test with a 6-replica Deployment.

### Pitfalls
- A typo in the apiserver manifest takes the API down. Back up the file first (`cp kube-apiserver.yaml /root/`), **outside** the manifests directory, otherwise kubelet runs the copy too.
- PSA checks Pods, not Deployments. A Deployment is accepted, but its ReplicaSet cannot create Pods. Look at the RS events (or the `warn` message).

### Check questions

**Q1.** In which order does the apiserver process a request?
- A. admission → authentication → authorization
- B. authentication → authorization → admission
- C. authorization → authentication → admission
- D. admission → authorization → authentication

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q2.** A Deployment in a namespace labelled `pod-security.kubernetes.io/enforce=restricted` shows 0/3 ready and no Pods exist. Where do you find the reason?
- A. `k logs deploy/x`
- B. `k describe rs` (events: "violates PodSecurity restricted…")
- C. `journalctl -u kubelet`
- D. `k describe node`

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q3.** How do you enable an extra admission plugin on a kubeadm cluster?
- A. `kubectl enable admission-plugin X`
- B. Edit `--enable-admission-plugins` in `/etc/kubernetes/manifests/kube-apiserver.yaml`
- C. Edit the kubelet config
- D. Create a ConfigMap in kube-system

<details><summary>▶ Answer</summary>

**B.**
</details>

---

## Chapter 2: exam-style tasks

**T2.1 (5 min)** Create Deployment `frontend` in namespace `shop` (image `nginx:1.26`, 3 replicas). Update it to `nginx:1.27` with a rolling update that never has fewer than 3 available Pods. Record the change cause `upgrade to 1.27`. Then roll back to revision 1.

<details><summary>▶ Solution</summary>

```bash
k create ns shop
k create deploy frontend -n shop --image=nginx:1.26 --replicas=3
k patch deploy frontend -n shop -p '{"spec":{"strategy":{"rollingUpdate":{"maxUnavailable":0,"maxSurge":1}}}}'
k set image deploy/frontend -n shop nginx=nginx:1.27
k annotate deploy/frontend -n shop kubernetes.io/change-cause="upgrade to 1.27" --overwrite
k rollout status deploy/frontend -n shop
k rollout undo deploy/frontend -n shop --to-revision=1
k rollout history deploy/frontend -n shop
```
</details>

**T2.2 (6 min)** Create Pod `logger` with a main container `app` (busybox, appends the date to `/var/log/app.log` every 5 s) and a **native sidecar** `tail` that prints that file to stdout. Verify with `k logs logger -c tail`.

<details><summary>▶ Solution</summary>

Use the Pod from section 2.1, minus `wait-db`. The sidecar goes in `initContainers` with `restartPolicy: Always`, and both containers mount an `emptyDir` at `/var/log`.
</details>

**T2.3 (5 min)** Namespace `limits`: make every container without resources get request `cpu: 50m, memory: 64Mi` and limit `memory: 128Mi`. Limit the namespace to 5 Pods. Prove it with a 6-replica Deployment.

<details><summary>▶ Solution</summary>

```yaml
apiVersion: v1
kind: LimitRange
metadata: {name: lr, namespace: limits}
spec:
  limits:
    - type: Container
      defaultRequest: {cpu: 50m, memory: 64Mi}
      default: {memory: 128Mi}
```
```bash
k create quota pods --hard=pods=5 -n limits
k create deploy six -n limits --image=nginx --replicas=6
k get pods -n limits        # 5 Pods
k describe rs -n limits | grep -i quota
```
</details>

**T2.4 (5 min)** Deployment `api` (2 replicas) must run only on nodes labelled `tier=backend`, and never two replicas on the same node. Label `cka-worker` and `cka-worker2` accordingly.

<details><summary>▶ Solution</summary>

```bash
k label node cka-worker cka-worker2 tier=backend
k create deploy api --image=nginx --replicas=2 $do > api.yaml
```
Add under `template.spec`:
```yaml
affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms: [{matchExpressions: [{key: tier, operator: In, values: [backend]}]}]
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector: {matchLabels: {app: api}}
        topologyKey: kubernetes.io/hostname
```
</details>

**T2.5 (4 min)** Create an HPA for Deployment `api` that keeps average CPU at 60%, with 2–6 replicas. Make sure the Deployment can be autoscaled.

<details><summary>▶ Solution</summary>

```bash
k set resources deploy/api --requests=cpu=100m
k autoscale deploy api --min=2 --max=6 --cpu-percent=60
k get hpa api
```
</details>

**T2.6 (4 min)** Create CronJob `backup` that runs `echo backup done` every 15 minutes, keeps 2 successful and 1 failed Job, and never runs concurrently. Trigger it once manually as Job `backup-now`.

<details><summary>▶ Solution</summary>

```bash
k create cronjob backup --image=busybox --schedule="*/15 * * * *" $do -- sh -c 'echo backup done' > cj.yaml
# under spec: concurrencyPolicy: Forbid, successfulJobsHistoryLimit: 2, failedJobsHistoryLimit: 1
k apply -f cj.yaml
k create job backup-now --from=cronjob/backup
k logs job/backup-now
```
</details>
