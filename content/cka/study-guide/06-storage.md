# 6. Storage (10%)

Official competencies: storage classes and dynamic volume provisioning · volume types, access modes and reclaim policies ·
persistent volumes and persistent volume claims.

Storage is the smallest domain, but its tasks are quick points if you can write a PV/PVC/Pod trio from memory
(or find "Configure a Pod to Use a PersistentVolume for Storage" in the docs in 20 seconds).

---

## 6.1 Volume types [`volumes-basics`]

### Key ideas
A volume is declared once in `spec.volumes` and mounted per container via `volumeMounts` (`mountPath`, optional `subPath`, `readOnly`).

| Type | Lifetime | Use |
|---|---|---|
| `emptyDir` | the Pod (survives container restarts, deleted with the Pod) | scratch space, sharing between containers; `medium: Memory` = tmpfs, `sizeLimit` |
| `hostPath` | the node | node agents reading `/var/log` and similar; ties the Pod to a node; security risk (blocked by PSA baseline) |
| `configMap`, `secret`, `downwardAPI` | API-backed | inject config, credentials and Pod metadata as files |
| `projected` | API-backed | combine several of the above (and SA tokens) into one directory |
| `persistentVolumeClaim` | independent of the Pod | real persistent storage (6.2) |
| `ephemeral` (generic ephemeral volume) | the Pod, but provisioned through a StorageClass | per-Pod scratch on real storage |
| `csi` / `nfs` / `iscsi` … | external | usually used through PVs |

```yaml
spec:
  containers:
    - name: app
      image: busybox:1.36
      command: ['sh', '-c', 'echo $(date) > /cache/t; cat /etc/podinfo/labels; sleep 3600']
      volumeMounts:
        - {name: cache, mountPath: /cache}
        - {name: podinfo, mountPath: /etc/podinfo}
        - {name: node-logs, mountPath: /host-logs, readOnly: true}
  volumes:
    - name: cache
      emptyDir: {sizeLimit: 100Mi}
    - name: podinfo
      downwardAPI:
        items: [{path: labels, fieldRef: {fieldPath: metadata.labels}}]
    - name: node-logs
      hostPath: {path: /var/log, type: Directory}
```

### Lab
1. Create the Pod above. Kill the container (`k exec -- kill 1`) and confirm `/cache/t` survives the restart. Delete the Pod and confirm the data is gone.
2. Write a `projected` volume that combines a ConfigMap and a Secret into `/etc/app`.
3. Mount the same `emptyDir` in two containers, writing from one and reading from the other.

### Check questions

**Q1.** Which volume keeps data across container restarts but loses it when the Pod is deleted?
- A. hostPath
- B. emptyDir
- C. persistentVolumeClaim
- D. nfs

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q2.** Why is hostPath discouraged for application data?
- A. It is read-only
- B. Data is tied to one node, so a rescheduled Pod loses it; it also exposes the node filesystem
- C. It needs a CSI driver
- D. It only works on the control plane

<details><summary>▶ Answer</summary>

**B.**
</details>

---

## 6.2 PersistentVolumes, PersistentVolumeClaims, access modes and reclaim policies [`pv-pvc`]

### Key ideas
- **PV** (cluster-scoped) = a piece of storage, created by an admin (static) or by a provisioner (dynamic). **PVC** (namespaced) = a request for storage. The control plane **binds** a PVC to one PV, one-to-one.
- A PVC binds to a PV when all of these match:
  - **storageClassName** is equal (`""` means "no class", which is different from leaving the field out, which means "use the default class")
  - **accessModes**: the PV offers what the PVC asks for
  - **capacity** ≥ request
  - **volumeMode** (Filesystem/Block) and any `selector` / `volumeName`
- **Access modes**:
  - `ReadWriteOnce` (RWO): read-write by **one node** (several Pods on that node can share it)
  - `ReadOnlyMany` (ROX): read-only by many nodes
  - `ReadWriteMany` (RWX): read-write by many nodes (NFS, CephFS…)
  - `ReadWriteOncePod` (RWOP): read-write by **one Pod** (CSI only)
- **Reclaim policy** (what happens to the PV when its PVC is deleted):
  - `Retain`: PV becomes `Released`; data is kept; an admin must clean it up (delete the PV, or clear `spec.claimRef` to make it `Available` again).
  - `Delete`: PV and backing storage are deleted (the default for dynamically provisioned PVs).
  - `Recycle`: deprecated.
- PV phases: `Available` → `Bound` → `Released` (→ `Failed`). PVC phases: `Pending` → `Bound`.
- A PVC in use by a Pod is protected (`kubernetes.io/pvc-protection` finalizer); deleting it waits until the Pod is gone.

### The trio (memorise this)

