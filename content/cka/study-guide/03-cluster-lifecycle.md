# 3. Cluster Lifecycle with kubeadm (part of Cluster Architecture, 25%)

Official competencies: prepare underlying infrastructure · create and manage clusters using kubeadm ·
manage the lifecycle of clusters · implement and configure a highly-available control plane.

Use **lab B** (VMs) for this whole chapter. These tasks are long (often 10–15 min each), but they are
mechanical: if you have done them three times, you can follow the docs page step by step in the exam.

---

## 3.1 Prepare the infrastructure [`infra-prep`]

### Key ideas
Every node (control plane and worker) needs:
1. A supported Linux, unique hostname/MAC/`product_uuid`, full network connectivity between nodes, required ports open (6443 apiserver, 2379–2380 etcd, 10250 kubelet, 10257/10259 controller-manager/scheduler, 30000–32767 NodePorts).
2. **Swap off** (kubeadm preflight fails on swap unless the kubelet is configured for it).
3. Kernel modules `overlay` and `br_netfilter`, and `net.ipv4.ip_forward = 1`.
4. A **container runtime** with a CRI socket. containerd is the usual choice; its cgroup driver must match kubelet's (**systemd**).
5. `kubeadm`, `kubelet` and `kubectl` packages of the **same minor version**, held so apt does not upgrade them by accident.

### Commands (run on every node; this is the docs page "Installing kubeadm" in condensed form)

```bash
# 1. swap off (now and after reboot)
sudo swapoff -a
sudo sed -i '/ swap / s/^/#/' /etc/fstab

# 2. kernel modules + sysctl
cat <<EOF | sudo tee /etc/modules-load.d/k8s.conf
overlay
br_netfilter
EOF
sudo modprobe overlay && sudo modprobe br_netfilter
cat <<EOF | sudo tee /etc/sysctl.d/k8s.conf
net.ipv4.ip_forward = 1
net.bridge.bridge-nf-call-iptables = 1
net.bridge.bridge-nf-call-ip6tables = 1
EOF
sudo sysctl --system

# 3. containerd with the systemd cgroup driver
sudo apt-get update && sudo apt-get install -y containerd
sudo mkdir -p /etc/containerd
containerd config default | sudo tee /etc/containerd/config.toml >/dev/null
sudo sed -i 's/SystemdCgroup = false/SystemdCgroup = true/' /etc/containerd/config.toml
grep -n SystemdCgroup /etc/containerd/config.toml       # verify it is true (the section name differs between containerd 1.7 and 2.x)
sudo systemctl restart containerd && sudo systemctl enable containerd

# 4. kubeadm, kubelet, kubectl from pkgs.k8s.io (one repo per minor version)
sudo apt-get install -y apt-transport-https ca-certificates curl gpg
sudo mkdir -p -m 755 /etc/apt/keyrings
curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.37/deb/Release.key | sudo gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.37/deb/ /' \
  | sudo tee /etc/apt/sources.list.d/kubernetes.list
sudo apt-get update && sudo apt-get install -y kubelet kubeadm kubectl
sudo apt-mark hold kubelet kubeadm kubectl
sudo systemctl enable --now kubelet       # it restarts in a loop until kubeadm init/join; that is normal
```

`crictl` needs to know the socket: `sudo crictl config --set runtime-endpoint=unix:///run/containerd/containerd.sock` (or it is set in `/etc/crictl.yaml`).

### Lab
1. Run the block above on three fresh VMs (use v1.36 in the repo URL if you want to practise the upgrade later).
2. On one VM, deliberately leave swap on and run `sudo kubeadm init`. Read the preflight error. Then `swapoff -a`.
3. On one VM, leave `ip_forward = 0` and read the preflight error.
4. Check the versions with `kubeadm version -o short`, `kubelet --version` and `kubectl version --client`.

### Pitfalls
- Cgroup driver mismatch (containerd `cgroupfs`, kubelet `systemd`): control-plane Pods restart randomly after init. Check `SystemdCgroup = true`.
- Forgetting `apt-mark hold`: a later `apt upgrade` silently bumps kubelet past the control plane.

### Check questions

