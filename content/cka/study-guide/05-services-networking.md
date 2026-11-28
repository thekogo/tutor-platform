# 5. Services & Networking (20%)

Official competencies: connectivity between Pods · Network Policies · ClusterIP, NodePort, LoadBalancer and endpoints ·
Gateway API for Ingress traffic · Ingress controllers and Ingress resources · CoreDNS.

Lab tip: keep a throwaway client Pod ready.

```bash
k run tmp --image=busybox:1.36 --rm -it --restart=Never -- sh
#   inside: wget -qO- -T2 http://web.default.svc.cluster.local ; nslookup web ; nc -zv -w2 10.0.0.5 80
k run curl --image=curlimages/curl --rm -it --restart=Never -- curl -s -m3 http://web
```

---

## 5.1 Pod connectivity (the Kubernetes network model) [`pod-connectivity`]

### Key ideas
The model has three rules:
1. Every Pod gets its own IP address (from the Pod CIDR; each node usually owns a slice, `node.spec.podCIDR`).
2. Pods can reach all other Pods on any node **without NAT**.
3. Agents on a node (kubelet) can reach all Pods on that node.

Containers in a Pod share the IP and talk over `localhost`. The **CNI plugin** implements the model (veth pairs, bridges or routing, overlays such as VXLAN, or eBPF). Pod IPs are ephemeral, which is why Services exist.

```bash
k get pods -o wide                                     # Pod IPs and nodes
k get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.podCIDR}{"\n"}{end}'
k get pod web -o jsonpath='{.status.podIP}'
k exec tmp -- wget -qO- -T2 <pod-ip>:80
sudo grep cluster-cidr /etc/kubernetes/manifests/kube-controller-manager.yaml   # Pod CIDR
sudo grep service-cluster-ip-range /etc/kubernetes/manifests/kube-apiserver.yaml # Service CIDR
```

### Lab
1. Run two nginx Pods pinned to different nodes (use `nodeName`). From a busybox Pod, reach both by Pod IP.
2. Find the Pod CIDR and Service CIDR from the control-plane manifests.
3. Delete a Pod managed by a Deployment and observe that the replacement gets a different IP.

### Check questions

**Q1.** Which component gives a Pod its IP address?
- A. kube-proxy
- B. The CNI plugin
- C. CoreDNS
- D. kube-scheduler

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q2.** Two containers in the same Pod need to talk over TCP. What address do they use?
- A. The Service IP
- B. `localhost`
- C. The node IP
- D. The other container's IP

<details><summary>▶ Answer</summary>

**B.**
</details>

---

## 5.2 Services and endpoints [`services`]

### Key ideas

| Type | Reachable from | Notes |
|---|---|---|
| **ClusterIP** (default) | inside the cluster | virtual IP from the Service CIDR |
| **NodePort** | `<anyNodeIP>:<nodePort>` (30000–32767) + ClusterIP | `externalTrafficPolicy: Local` keeps the client IP but only routes to Pods on the receiving node |
| **LoadBalancer** | external LB IP + NodePort + ClusterIP | needs a cloud controller or MetalLB/cloud-provider-kind; otherwise EXTERNAL-IP stays `<pending>` |
| **ExternalName** | DNS CNAME to an external name | no proxying, no selector |
| **Headless** (`clusterIP: None`) | DNS returns the Pod IPs directly | used by StatefulSets |

- **Ports**: `port` (on the Service) → `targetPort` (on the container; number or **named port**) and `nodePort` (on every node).
- **Endpoints**: the Service controller writes **EndpointSlices** listing the IPs of Pods that match the `selector` **and are Ready**. A Service with no selector has no automatic endpoints; you can create the EndpointSlice by hand to point at external IPs. The old `Endpoints` API still exists but is deprecated (since v1.33) in favour of EndpointSlices.
- **kube-proxy** turns Services into iptables, IPVS or nftables rules on every node (`k -n kube-system get cm kube-proxy -o yaml | grep mode`).
- `sessionAffinity: ClientIP` makes a client stick to one Pod.

