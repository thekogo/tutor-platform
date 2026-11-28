# CKA preparation course: outline and study plan

The graph source is [graph.yaml](graph.yaml) (7 Topics, 39 Concepts). The rendered map is [graph.md](graph.md). The full lessons, labs and practice questions are in [study-guide/](study-guide/README.md).
Research date: 2026-11-26. Check the [LF exam page](https://training.linuxfoundation.org/certification/certified-kubernetes-administrator-cka/)
and Candidate Handbook again before booking.

## The exam

| Item | Value |
|---|---|
| Format | Online, proctored, **hands-on**. You solve tasks on a command line against live clusters. There is no multiple choice. |
| Time | 2 hours. About 15–20 tasks, with partial credit. |
| Pass mark | 66% |
| Version | Kubernetes **v1.37** |
| Price | $445. Includes 1 free retake and 2 killer.sh simulator sessions (36 h each). |
| Validity | 2 years |
| Open book | One tab for kubernetes.io/docs, kubernetes.io/blog and helm.sh/docs (Gateway API docs per secondary sources) |

## Domains → Topics

| Exam domain | Weight | Topic(s) in this course |
|---|---|---|
| Troubleshooting | 30% | `troubleshooting` |
| Cluster Architecture, Installation & Configuration | 25% | `cluster-lifecycle`, `access-and-extensions` |
| Services & Networking | 20% | `services-networking` |
| Workloads & Scheduling | 15% | `workloads-scheduling` |
| Storage | 10% | `storage` |
| (not a domain) | n/a | `foundations`: kubectl speed, contexts, exam strategy |

## Modules (suggested order, 8 weeks at ~8–10 h/week)

**Week 1: Foundations.** Architecture, imperative kubectl with `--dry-run=client -o yaml`, YAML/labels/selectors,
kubeconfig contexts and namespaces. *Lab:* build a kind cluster. Set up aliases and vim. Generate 10 resources without writing YAML by hand.

**Week 2: Workloads I.** Pods, init containers, sidecars, Deployments, rolling update/rollback, ConfigMaps and Secrets.
*Lab:* roll a bad image out, then undo it. Inject config three ways (env, envFrom, volume).

**Week 3: Workloads II and Scheduling.** Probes, DaemonSet, StatefulSet, Job/CronJob, requests/limits, LimitRange,
ResourceQuota, HPA, node affinity, taints/tolerations, pod anti-affinity, Pod Security Admission.
*Lab:* pin a Pod to a tainted node. Make HPA scale under load.

**Week 4: Cluster lifecycle (kubeadm).** Infrastructure prep, `kubeadm init/join`, upgrade one minor version, etcd snapshot
save/restore, cert renewal, HA topology. *Lab:* 1 CP + 2 worker VMs (Vagrant/multipass). Upgrade the cluster. Back up and restore etcd **twice**.

**Week 5: Access and extensions.** RBAC with ServiceAccounts and `auth can-i`, Helm install/upgrade/rollback, Kustomize
overlays, CRI/CNI/CSI, CRDs and operators. *Lab:* install a CNI and an operator via Helm. Grant a ServiceAccount least privilege.

**Week 6: Services and networking.** Pod network model, Service types and EndpointSlices, CoreDNS, NetworkPolicy,
Ingress, **Gateway API** (GatewayClass → Gateway → HTTPRoute). *Lab:* default-deny a namespace, then open only the needed paths and DNS.
Convert an Ingress to an HTTPRoute.

**Week 7: Storage and troubleshooting.** Volumes, PV/PVC, access modes, reclaim policies, StorageClass with
WaitForFirstConsumer. Troubleshooting drills: broken static Pod manifest, stopped kubelet, Service with no endpoints, DNS failure,
CrashLoopBackOff, Pending Pod. *Lab:* have a partner (or a script) break the cluster, then fix it under a timer.

**Week 8: Exam rehearsal.** killer.sh session 1, then review every miss against the docs. Redo the weak Topics. killer.sh session 2 about 3 days
before the exam. Bookmark the doc pages for etcd, kubeadm upgrade, NetworkPolicy, PV/PVC and Gateway API.

## Exam-day rules of thumb
1. Run the task's `kubectl config use-context` command **every time**.
2. Skim all tasks first. Do the quick, high-value ones. Flag anything that takes over ~8 minutes.
3. Generate YAML imperatively. Look fields up with `kubectl explain`.
4. Verify every task: `get`, `describe`, `auth can-i`, curl from a temporary Pod.
5. After `ssh` to a node, `exit` back before the next task.

## Platform note
The platform marks multiple choice only, so Questions train recognition and diagnosis through five formats:
command-choice, manifest-fix, output-reading, scenario-diagnosis and concept-check. Each Lesson must end with a short hands-on lab.
Hands-on practice is still required to pass.