**Q1.** kubeadm preflight reports `[ERROR FileContent--proc-sys-net-ipv4-ip_forward]: contents are not set to 1`. Fix?
- A. `modprobe br_netfilter`
- B. Set `net.ipv4.ip_forward = 1` in `/etc/sysctl.d/` and run `sysctl --system`
- C. Disable swap
- D. Restart containerd

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q2.** Why run `apt-mark hold kubelet kubeadm kubectl`?
- A. Required for kubelet to start
- B. To stop routine apt upgrades from changing cluster component versions outside a planned upgrade
- C. To lock the container runtime version
- D. To enable automatic upgrades

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q3.** Which cgroup driver should containerd use on a kubeadm cluster with systemd?
- A. cgroupfs
- B. systemd
- C. none
- D. Either; they don't need to match kubelet

<details><summary>▶ Answer</summary>

**B.**
</details>

---

## 3.2 Create a cluster with kubeadm [`kubeadm-install`]

### Key ideas
`kubeadm init` on the first control plane does the following:
- generates certificates in `/etc/kubernetes/pki`
- writes kubeconfigs (`admin.conf`, `super-admin.conf`, `controller-manager.conf`, `scheduler.conf`, `kubelet.conf`) in `/etc/kubernetes/`
- writes static Pod manifests for etcd, apiserver, controller-manager and scheduler to `/etc/kubernetes/manifests`
- waits for the control plane, then installs CoreDNS and kube-proxy
- prints a `kubeadm join` command (with a bootstrap token, TTL 24 h)

CoreDNS stays **Pending until you install a CNI**, and nodes stay `NotReady` too.

### Commands

```bash
# control plane
sudo kubeadm init \
  --pod-network-cidr=192.168.0.0/16 \
  --apiserver-advertise-address=<cp1-ip> \
  --kubernetes-version=v1.37.0            # optional; defaults to the kubeadm version's stable
mkdir -p $HOME/.kube && sudo cp /etc/kubernetes/admin.conf $HOME/.kube/config && sudo chown $(id -u):$(id -g) $HOME/.kube/config

# CNI (example: Calico operator; the pod CIDR must match --pod-network-cidr)
kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.30.0/manifests/tigera-operator.yaml
kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.30.0/manifests/custom-resources.yaml

# print a fresh join command (tokens expire after 24 h)
kubeadm token create --print-join-command
kubeadm token list

# each worker
sudo kubeadm join <cp1-ip>:6443 --token <t> --discovery-token-ca-cert-hash sha256:<hash>

kubectl get nodes -o wide
```

**Config-file style** (useful when a task gives you a file or asks for specific settings):

```bash
kubeadm config print init-defaults > kubeadm.yaml      # ClusterConfiguration + InitConfiguration (kubeadm.k8s.io/v1beta4)
# edit networking.podSubnet, kubernetesVersion, controlPlaneEndpoint, localAPIEndpoint.advertiseAddress ...
sudo kubeadm init --config kubeadm.yaml
sudo kubeadm reset -f    # tear down a node (then clean /etc/cni/net.d and iptables if you want a clean slate)
```

### Lab
1. Init `cp1`, set up kubeconfig, check `kubectl get nodes` (NotReady) and `kubectl get pods -n kube-system` (CoreDNS Pending).
2. Install Calico, wait for Ready.
3. Join `w1` with the printed command. Join `w2` with a **new** token from `kubeadm token create --print-join-command`.
4. Look through `/etc/kubernetes/` and `/etc/kubernetes/pki` on cp1 and name every file.
5. `kubeadm reset -f` on `w2` and rejoin it.
6. **Snapshot all VMs now** so you can return to a clean cluster.

### Pitfalls
- Pod CIDR overlapping your VM network: Pods get IPs that collide with node IPs. Pick a CIDR that does not overlap.
- Running kubectl as root without `KUBECONFIG=/etc/kubernetes/admin.conf`.
- Worker join fails with "token not found": the token expired. Create a new one.

### Check questions