```yaml
apiVersion: v1
kind: PersistentVolume
metadata: {name: pv-data}
spec:
  capacity: {storage: 1Gi}
  accessModes: [ReadWriteOnce]
  persistentVolumeReclaimPolicy: Retain
  storageClassName: manual
  hostPath: {path: /mnt/data}           # lab only; real clusters use nfs/csi
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata: {name: pvc-data, namespace: default}
spec:
  accessModes: [ReadWriteOnce]
  storageClassName: manual
  resources: {requests: {storage: 500Mi}}
---
apiVersion: v1
kind: Pod
metadata: {name: web}
spec:
  containers:
    - name: nginx
      image: nginx
      volumeMounts: [{name: data, mountPath: /usr/share/nginx/html}]
  volumes:
    - name: data
      persistentVolumeClaim: {claimName: pvc-data}
```

```bash
k get pv,pvc
k describe pvc pvc-data          # Events explain why it is Pending
k get pv pv-data -o jsonpath='{.spec.claimRef.name}'
k patch pv pv-data -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}'
k patch pv pv-data --type=json -p='[{"op":"remove","path":"/spec/claimRef"}]'   # Released → Available
```

### Lab
1. Apply the trio. `PVC Bound` to `pv-data` with **capacity 1Gi** (the whole PV, not 500Mi). Write a file through the Pod.
2. Delete the Pod and the PVC. PV is `Released` and the data is still on the node under `/mnt/data`. Recreate the PVC: it stays **Pending**, because a Released PV is not rebound. Remove the `claimRef` and watch it bind.
3. Create a PVC asking for `ReadWriteMany` against a PV that offers only RWO. Read the Pending reason.
4. Create a PVC that leaves out `storageClassName` on kind and see it bind to a **dynamic** PV of the default class instead of your manual one.
5. Change the reclaim policy of a dynamic PV to `Retain` before deleting its PVC, and confirm the PV survives.

### Pitfalls
- Omitting `storageClassName` in a PVC when the task wants it bound to a static PV: the default StorageClass grabs it. Set the class (or `""`) explicitly.
- RWO does not mean "one Pod". Two Pods on the same node can mount it; use RWOP for one Pod.
- The Pod stays `Pending` with "persistentvolumeclaim not found" or "unbound immediate PersistentVolumeClaims". Fix the PVC first.

### Check questions

**Q1.** A PVC requests 2Gi RWO with `storageClassName: manual`. PVs: A (1Gi, RWO, manual), B (5Gi, RWX, manual), C (3Gi, RWO, fast), D (5Gi, RWO, manual). Which binds?
- A. A
- B. B
- C. C
- D. D

<details><summary>▶ Answer</summary>

**D.** A is too small, C has the wrong class, and B does not offer RWO.
</details>

**Q2.** A PV with `Retain` had its PVC deleted. A new PVC with identical specs stays Pending. Why?
- A. Retain PVs can never be reused
- B. The PV is `Released` with a stale `claimRef`; an admin must clear it (or recreate the PV)
- C. The new PVC needs a different name
- D. The StorageClass is missing

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q3.** Which access mode guarantees that only a single Pod in the whole cluster can write to the volume?
- A. ReadWriteOnce
- B. ReadWriteOncePod
- C. ReadOnlyMany
- D. ReadWriteMany

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q4.** What is the default reclaim policy for a dynamically provisioned PV (unless the StorageClass says otherwise)?
- A. Retain
- B. Recycle
- C. Delete
- D. Archive

<details><summary>▶ Answer</summary>

**C.**
</details>

---

## 6.3 StorageClasses and dynamic provisioning [`storageclass-dynamic`]

### Key ideas
- A **StorageClass** describes a "kind" of storage: `provisioner` (a CSI driver name such as `ebs.csi.aws.com`, or `rancher.io/local-path`), `parameters` (driver-specific), `reclaimPolicy` (default Delete), `volumeBindingMode`, `allowVolumeExpansion`, `mountOptions`.
- **Dynamic provisioning**: a PVC naming a StorageClass (or none, when a default exists) makes the provisioner create a PV automatically.
- **Default StorageClass**: annotation `storageclass.kubernetes.io/is-default-class: "true"`. PVCs without `storageClassName` get it (through the DefaultStorageClass admission plugin).
- `volumeBindingMode`:
  - `Immediate`: provision and bind as soon as the PVC is created (may pick a zone or node where the Pod cannot run).
  - `WaitForFirstConsumer`: wait until a Pod using the PVC is scheduled, then provision in the right topology. The PVC stays **Pending until a Pod uses it**, which is normal.
- `allowVolumeExpansion: true` lets you **grow** a PVC by editing `spec.resources.requests.storage` (never shrink).
- `kubernetes.io/no-provisioner` = a class for static local PVs, used with WaitForFirstConsumer.

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: fast
  annotations:
    storageclass.kubernetes.io/is-default-class: "false"
