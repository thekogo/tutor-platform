# 0. Lab setup

You need two kinds of lab:

| Lab | Use for | Cost |
|---|---|---|
| **A. kind** (Kubernetes in Docker) | Chapters 1, 2, 4, 5, 6, most of 7. Fast to create and destroy. | Docker only |
| **B. kubeadm on VMs** | Chapter 3 (install, upgrade, etcd, certs) and node/control-plane troubleshooting in chapter 7. | 3 VMs, 2 vCPU / 2–4 GB each |

The real exam uses kubeadm-built clusters on Ubuntu, so do lab B at least twice end to end.

---

## A. kind cluster (multi-node, Calico CNI)

We disable kind's default CNI and install Calico, so NetworkPolicy is guaranteed to be enforced.

```bash
# Install kind + kubectl + helm (Linux/WSL)
curl -Lo ./kind https://kind.sigs.k8s.io/dl/latest/kind-linux-amd64 && chmod +x kind && sudo mv kind /usr/local/bin/
curl -LO "https://dl.k8s.io/release/$(curl -Ls https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install kubectl /usr/local/bin/
curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash   # or get-helm-4

cat > kind-cka.yaml <<'EOF'
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  disableDefaultCNI: true
  podSubnet: 192.168.0.0/16
nodes:
  - role: control-plane
    extraPortMappings:            # lets you reach NodePort 30080 from your host
      - containerPort: 30080
        hostPort: 30080
  - role: worker
  - role: worker
EOF
kind create cluster --name cka --config kind-cka.yaml   # add --image kindest/node:v1.37.x to pin the version

# Calico (check the latest version on docs.tigera.io)
kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.30.0/manifests/tigera-operator.yaml
kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.30.0/manifests/custom-resources.yaml
kubectl get nodes -w          # wait until all 3 are Ready
```

### metrics-server (needed for `kubectl top` and HPA)

```bash
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
# kind kubelets use self-signed certs:
kubectl -n kube-system patch deploy metrics-server --type=json \
  -p='[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]'
kubectl top nodes             # works after ~60 s
```

### Get a shell on a kind "node"

```bash
docker exec -it cka-control-plane bash    # behaves like ssh to the node
crictl ps                                  # containerd is the runtime inside
ls /etc/kubernetes/manifests               # static Pod manifests are here too
```

Reset any time: `kind delete cluster --name cka`.

---

## B. kubeadm cluster on VMs (1 control plane + 2 workers)

Use any VM tool: multipass, Vagrant + VirtualBox, Hyper-V, or cloud VMs. Example with **multipass**:

```bash
for n in cp1 w1 w2; do multipass launch 24.04 -n $n -c 2 -m 3G -d 20G; done
multipass shell cp1
```

Chapter 3 walks through the install step by step. To practise the **upgrade**, install **one minor version below** the current one (for example v1.36), then upgrade to v1.37.

Snapshots save time: take a VM snapshot after the install, so you can break and restore the cluster repeatedly.

---

## Shell setup you should also use in the exam

The exam terminal already has `k` aliased and bash completion on, but check it, and add the rest in the first minute:

```bash
cat >> ~/.bashrc <<'EOF'
alias k=kubectl
source <(kubectl completion bash)
complete -o default -F __start_kubectl k
export do="--dry-run=client -o yaml"
export now="--force --grace-period=0"
EOF
cat >> ~/.vimrc <<'EOF'
set tabstop=2 shiftwidth=2 expandtab autoindent number
EOF
source ~/.bashrc
```

## Practice resources

- **killer.sh**: two sessions come with the exam. Save them for week 8; the simulator is harder than the real exam.
- **Killercoda CKA scenarios** (free, in the browser): good for quick daily drills.
- **kubernetes.io/docs**: practise finding pages with the docs search, because that is the only reference you get in the exam.