**Q1.** After `kubeadm init`, all nodes are `NotReady` and CoreDNS is Pending. What is the most likely missing step?
- A. Join a worker
- B. Install a CNI plugin
- C. Restart kubelet
- D. Run `kubeadm upgrade`

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q2.** The join command from yesterday fails with an authentication/token error. Fix?
- A. `kubeadm reset` on the control plane
- B. `kubeadm token create --print-join-command` on the control plane and use the new command
- C. Copy `admin.conf` to the worker
- D. Re-run `kubeadm init`

<details><summary>▶ Answer</summary>

**B.** Bootstrap tokens expire after 24 h by default.
</details>

**Q3.** Where does kubeadm put the static Pod manifests for the control plane?
- A. `/var/lib/kubelet/manifests`
- B. `/etc/kubernetes/manifests`
- C. `/etc/kubernetes/pki`
- D. `/opt/kubernetes`

<details><summary>▶ Answer</summary>

**B.**
</details>

---

## 3.3 Upgrade the cluster, drain and cordon [`cluster-upgrade`]

### Key ideas
- Upgrade **one minor version at a time** (1.36 → 1.37, never 1.35 → 1.37). Patch versions within a minor can be skipped.
- Order: **first control plane → other control planes → workers**.
- **Version skew** rules: the kubelet may be up to 3 minor versions **older** than the apiserver, never newer. kube-controller-manager and the scheduler may be up to 1 minor older. kubectl may be ±1 from the apiserver.
- `kubectl cordon` = mark unschedulable. `kubectl drain` = cordon + evict all Pods (respects PDBs). `kubectl uncordon` = schedulable again.
- `kubeadm upgrade` also **renews control-plane certificates**.

### Procedure (docs page "Upgrading kubeadm clusters")

```bash
### FIRST CONTROL PLANE (cp1) ###
# a) point apt at the new minor repo
sudo sed -i 's/v1.36/v1.37/' /etc/apt/sources.list.d/kubernetes.list
sudo apt-get update
apt-cache madison kubeadm | head                       # find the exact version, for example 1.37.1-1.1

# b) upgrade kubeadm
sudo apt-mark unhold kubeadm && sudo apt-get install -y kubeadm='1.37.1-*' && sudo apt-mark hold kubeadm
kubeadm version -o short

# c) plan + apply
sudo kubeadm upgrade plan
sudo kubeadm upgrade apply v1.37.1                    # on additional control planes: sudo kubeadm upgrade node

# d) drain, upgrade kubelet + kubectl, restart, uncordon
kubectl drain cp1 --ignore-daemonsets
sudo apt-mark unhold kubelet kubectl && sudo apt-get install -y kubelet='1.37.1-*' kubectl='1.37.1-*' && sudo apt-mark hold kubelet kubectl
sudo systemctl daemon-reload && sudo systemctl restart kubelet
kubectl uncordon cp1

### EACH WORKER (w1) ###
# on w1: repo + kubeadm as in a) and b), then
sudo kubeadm upgrade node                              # updates the local kubelet config
# from the control plane:
kubectl drain w1 --ignore-daemonsets --delete-emptydir-data
# on w1:
sudo apt-mark unhold kubelet kubectl && sudo apt-get install -y kubelet='1.37.1-*' kubectl='1.37.1-*' && sudo apt-mark hold kubelet kubectl
sudo systemctl daemon-reload && sudo systemctl restart kubelet
# from the control plane:
kubectl uncordon w1
kubectl get nodes          # VERSION column shows each node's kubelet version
```

Drain flags:
- `--ignore-daemonsets`: DaemonSet Pods cannot be evicted (they would come straight back), so skip them.
- `--delete-emptydir-data`: allow eviction of Pods using emptyDir (that data is lost).
- `--force`: also delete bare Pods not managed by a controller (they are **not** recreated).

### Lab
1. Build lab B on v1.36. Run the full upgrade to v1.37 on cp1 and both workers. Time yourself; aim for under 15 minutes for cp + 1 worker.
2. Create a bare Pod (`k run bare --image=nginx`) on a worker and try `kubectl drain` without `--force`. Read the error, then decide.
3. Create a PDB with `minAvailable` equal to the replica count and watch drain retry.
4. Compare `kubectl version` (server) and `kubectl get nodes` (kubelet versions) during the upgrade.