provisioner: rancher.io/local-path          # on kind; in the exam use whatever provisioner exists
reclaimPolicy: Retain
volumeBindingMode: WaitForFirstConsumer
allowVolumeExpansion: true
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata: {name: fast-claim}
spec:
  storageClassName: fast
  accessModes: [ReadWriteOnce]
  resources: {requests: {storage: 1Gi}}
```

```bash
k get sc                                     # (default) marker, PROVISIONER, RECLAIMPOLICY, VOLUMEBINDINGMODE
k patch sc standard -p '{"metadata":{"annotations":{"storageclass.kubernetes.io/is-default-class":"false"}}}'
k patch sc fast     -p '{"metadata":{"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
k patch pvc fast-claim -p '{"spec":{"resources":{"requests":{"storage":"2Gi"}}}}'   # expand
k get pv -o custom-columns=NAME:.metadata.name,CLAIM:.spec.claimRef.name,SC:.spec.storageClassName,POLICY:.spec.persistentVolumeReclaimPolicy
```

### Lab
1. On kind, `k get sc` shows `standard (default)` with `rancher.io/local-path` and `WaitForFirstConsumer`. Create a PVC and see it Pending. Create a Pod using it and see the PVC become Bound and a PV appear.
2. Create the `fast` class above. Make it the default and remove the default flag from `standard` (only one class should be default).
3. Create a PVC on `fast`, use it in a Pod, delete both, and confirm the PV is kept (Retain).
4. Try expanding a PVC on a class **without** `allowVolumeExpansion` and read the error. (local-path does not support expansion even with the flag; the task in the exam will use a provisioner that does, so focus on the commands.)
5. Create a StatefulSet with `volumeClaimTemplates` using `storageClassName: fast` and scale it to 3. Count the PVCs.

### Pitfalls
- Two default StorageClasses: new PVCs without a class get the most recently created default. Avoid it; there should be exactly one.
- A PVC Pending with `WaitForFirstConsumer` is not an error.
- Changing a StorageClass's parameters does not affect PVs that already exist (most StorageClass fields are immutable; delete and recreate the class).

### Check questions

**Q1.** A PVC with a `WaitForFirstConsumer` class has been Pending for 10 minutes. No Pod references it. What is going on?
- A. The provisioner is broken
- B. Normal: provisioning waits until a Pod using the claim is scheduled
- C. The class has no reclaim policy
- D. The PVC requests too much

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q2.** How do you make StorageClass `fast` the default?
- A. `k label sc fast default=true`
- B. Annotate it `storageclass.kubernetes.io/is-default-class: "true"` (and remove the annotation from the old default)
- C. Set `spec.default: true`
- D. Name it `default`

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q3.** A PVC must grow from 1Gi to 5Gi. Requirements?
- A. Delete and recreate it
- B. The StorageClass has `allowVolumeExpansion: true` (and the driver supports it); then edit the PVC's request
- C. Edit the PV's capacity
- D. Not possible

<details><summary>▶ Answer</summary>

**B.**
</details>

---

## Chapter 6: exam-style tasks

**T6.1 (5 min)** Create PV `log-pv` (2Gi, RWX, reclaim Retain, hostPath `/srv/logs`, class `logs`). Create PVC `log-pvc` in namespace `ops` requesting 1Gi RWX with class `logs`. Mount it in Pod `logger` (busybox, `sleep 3600`) at `/logs`. Confirm the PVC is Bound.

<details><summary>▶ Solution</summary>

Use the trio from 6.2, changing names, size, access mode (`ReadWriteMany`), `storageClassName: logs`, the hostPath and the namespace. Then `k get pvc -n ops` should show Bound to `log-pv`.
</details>

**T6.2 (4 min)** Create StorageClass `retain-local` using the cluster's existing provisioner, with reclaim policy Retain, `WaitForFirstConsumer`, and volume expansion allowed. Do not make it the default.

<details><summary>▶ Solution</summary>

```bash
k get sc -o jsonpath='{.items[0].provisioner}'    # reuse this provisioner
```
Then apply the `fast` StorageClass from 6.3 with name `retain-local` and that provisioner, without the default annotation.
</details>

**T6.3 (3 min)** PVC `data` in namespace `db` is bound to a dynamically provisioned PV. Make sure the data survives if someone deletes the PVC.

<details><summary>▶ Solution</summary>

```bash
PV=$(k get pvc data -n db -o jsonpath='{.spec.volumeName}')
k patch pv $PV -p '{"spec":{"persistentVolumeReclaimPolicy":"Retain"}}'
k get pv $PV
```
</details>

**T6.4 (3 min)** Write the names of all PVs sorted by capacity to `/tmp/pv-sorted.txt`.

<details><summary>▶ Solution</summary>

```bash
k get pv --sort-by=.spec.capacity.storage -o name > /tmp/pv-sorted.txt
```
(If the task wants the table, drop `-o name`.)
</details>