```bash
k expose deploy web --port=80 --target-port=8080                       # ClusterIP
k expose deploy web --name=web-np --port=80 --target-port=8080 --type=NodePort
k create svc nodeport web-np2 --tcp=80:8080 --node-port=30080          # selector = app=web-np2; fix it if needed!
k create svc clusterip db --clusterip=None --tcp=5432                   # headless
k create svc externalname ext --external-name=example.com
k get svc,endpointslices -o wide
k get endpointslices -l kubernetes.io/service-name=web
k describe svc web                                                      # Endpoints: line
```

```yaml
apiVersion: v1
kind: Service
metadata: {name: web}
spec:
  type: NodePort
  selector: {app: web}
  ports:
    - name: http
      port: 80
      targetPort: http          # named containerPort in the Pod spec
      nodePort: 30080
```

### Lab
1. Create Deployment `web` (nginx, 3 replicas). Expose it as ClusterIP on port 80 and curl it from a temp Pod several times.
2. Create a NodePort 30080 Service. On kind with the port mapping from lab setup, `curl localhost:30080` from your host.
3. Break the selector (`k edit svc web` → `app: wrong`). Check the empty EndpointSlice and the failing curl. Fix it.
4. Use a wrong `targetPort` (8080 for nginx). Endpoints exist but curl fails with "connection refused". Remember this difference from step 3.
5. Create a selector-less Service plus a manual EndpointSlice pointing at an IP outside the cluster.
6. Create a headless Service and run `nslookup` from busybox to see one A record per Pod.

### Pitfalls
- `k create svc` sets the selector to `app=<service-name>`. Prefer `k expose`, which copies the Deployment's selector.
- Endpoints present + connection refused → wrong `targetPort`. No endpoints → selector mismatch or Pods not Ready.

### Check questions

**Q1.** A Service shows `Endpoints: <none>`. The Deployment's Pods are Running and Ready. What is the most likely cause?
- A. Wrong targetPort
- B. Service selector does not match the Pod labels
- C. CoreDNS is down
- D. NodePort out of range

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q2.** Which Service type gives you a stable DNS name that resolves directly to each Pod's IP, with no virtual IP?
- A. ClusterIP
- B. NodePort
- C. Headless (`clusterIP: None`)
- D. ExternalName

<details><summary>▶ Answer</summary>

**C.**
</details>

**Q3.** A LoadBalancer Service on a bare kubeadm cluster shows `EXTERNAL-IP <pending>`. Can clients reach it at all?
- A. No
- B. Yes, through its NodePort on any node, and its ClusterIP inside the cluster
- C. Only from the control plane
- D. Only through CoreDNS

<details><summary>▶ Answer</summary>

**B.** A LoadBalancer Service is also a NodePort Service.
</details>

**Q4.** For `port: 80, targetPort: 8080, nodePort: 30080`, which port does the container listen on?
- A. 80
- B. 8080
- C. 30080
- D. Any

<details><summary>▶ Answer</summary>

**B.**
</details>

---

## 5.3 CoreDNS [`coredns`]

### Key ideas
- CoreDNS runs as a Deployment `coredns` in kube-system behind the Service `kube-dns` (ClusterIP usually `10.96.0.10`). kubelet writes this IP into every Pod's `/etc/resolv.conf` (`dnsPolicy: ClusterFirst`).
- Names:
  - Service: `<svc>.<ns>.svc.cluster.local` (A/AAAA → ClusterIP; headless → Pod IPs)
  - StatefulSet Pod behind a headless Service: `<pod>.<svc>.<ns>.svc.cluster.local`
  - Pod: `<ip-with-dashes>.<ns>.pod.cluster.local`
  - SRV records for named ports: `_http._tcp.<svc>.<ns>.svc.cluster.local`