### Pitfalls
- In the exam, the apt repo is often already set for the target version. Check `apt-cache madison kubeadm` before changing the repo.
- Running `kubeadm upgrade apply` on a worker. Workers use `kubeadm upgrade node`.
- Forgetting `systemctl daemon-reload && systemctl restart kubelet`: the node still reports the old version.
- Upgrading the worker's packages while you are on the control plane. Check `hostname` before you type.

### Check questions

**Q1.** The cluster runs v1.35. The target is v1.37. Correct path?
- A. 1.35 → 1.37 directly with `kubeadm upgrade apply v1.37.x`
- B. 1.35 → 1.36 → 1.37, completing each step on all nodes (at least the control plane) before the next
- C. Upgrade workers first, then the control plane
- D. Reinstall the cluster

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q2.** On a worker node, which kubeadm command applies the upgrade?
- A. `kubeadm upgrade apply`
- B. `kubeadm upgrade node`
- C. `kubeadm init phase upgrade`
- D. `kubeadm join --upgrade`

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q3.** `kubectl drain w1` fails with "cannot delete Pods not managed by ReplicationController, ReplicaSet, Job, DaemonSet or StatefulSet". What does `--force` do?
- A. Deletes those bare Pods; they will not be recreated
- B. Moves them to another node
- C. Ignores them and leaves them running
- D. Converts them to Deployments

<details><summary>▶ Answer</summary>

**A.**
</details>

**Q4.** After upgrading the packages on a worker, `kubectl get nodes` still shows the old version for it. What was forgotten?
- A. `kubeadm upgrade apply`
- B. `systemctl daemon-reload && systemctl restart kubelet`
- C. `kubectl uncordon`
- D. Upgrading kubectl

<details><summary>▶ Answer</summary>

**B.**
</details>

---

## 3.4 etcd backup and restore [`etcd-backup-restore`]

### Key ideas
- On kubeadm, etcd is a static Pod listening on `https://127.0.0.1:2379` (and the node IP), using mutual TLS. You need **three files**: CA cert, client cert, client key. Copy their paths from `/etc/kubernetes/manifests/etcd.yaml`:
  - `--trusted-ca-file` → `--cacert`
  - `--cert-file` → `--cert`
  - `--key-file` → `--key`
- **Backup**: `etcdctl snapshot save` (talks to the live server, so it needs the endpoint and certs).
- **Restore**: `etcdutl snapshot restore` (offline file operation, no certs needed) into a **new data dir**, then point etcd at it. (`etcdctl snapshot restore` is deprecated and removed in etcd 3.6+; use etcdutl when it is available.)
- `ETCDCTL_API=3` is the default in current etcdctl, but setting it does no harm and old exam material uses it.

### Backup

```bash
sudo grep -E 'trusted-ca-file|cert-file|key-file|listen-client-urls|data-dir' /etc/kubernetes/manifests/etcd.yaml

sudo ETCDCTL_API=3 etcdctl \
  --endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/kubernetes/pki/etcd/ca.crt \
  --cert=/etc/kubernetes/pki/etcd/server.crt \
  --key=/etc/kubernetes/pki/etcd/server.key \
  snapshot save /opt/backup/etcd-snap.db

sudo etcdutl snapshot status /opt/backup/etcd-snap.db -w table     # hash, revision, total keys, size
```

If `etcdctl` is not installed on the node, use the one in the etcd Pod. `/var/lib/etcd` inside the Pod is a hostPath, so the file lands on the node:

```bash
kubectl -n kube-system exec etcd-cp1 -- etcdctl --endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/kubernetes/pki/etcd/ca.crt --cert=/etc/kubernetes/pki/etcd/server.crt \
  --key=/etc/kubernetes/pki/etcd/server.key snapshot save /var/lib/etcd/snap.db
sudo cp /var/lib/etcd/snap.db /opt/backup/
```

### Restore (single control plane)

