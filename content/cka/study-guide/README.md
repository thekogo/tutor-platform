# CKA Study Guide (Kubernetes v1.37)

A self-study companion to [../outline.md](../outline.md) and [../graph.yaml](../graph.yaml).
Each chapter maps to one Topic of the graph, and each section maps to one Concept (the Concept slug is shown in brackets).

## How to use each section

1. **Read** "Key ideas" and "Commands / YAML".
2. **Do the lab** on your own cluster. Type every command yourself; do not copy-paste. Typing speed is part of the exam.
3. **Answer the check questions** before opening the hidden answer (`▶ Answer`).
4. At the end of each chapter, do the **exam-style tasks** under a timer (target time is shown next to each task), then compare with the solution.

## Chapters

| # | Chapter | Exam domain (weight) | Week |
|---|---|---|---|
| 0 | [Lab setup](00-lab-setup.md) | n/a | 1 |
| 1 | [Foundations](01-foundations.md) | all domains | 1 |
| 2 | [Workloads & Scheduling](02-workloads-scheduling.md) | Workloads & Scheduling (15%) | 2–3 |
| 3 | [Cluster Lifecycle (kubeadm)](03-cluster-lifecycle.md) | Cluster Architecture (25%, part 1) | 4 |
| 4 | [Access & Extensions](04-access-and-extensions.md) | Cluster Architecture (25%, part 2) | 5 |
| 5 | [Services & Networking](05-services-networking.md) | Services & Networking (20%) | 6 |
| 6 | [Storage](06-storage.md) | Storage (10%) | 7 |
| 7 | [Troubleshooting](07-troubleshooting.md) | Troubleshooting (30%) | 7 |
| 8 | [Mock exam](08-mock-exam.md) | all | 8 |

## Conventions

- `k` = `kubectl` (alias set up in chapter 1).
- `$do` = `--dry-run=client -o yaml`.
- `$now` = `--force --grace-period=0`.
- Commands that start with `sudo` run **on a node** (after `ssh`); everything else runs from your workstation or the exam terminal.
- Version-specific facts were checked against v1.37 behaviour as far as possible. When a flag or default has changed recently, the text says so. Always confirm with `kubectl <cmd> -h` and kubernetes.io/docs, because those are what you will have in the exam.