- A Pod's resolv.conf has `search <ns>.svc.cluster.local svc.cluster.local cluster.local` and `options ndots:5`, so `web` resolves within the same namespace and `web.other` resolves across namespaces.
- Configuration is the **Corefile** in ConfigMap `coredns` (kube-system). The CoreDNS `reload` plugin picks up edits after a short time; or `k -n kube-system rollout restart deploy coredns`.
- `dnsPolicy`: `ClusterFirst` (default), `Default` (inherit the node's resolv.conf), `None` (use `dnsConfig` only), `ClusterFirstWithHostNet` (for hostNetwork Pods).

```bash
k -n kube-system get deploy coredns ; k -n kube-system get svc kube-dns
k -n kube-system get cm coredns -o yaml
k run tmp --image=busybox:1.36 --rm -it --restart=Never -- sh -c 'cat /etc/resolv.conf; nslookup kubernetes.default'
k -n kube-system logs -l k8s-app=kube-dns
```

Corefile with a custom stub domain and a static host entry:

```
.:53 {
    errors
    health { lameduck 5s }
    ready
    kubernetes cluster.local in-addr.arpa ip6.arpa {
       pods insecure
       fallthrough in-addr.arpa ip6.arpa
       ttl 30
    }
    hosts {
       10.0.0.50 legacy.internal
       fallthrough
    }
    prometheus :9153
    forward . /etc/resolv.conf
    cache 30
    loop
    reload
    loadbalance
}
corp.example:53 {
    forward . 10.0.0.53
}
```

### Lab
1. From busybox, resolve `kubernetes.default`, `kube-dns.kube-system.svc.cluster.local`, and a Service in another namespace by its short name (fails) and by `name.ns` (works).
2. Create a StatefulSet with a headless Service and resolve `web-0.web`.
3. Add the `hosts` block above to the Corefile and resolve `legacy.internal` from a Pod.
4. Scale CoreDNS to 0, see name resolution fail (IP access still works), then scale it back to 2.
5. Create a Pod with `dnsPolicy: None` and a custom `dnsConfig` (nameserver 1.1.1.1) and check its resolv.conf.

### Pitfalls
- `nslookup` in some busybox versions prints odd errors for search domains. Use `busybox:1.36` or `registry.k8s.io/e2e-test-images/jessie-dnsutils` for reliable results.
- Indentation errors in the Corefile crash CoreDNS (CrashLoopBackOff). Check its logs.

### Check questions

**Q1.** A Pod in namespace `a` must reach Service `api` in namespace `b`. Which name works?
- A. `api`
- B. `api.b`
- C. `b.api`
- D. `api.svc`

<details><summary>▶ Answer</summary>

**B.** The search list completes it to `api.b.svc.cluster.local`.
</details>

**Q2.** Where is CoreDNS configured on a kubeadm cluster?
- A. `/etc/coredns/Corefile` on each node
- B. ConfigMap `coredns` in kube-system
- C. kubelet config `clusterDNS`
- D. The `kube-dns` Service annotations

<details><summary>▶ Answer</summary>

**B.** kubelet's `clusterDNS` only sets the nameserver IP that Pods use.
</details>

**Q3.** Pods can reach other Pods by IP but not by Service name. The first thing to check:
- A. NetworkPolicy on port 80
- B. CoreDNS Pods and the `kube-dns` Service and its endpoints
- C. kube-proxy mode
- D. etcd

<details><summary>▶ Answer</summary>

**B.**
</details>

---

## 5.4 Network Policies [`network-policies`]

### Key ideas
- Without any NetworkPolicy, all traffic is allowed. A policy needs a CNI that **enforces** it (Calico, Cilium, … ; plain Flannel does not).
- A policy selects Pods with `podSelector` (`{}` = all Pods in the policy's namespace). For each direction listed in `policyTypes`, selected Pods become **deny-all except what some policy allows**.
- Policies are **additive** (a union of allows). Order does not matter. There is no explicit deny.
- Peers in `from` / `to`: `podSelector` (same namespace), `namespaceSelector`, both combined, or `ipBlock` (`cidr`, `except`).
- **The classic trap**:

```yaml
from:
  - namespaceSelector: {matchLabels: {team: a}}
    podSelector: {matchLabels: {app: api}}     # ONE item → AND: api Pods in team=a namespaces
---
from:
  - namespaceSelector: {matchLabels: {team: a}}
  - podSelector: {matchLabels: {app: api}}     # TWO items → OR: anything in team=a namespaces, OR api Pods in my namespace
```

- Namespaces automatically carry the label `kubernetes.io/metadata.name=<name>`, which is handy for selecting a namespace by name.
- If you default-deny **egress**, also allow DNS (UDP and TCP 53 to kube-dns), or every name lookup fails.

### Templates

```yaml
# 1) default deny all ingress + egress in a namespace
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata: {name: default-deny, namespace: prod}
spec:
  podSelector: {}
  policyTypes: [Ingress, Egress]
---
# 2) allow DNS egress for every Pod in the namespace
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata: {name: allow-dns, namespace: prod}
spec:
  podSelector: {}
  policyTypes: [Egress]
  egress:
    - to:
        - namespaceSelector: {matchLabels: {kubernetes.io/metadata.name: kube-system}}
          podSelector: {matchLabels: {k8s-app: kube-dns}}
      ports: [{protocol: UDP, port: 53}, {protocol: TCP, port: 53}]
---
# 3) db accepts only from api Pods on 5432; api may only talk to db
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata: {name: db-ingress, namespace: prod}
spec:
  podSelector: {matchLabels: {app: db}}
  policyTypes: [Ingress]
  ingress:
    - from: [{podSelector: {matchLabels: {app: api}}}]
      ports: [{protocol: TCP, port: 5432}]
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata: {name: api-egress, namespace: prod}
spec:
  podSelector: {matchLabels: {app: api}}
  policyTypes: [Egress]
  egress:
    - to: [{podSelector: {matchLabels: {app: db}}}]
      ports: [{protocol: TCP, port: 5432}]
---
# 4) allow from the monitoring namespace and from an external CIDR
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata: {name: web-ingress, namespace: prod}
spec:
  podSelector: {matchLabels: {app: web}}
  policyTypes: [Ingress]
  ingress:
    - from:
        - namespaceSelector: {matchLabels: {kubernetes.io/metadata.name: monitoring}}
        - ipBlock: {cidr: 203.0.113.0/24, except: [203.0.113.7/32]}
      ports: [{port: 80}]
```

### Lab (needs Calico from lab setup)
1. Namespace `prod` with Deployments `web`, `api` and `db` (nginx is fine for all; label them `app=...`) and Services for each.
2. From a busybox Pod in `prod`, confirm everything can reach everything.
3. Apply default-deny. Confirm everything fails, including DNS (`nslookup db` times out).
4. Add allow-dns, then the db/api policies. Test: api→db works, web→db fails, api→web fails.
5. Rewrite one policy using the AND form and then the OR form, and test the difference with a Pod in another namespace.
6. Practise reading: given a policy YAML, write down in words what it allows before testing it.

### Pitfalls
- Forgetting `policyTypes: [Egress]`: an `egress:` section without it may still be inferred, but an **egress-only deny** needs `policyTypes` explicitly. Always write `policyTypes`.
- Port in a policy = the **Pod's** port (targetPort), not the Service port.
- Writing `podSelector` at the wrong indentation, which turns an AND into an OR.

### Check questions

**Q1.** A namespace has one policy: `podSelector: {}`, `policyTypes: [Ingress]`, no ingress rules. Effect?
- A. All ingress to all Pods in the namespace is denied; egress unaffected
- B. All traffic allowed
- C. Ingress and egress denied
- D. Only Pods without labels are affected

<details><summary>▶ Answer</summary>

**A.**
</details>

**Q2.** Which `from` allows **only** Pods labelled `app=api` that are in namespaces labelled `team=a`?
- A. Two list items, one `namespaceSelector` and one `podSelector`
- B. One list item containing both `namespaceSelector` and `podSelector`
- C. `ipBlock` with the namespace CIDR
- D. `podSelector` alone

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q3.** After applying default-deny egress, Pods cannot reach anything **by name**, even targets that have allow rules. Why?
- A. kube-proxy is broken
- B. DNS egress (port 53 to CoreDNS) is not allowed
- C. The Service has no endpoints
- D. NetworkPolicies block IP traffic only

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q4.** Service `db` has `port: 80, targetPort: 5432`. What port goes in a NetworkPolicy allowing traffic to the db Pods?
- A. 80
- B. 5432
- C. Both
- D. None; policies ignore ports

<details><summary>▶ Answer</summary>

**B.**
</details>

---

## 5.5 Ingress controllers and Ingress resources [`ingress`]

### Key ideas
- An **Ingress** resource declares HTTP(S) routing (host and path → Service:port, TLS). It does **nothing without an Ingress controller** (Traefik, HAProxy, NGINX-based controllers, cloud LBs…). Note: the community `ingress-nginx` project was retired in 2026; exam clusters may still run some controller, so just use the one installed.
- `ingressClassName` selects the controller (`k get ingressclass`; one may be marked default with the annotation `ingressclass.kubernetes.io/is-default-class: "true"`).
- `pathType`: `Prefix` (by path element: `/api` matches `/api/v1`, not `/apis`), `Exact`, or `ImplementationSpecific`.
- TLS: a `kubernetes.io/tls` Secret in the **same namespace** as the Ingress, listed under `tls[].secretName` with its `hosts`.
- The backend Service must be in the same namespace as the Ingress.

```bash
k get ingressclass
k create ingress web --class=traefik \
  --rule="shop.example.com/=web:80" \
  --rule="shop.example.com/api*=api:8080" \
  --rule="secure.example.com/*=web:80,tls=web-tls" $do
k describe ingress web
curl -H 'Host: shop.example.com' http://<node-ip>:<controller-nodeport>/
```
(In `--rule`, a trailing `*` means `pathType: Prefix`; without it, `Exact`.)

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata: {name: web, namespace: default}
spec:
  ingressClassName: traefik
  tls:
    - hosts: [secure.example.com]
      secretName: web-tls
  rules:
    - host: shop.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend: {service: {name: web, port: {number: 80}}}
          - path: /api
            pathType: Prefix
            backend: {service: {name: api, port: {number: 8080}}}
```

### Lab
1. Install Traefik as the controller, exposed on NodePort 30080 (mapped by the kind config):
   ```bash
   helm repo add traefik https://traefik.github.io/charts && helm repo update
   helm install traefik traefik/traefik -n traefik --create-namespace \
     --set service.type=NodePort --set ports.web.nodePort=30080
   k get ingressclass        # traefik
   ```
2. Deploy `web` and `api` (for example `hashicorp/http-echo` with different `-text` args, listening on 5678) and their Services.
3. Create the Ingress above (without TLS) and `curl -H 'Host: shop.example.com' localhost:30080/api`.
4. Create a self-signed cert (`openssl req -x509 -nodes -newkey rsa:2048 -keyout tls.key -out tls.crt -subj "/CN=secure.example.com" -days 30`), `k create secret tls web-tls --cert=tls.crt --key=tls.key`, and add TLS (you would need the websecure port mapped to test it; reading the YAML is enough for the exam).
5. Set a wrong backend Service name and read `k describe ingress` (backend error).

### Check questions

**Q1.** You created an Ingress, but `k get ingress` shows no ADDRESS and nothing routes. The first thing to check:
- A. That an Ingress controller is running and the `ingressClassName` matches its IngressClass
- B. CoreDNS
- C. kube-proxy mode
- D. NetworkPolicy

<details><summary>▶ Answer</summary>

**A.**
</details>

**Q2.** `path: /app` with `pathType: Prefix`. Which request does **not** match?
- A. `/app`
- B. `/app/`
- C. `/app/login`
- D. `/application`

<details><summary>▶ Answer</summary>

**D.** Prefix matching works by path element.
</details>

**Q3.** Where must the TLS Secret for an Ingress live?
- A. kube-system
- B. The controller's namespace
- C. The same namespace as the Ingress
- D. Any namespace

<details><summary>▶ Answer</summary>

**C.**
</details>

---

## 5.6 Gateway API [`gateway-api`]

### Key ideas
Gateway API (`gateway.networking.k8s.io/v1`) is the successor to Ingress. It is role-oriented, more expressive, and installed as **CRDs** plus a controller (Envoy Gateway, NGINX Gateway Fabric, Istio, Cilium, Traefik, cloud providers…).

| Resource | Owner | Purpose |
|---|---|---|
| **GatewayClass** (cluster-scoped) | infra provider | names the controller (`controllerName`) |
| **Gateway** | cluster operator | an actual load balancer: `gatewayClassName`, `listeners` (name, protocol HTTP/HTTPS/TLS/TCP, port, hostname, `tls.certificateRefs`, `allowedRoutes`) |
| **HTTPRoute** (also GRPCRoute, TLSRoute…) | app team | `parentRefs` (which Gateway/listener), `hostnames`, `rules[]` with `matches` (path, headers, query, method), `filters` (RequestHeaderModifier, RequestRedirect, URLRewrite, RequestMirror), `backendRefs` (Service + port, **weight** for traffic splitting) |
| **ReferenceGrant** | target namespace owner | allows a route to reference a Service (or a Gateway to reference a Secret) in **another** namespace |

Status matters: a Gateway reports `Accepted` and `Programmed`; a route reports `Accepted` and `ResolvedRefs` per parent. Read `k describe` for the reasons.

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata: {name: web-gw, namespace: infra}
spec:
  gatewayClassName: eg
  listeners:
    - name: http
      protocol: HTTP
      port: 80
      allowedRoutes:
        namespaces: {from: All}          # Same (default) | All | Selector
    - name: https
      protocol: HTTPS
      port: 443
      hostname: "*.example.com"
      tls:
        mode: Terminate
        certificateRefs: [{kind: Secret, name: wildcard-tls}]
      allowedRoutes: {namespaces: {from: All}}
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata: {name: shop, namespace: shop}
spec:
  parentRefs:
    - {name: web-gw, namespace: infra, sectionName: http}
  hostnames: [shop.example.com]
  rules:
    - matches: [{path: {type: PathPrefix, value: /api}}]
      backendRefs: [{name: api, port: 8080}]
    - matches: [{path: {type: PathPrefix, value: /}}]
      backendRefs:                                   # canary: 90/10 split
        - {name: web-v1, port: 80, weight: 90}
        - {name: web-v2, port: 80, weight: 10}
    - matches: [{headers: [{name: x-beta, value: "true"}]}]
      backendRefs: [{name: web-v2, port: 80}]
---
# HTTP → HTTPS redirect
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata: {name: redirect, namespace: shop}
spec:
  parentRefs: [{name: web-gw, namespace: infra, sectionName: http}]
  hostnames: [secure.example.com]
  rules:
    - filters:
        - type: RequestRedirect
          requestRedirect: {scheme: https, statusCode: 301}
```

**Ingress → Gateway API mapping** (a likely task: "migrate this Ingress"):

| Ingress | Gateway API |
|---|---|
| `ingressClassName` | Gateway `gatewayClassName` (+ the route's `parentRefs` to the Gateway) |
| `rules[].host` | HTTPRoute `hostnames` |
| `paths[].path` + `pathType: Prefix/Exact` | `matches[].path.type: PathPrefix/Exact` + `value` |
| `backend.service.name/port.number` | `backendRefs[].name` / `port` |
| `tls[].secretName` | Gateway listener `tls.certificateRefs` (HTTPS listener) |

```bash
k get gatewayclass ; k get gateway -A ; k get httproute -A
k describe gateway web-gw -n infra        # listener status, attached routes, address
k describe httproute shop -n shop         # Accepted / ResolvedRefs conditions
k api-resources --api-group=gateway.networking.k8s.io
k explain httproute.spec.rules.matches
```

### Lab (Envoy Gateway quickstart; check the current version on gateway.envoyproxy.io)
1. Install it (this also installs the Gateway API CRDs):
   ```bash
   helm install eg oci://docker.io/envoyproxy/gateway-helm --version <latest> -n envoy-gateway-system --create-namespace
   k wait --timeout=5m -n envoy-gateway-system deploy/envoy-gateway --for=condition=Available
   ```
2. Create the GatewayClass:
   ```yaml
   apiVersion: gateway.networking.k8s.io/v1
   kind: GatewayClass
   metadata: {name: eg}
   spec: {controllerName: gateway.envoyproxy.io/gatewayclass-controller}
   ```
3. Create the `web-gw` Gateway (HTTP listener only) and the `shop` HTTPRoute with two backends (`hashicorp/http-echo` printing `v1` / `v2`).
4. Reach it: the Gateway's LoadBalancer Service stays pending on kind, so port-forward to the generated Envoy Service:
   ```bash
   SVC=$(k get svc -n envoy-gateway-system -l gateway.envoyproxy.io/owning-gateway-name=web-gw -o name)
   k -n envoy-gateway-system port-forward $SVC 8888:80 &
   for i in $(seq 20); do curl -s -H 'Host: shop.example.com' localhost:8888/; done | sort | uniq -c   # about 18 v1 / 2 v2
   ```
5. Move the route's backend Service into a different namespace and read the `ResolvedRefs: False (RefNotPermitted)` condition. Fix it with a ReferenceGrant:
   ```yaml
   apiVersion: gateway.networking.k8s.io/v1beta1
   kind: ReferenceGrant
   metadata: {name: allow-shop, namespace: backend}
   spec:
     from: [{group: gateway.networking.k8s.io, kind: HTTPRoute, namespace: shop}]
     to:   [{group: "", kind: Service}]
   ```
6. Convert your Ingress from 5.5 into a Gateway + HTTPRoute using the mapping table.

### Pitfalls
- Route in namespace `shop`, Gateway in `infra` with the default `allowedRoutes: Same` → the route is not accepted. Use `from: All` or `Selector`.
- `sectionName` must match a listener `name`.
- Gateway API CRDs come in "standard" and "experimental" channels. TCPRoute/TLSRoute are experimental; HTTPRoute, GRPCRoute, Gateway, GatewayClass and ReferenceGrant are standard.

### Check questions

**Q1.** Which resource specifies the controller implementation?
- A. Gateway
- B. GatewayClass
- C. HTTPRoute
- D. IngressClass

<details><summary>▶ Answer</summary>

**B.** (`spec.controllerName`)
</details>

**Q2.** An HTTPRoute in namespace `app` references Gateway `gw` in namespace `infra`. The route status shows `NotAllowedByListeners`. Fix?
- A. Create a ReferenceGrant in `app`
- B. Set the listener's `allowedRoutes.namespaces.from` to `All` (or a Selector that matches `app`)
- C. Move the Service to `infra`
- D. Add `ingressClassName`

<details><summary>▶ Answer</summary>

**B.** ReferenceGrant is for referencing *backends or Secrets* across namespaces, not for attaching to a Gateway.
</details>

**Q3.** How do you send 20% of traffic to `web-v2` and 80% to `web-v1` with Gateway API?
- A. Two HTTPRoutes with the same hostname
- B. One rule with two `backendRefs` with `weight: 80` and `weight: 20`
- C. A `RequestMirror` filter
- D. Not possible

<details><summary>▶ Answer</summary>

**B.**
</details>

**Q4.** An Ingress uses `tls.secretName: shop-tls`. Where does this go in Gateway API?
- A. HTTPRoute `tls`
- B. Gateway listener with `protocol: HTTPS` and `tls.certificateRefs: [{name: shop-tls}]`
- C. GatewayClass parameters
- D. A ReferenceGrant

<details><summary>▶ Answer</summary>

**B.**
</details>

---

## Chapter 5: exam-style tasks

**T5.1 (4 min)** Deployment `echo` in namespace `net` listens on 5678. Expose it as Service `echo-svc` of type NodePort on port 80 with nodePort `30570`. Verify from inside the cluster by Service name.

<details><summary>▶ Solution</summary>

```bash
k expose deploy echo -n net --name=echo-svc --port=80 --target-port=5678 --type=NodePort $do > svc.yaml
# add nodePort: 30570 under ports[0]
k apply -f svc.yaml
k run tmp -n net --image=busybox:1.36 --rm -it --restart=Never -- wget -qO- echo-svc
```
</details>

**T5.2 (7 min)** In namespace `secure`: deny all ingress and egress for all Pods. Then allow Pods labelled `role=frontend` to reach Pods labelled `role=backend` on TCP 8080, and allow all Pods DNS egress.

<details><summary>▶ Solution</summary>

Apply templates 1 and 2 from 5.4 in `secure`. Then:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata: {name: backend-from-frontend, namespace: secure}
spec:
  podSelector: {matchLabels: {role: backend}}
  policyTypes: [Ingress]
  ingress:
    - from: [{podSelector: {matchLabels: {role: frontend}}}]
      ports: [{protocol: TCP, port: 8080}]
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata: {name: frontend-to-backend, namespace: secure}
spec:
  podSelector: {matchLabels: {role: frontend}}
  policyTypes: [Egress]
  egress:
    - to: [{podSelector: {matchLabels: {role: backend}}}]
      ports: [{protocol: TCP, port: 8080}]
```
Both are needed, because egress is also default-denied.
</details>

**T5.3 (5 min)** Create Ingress `shop` in namespace `shop` (class `traefik`): `shop.local/` → Service `web:80`, `shop.local/api` → Service `api:8080`, both Prefix.

<details><summary>▶ Solution</summary>

```bash
k create ingress shop -n shop --class=traefik --rule="shop.local/*=web:80" --rule="shop.local/api*=api:8080"
k describe ingress shop -n shop
```
</details>

**T5.4 (8 min)** An existing Ingress `legacy` (namespace `shop`, host `legacy.local`, path `/` Prefix → `web:80`) must be migrated to Gateway API. A GatewayClass `eg` exists. Create Gateway `legacy-gw` in `shop` with an HTTP listener on port 80 and an HTTPRoute `legacy` with the same routing. Then delete the Ingress.

<details><summary>▶ Solution</summary>

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata: {name: legacy-gw, namespace: shop}
spec:
  gatewayClassName: eg
  listeners: [{name: http, protocol: HTTP, port: 80}]
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata: {name: legacy, namespace: shop}
spec:
  parentRefs: [{name: legacy-gw}]
  hostnames: [legacy.local]
  rules:
    - matches: [{path: {type: PathPrefix, value: /}}]
      backendRefs: [{name: web, port: 80}]
```
```bash
k describe httproute legacy -n shop    # Accepted=True, ResolvedRefs=True
k delete ingress legacy -n shop
```
</details>

**T5.5 (5 min)** Pods can't resolve `*.corp.internal`. The corporate DNS server is `10.10.0.53`. Configure CoreDNS to forward that domain there, and verify that cluster DNS still works.

<details><summary>▶ Solution</summary>

```bash
k -n kube-system edit cm coredns
# append a server block after the main `.:53 { ... }` block:
# corp.internal:53 {
#     forward . 10.10.0.53
# }
k -n kube-system rollout restart deploy coredns     # or wait for `reload`
k run tmp --image=busybox:1.36 --rm -it --restart=Never -- nslookup kubernetes.default
```
</details>