```bash
# 1) stop the apiserver (and etcd) so nothing writes during the restore
sudo mkdir -p /root/manifests-bak
sudo mv /etc/kubernetes/manifests/kube-apiserver.yaml /etc/kubernetes/manifests/etcd.yaml /root/manifests-bak/

# 2) restore into a NEW directory
sudo etcdutl snapshot restore /opt/backup/etcd-snap.db --data-dir /var/lib/etcd-restore

# 3) point etcd's hostPath volume at the new directory
sudo vi /root/manifests-bak/etcd.yaml
#    volumes:
#    - hostPath: {path: /var/lib/etcd-restore, type: DirectoryOrCreate}   # was /var/lib/etcd
#      name: etcd-data
# (Leaving --data-dir=/var/lib/etcd unchanged is fine: that is the path inside the container.)

# 4) put the manifests back; kubelet restarts etcd, then the apiserver
sudo mv /root/manifests-bak/etcd.yaml /root/manifests-bak/kube-apiserver.yaml /etc/kubernetes/manifests/
watch sudo crictl ps            # wait for etcd and kube-apiserver to be Running
kubectl get pods -A             # objects created after the snapshot are gone
```

If the control-plane Pods misbehave after the restore, restart the kubelet (`sudo systemctl restart kubelet`) and give controller-manager and scheduler a minute.

### Lab
1. Create Deployment `before-backup`. Take a snapshot. Create Deployment `after-backup`.
2. Restore the snapshot as above. Confirm `before-backup` exists and `after-backup` does not.
3. Repeat the whole thing **without notes**, using only the docs page "Operating etcd clusters for Kubernetes". Target: 8 minutes.
4. Try `etcdctl snapshot save` with a wrong `--cert` and read the error (`authentication handshake failed` / `bad certificate`).
5. Run `etcdctl member list -w table` and `etcdctl endpoint health` with the same flags.

### Pitfalls
- Using `/etc/kubernetes/pki/apiserver-etcd-client.crt` vs `etcd/server.crt`: both can authenticate. Use the paths the task gives, or the ones in etcd.yaml.
- Restoring into the **existing** `/var/lib/etcd` without stopping etcd first, which corrupts it.
- Saving the snapshot to the path the task asks for **on the right host**.

### Check questions

**Q1.** Which three etcdctl flags must match the etcd static Pod's TLS settings?
- A. `--user --password --endpoints`
- B. `--cacert --cert --key`
- C. `--ca --tls --server`
- D. `--kubeconfig --context --namespace`

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q2.** You restored a snapshot into `/var/lib/etcd-new`. What else must change for etcd to use it on a kubeadm cluster?
- A. Nothing; etcd finds it automatically
- B. The `etcd-data` hostPath (or `--data-dir`) in `/etc/kubernetes/manifests/etcd.yaml`
- C. The kubelet config
- D. The apiserver `--etcd-servers` flag

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q3.** Which tool is recommended for `snapshot restore` with etcd 3.5+?
- A. `etcdctl snapshot restore`
- B. `etcdutl snapshot restore`
- C. `kubeadm etcd restore`
- D. `kubectl restore`

<details><summary>▶ Answer</summary>

**B.**
</details>

---

## 3.5 Highly available control plane [`ha-control-plane`]

### Key ideas
- **Stacked etcd** (default): each control-plane node runs its own etcd member. Simpler, fewer machines; losing a node loses an apiserver *and* an etcd member.
- **External etcd**: etcd on separate hosts. More machines, better isolation.
- etcd uses Raft, so a majority (**quorum = ⌊n/2⌋ + 1**) must be up. 3 members tolerate 1 failure, 5 tolerate 2. Even numbers add no tolerance (4 still tolerates 1).
- All clients (kubelets, kubectl, controllers) must reach the apiservers through a **stable endpoint**: a load balancer or VIP (HAProxy + keepalived, kube-vip, or a cloud LB) passed as `--control-plane-endpoint`.
- The scheduler and controller-manager run on every control-plane node but use **leader election** (only one is active; see the Leases in kube-system).

### Commands

```bash
# first control plane
sudo kubeadm init --control-plane-endpoint "k8s-api.lab:6443" --upload-certs --pod-network-cidr=192.168.0.0/16
# output contains two join commands; the control-plane one includes:
sudo kubeadm join k8s-api.lab:6443 --token <t> --discovery-token-ca-cert-hash sha256:<h> \
  --control-plane --certificate-key <key>
# the uploaded certs (and the key) expire after 2 hours; to regenerate:
sudo kubeadm init phase upload-certs --upload-certs
kubeadm token create --print-join-command        # then add --control-plane --certificate-key <new key>

kubectl get leases -n kube-system                 # kube-scheduler, kube-controller-manager holders
kubectl -n kube-system exec etcd-cp1 -- etcdctl --endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/kubernetes/pki/etcd/ca.crt --cert=/etc/kubernetes/pki/etcd/server.crt \
  --key=/etc/kubernetes/pki/etcd/server.key member list -w table
```

### Lab (optional, needs 3 control-plane VMs + 1 LB VM or kube-vip)
1. Put HAProxy on an extra VM forwarding TCP 6443 to cp1–cp3 (or follow the kube-vip static Pod guide).
2. Init cp1 with `--control-plane-endpoint` and `--upload-certs`; join cp2 and cp3 as control planes.
3. Run `etcdctl member list`: 3 members. Shut down cp1 and confirm `kubectl get nodes` still works through the LB.
4. Shut down a second control plane and see the API fail (no quorum).

If you cannot build it, make sure you can at least explain stacked vs external etcd, quorum math, and the `--control-plane` / `--certificate-key` join flags. Exam tasks here are usually about knowing the procedure.

### Check questions

**Q1.** How many etcd member failures can a 5-member cluster tolerate?
- A. 1
- B. 2
- C. 3
- D. 4

<details><summary>▶ Answer</summary>

**B.** Quorum is 3.
</details>

**Q2.** Which `kubeadm init` flag gives all nodes a stable, load-balanced apiserver address?
- A. `--apiserver-advertise-address`
- B. `--control-plane-endpoint`
- C. `--upload-certs`
- D. `--node-name`

<details><summary>▶ Answer</summary>

**B.** A is the local node's own IP.
</details>

**Q3.** A second control-plane join fails because the certificate key is invalid. It is 3 hours after init. Fix?
- A. Re-run `kubeadm init`
- B. `kubeadm init phase upload-certs --upload-certs` to get a new key, then join with it
- C. Copy `/etc/kubernetes/pki` manually and join as a worker
- D. Use `--ignore-preflight-errors=all`

<details><summary>▶ Answer</summary>

**B.** C also works if done correctly, but it is slower and easy to get wrong.
</details>

---

## 3.6 Certificates, PKI and control-plane static Pod manifests [`certificates-pki`]

### Key ideas
- kubeadm PKI in `/etc/kubernetes/pki`: `ca.crt/key` (cluster CA, 10 years), `apiserver.crt` (serving cert, with SANs for every name/IP of the API), `apiserver-kubelet-client.crt`, `front-proxy-*`, `sa.key/pub` (ServiceAccount token signing), and `etcd/` (etcd's own CA, server, peer, healthcheck-client).
- Leaf certs are valid for **1 year**. A cluster upgrade renews them; otherwise renew by hand.
- The kubelet client cert rotates automatically (`/var/lib/kubelet/pki/kubelet-client-current.pem`).
- Control-plane config lives in the **static Pod manifests**. Editing a flag (apiserver, controller-manager, scheduler, etcd) = edit the YAML; kubelet notices and recreates the Pod within ~30 s.

### Commands

```bash
sudo kubeadm certs check-expiration
sudo kubeadm certs renew all              # or: renew apiserver
# renewed certs are only loaded after the static Pods restart:
sudo mkdir -p /root/tmp-manifests && sudo mv /etc/kubernetes/manifests/*.yaml /root/tmp-manifests/
sleep 20 && sudo mv /root/tmp-manifests/*.yaml /etc/kubernetes/manifests/
sudo cp /etc/kubernetes/admin.conf ~/.kube/config            # admin.conf was renewed too

# inspect a cert
sudo openssl x509 -in /etc/kubernetes/pki/apiserver.crt -noout -subject -issuer -dates
sudo openssl x509 -in /etc/kubernetes/pki/apiserver.crt -noout -text | grep -A1 'Subject Alternative Name'

# which cert/key flags does the apiserver use?
sudo grep -E '\-\-(tls|client-ca|etcd|kubelet-client)' /etc/kubernetes/manifests/kube-apiserver.yaml
```

### Lab
1. Run `kubeadm certs check-expiration` and note the residual time and the CA expiry.
2. `kubeadm certs renew apiserver`, then confirm the new `notAfter` with openssl (it only takes effect after the apiserver restarts).
3. Break the apiserver on purpose: change `--etcd-cafile` to a wrong path. Watch with `crictl ps -a`, then read the error with `crictl logs <id>` (or `/var/log/pods/kube-system_kube-apiserver-*/`). Fix it.
4. Find the SANs on the apiserver cert and confirm the node IP and `kubernetes.default.svc` are present.

### Pitfalls
- Keeping a backup copy of a manifest **inside** `/etc/kubernetes/manifests`: kubelet runs it as a second Pod. Keep backups elsewhere.
- After renewing certs, also refresh `~/.kube/config`, which embeds an admin cert.

### Check questions

**Q1.** After `kubeadm certs renew all`, the apiserver still serves the old certificate. Why?
- A. The renewal failed silently
- B. The control-plane static Pods must be restarted to load the new files
- C. You must run `kubeadm upgrade apply`
- D. The CA must also be renewed

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q2.** You change a flag in `/etc/kubernetes/manifests/kube-scheduler.yaml`. How does the change take effect?
- A. `kubectl apply -f` the file
- B. kubelet detects the change and recreates the static Pod automatically
- C. `systemctl restart kube-scheduler`
- D. `kubeadm upgrade node`

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q3.** How long are kubeadm-generated leaf certificates valid by default?
- A. 90 days
- B. 1 year
- C. 5 years
- D. 10 years

<details><summary>▶ Answer</summary>

**B.** The CA lasts 10 years.
</details>

---

## Chapter 3: exam-style tasks

**T3.1 (12 min)** Cluster `k8s-c1` has control plane `cp1` (v1.36.x) and worker `w1`. Upgrade **only the control plane** components and kubelet/kubectl on `cp1` to v1.37.x (the latest patch available in apt). Do not upgrade `w1`.

<details><summary>▶ Solution</summary>

Follow section 3.3, steps a–d, on cp1 only. Check `kubectl get nodes`: cp1 is v1.37.x and w1 is v1.36.x, which is allowed by the skew policy.
</details>

**T3.2 (8 min)** Back up etcd running on `cp1` to `/opt/etcd-backup.db`, using the certificates in `/etc/kubernetes/pki/etcd`. Then restore the snapshot from `/opt/etcd-old.db` (given) so the cluster uses it, with data dir `/var/lib/etcd-old`.

<details><summary>▶ Solution</summary>

Section 3.4 backup command with `snapshot save /opt/etcd-backup.db`. Then `etcdutl snapshot restore /opt/etcd-old.db --data-dir /var/lib/etcd-old`, change the `etcd-data` hostPath in etcd.yaml to `/var/lib/etcd-old`, and wait for etcd and the apiserver to come back. Verify with `kubectl get pods -A`.
</details>

**T3.3 (6 min)** A new node `w2` has containerd and the kube packages installed. Join it to the cluster. The original join token has expired.

<details><summary>▶ Solution</summary>

```bash
# on cp1
kubeadm token create --print-join-command
# on w2
sudo <paste the join command>
# on cp1
kubectl get nodes      # w2 Ready once the CNI Pod runs there
```
</details>

**T3.4 (4 min)** Write the expiration date of the apiserver certificate on `cp1` to `/opt/apiserver-expiry.txt`, and renew only that certificate.

<details><summary>▶ Solution</summary>

```bash
sudo openssl x509 -in /etc/kubernetes/pki/apiserver.crt -noout -enddate | cut -d= -f2 > /opt/apiserver-expiry.txt
# (or take the date from kubeadm certs check-expiration)
sudo kubeadm certs renew apiserver
# restart the apiserver static Pod: move its manifest out, wait about 20 s, move it back
```
</details>

**T3.5 (3 min)** Node `w1` needs maintenance. Make sure no new Pods are scheduled on it and existing workloads are moved, ignoring DaemonSets. After "maintenance", make it schedulable again.

<details><summary>▶ Solution</summary>

```bash
k drain w1 --ignore-daemonsets --delete-emptydir-data
k uncordon w1
```
</details>
