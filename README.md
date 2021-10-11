# K3s setup

My notes on how to setup a single node k3s cluster, with a local docker registry outside the cluster, and installing istio, knative, prometheus, etc.

The setup will run a docker container outside the k3s cluster as the local docker registry, shown below

```bash
$ docker ps
CONTAINER ID        IMAGE               COMMAND                  CREATED             STATUS              PORTS                    NAMES
3dedcfb8c3d7        registry:latest     "/entrypoint.sh /etc…"   14 minutes ago      Up 14 minutes       0.0.0.0:5000->5000/tcp   registry
```

and will install k3s, k9s, Istio, and Knative

![alt text](k3s-setup.png "Screentshot of all the pods running after the setup")

## 1. Pre-requisites

* Assumes a debian 9 VM named `paarijaat-debian-vm`.

* Ensure `docker` is installed and you can access docker without `sudo`

* Ensure that the `/etc/hosts` files is correctly set:

```bash
$ cat /etc/hosts
127.0.0.1       localhost
10.0.2.15 paarijaat-debian-vm paarijaat-debian-vm
```

* Create a directory for k3s:

```bash
mkdir k3s
cd k3s
```

## 2. Create a local docker registry with TLS and password protection (OPTIONAL)

### Creating SSL certificate for docker registry

Notes below follow instructions from: 
* [Deploy a registry server](https://docs.docker.com/registry/deploying/)
* [Test an insecure registry](https://docs.docker.com/registry/insecure/)
* [How to use imagePullSecrets cluster-wide??](https://devopstales.github.io/kubernetes/k8s-imagepullsecret-patcher/)
* <https://fabianlee.org/2018/02/17/ubuntu-creating-a-trusted-ca-and-san-certificate-using-openssl-on-ubuntu/>
* <https://www.linuxtechi.com/setup-private-docker-registry-kubernetes/>

### Create password authentication

```bash
mkdir auth
docker run --entrypoint htpasswd httpd:2 -Bbn paarijaat paari > auth/htpasswd
```

### Add registry to insecure-registries

```bash
$ sudo vi /etc/docker/daemon.json
{
  "tls": false,
  "insecure-registries":["paarijaat-debian-vm:5000"]
}
```

### Geneate self-signed CA cert

```bash
$ openssl req \
  -newkey rsa:4096 -nodes -sha256 -keyout certs/registry.key \
  -addext "subjectAltName = DNS:paarijaat-debian-vm" \
  -x509 -days 3650 -out certs/registry.crt
# use CN=paarijaat-debian-vm
# use email=paarijaat@gmail.com

$ openssl x509  -noout -text -in ./certs/registry.crt
# check if 
#      X509v3 Basic Constraints: critical
#                CA:TRUE
```

### Copy the certificate to trusted ca-certificates

```bash
sudo cp certs/registry.crt /usr/local/share/ca-certificates/paarijaat-debian-vm.crt
sudo update-ca-certificates
```

### Start docker registry container using all of the above

```bash
$ mkdir registry
$ docker container stop registry
$ docker rm registry

$ docker run -d \
  -p 5000:5000 \
  --restart=always \
  --name registry \
  -v "$(pwd)"/auth:/auth \
  -e "REGISTRY_AUTH=htpasswd" \
  -e "REGISTRY_AUTH_HTPASSWD_REALM=Registry Realm" \
  -e REGISTRY_AUTH_HTPASSWD_PATH=/auth/htpasswd \
  -v "$(pwd)"/certs:/certs \
  -e REGISTRY_HTTP_TLS_CERTIFICATE=/certs/registry.crt \
  -e REGISTRY_HTTP_TLS_KEY=/certs/registry.key \
  -v "$(pwd)"/registry:/var/lib/registry \
  registry:latest
```

### Copy cert to docker's trusted certs

```bash
sudo mkdir -p /etc/docker/certs.d/paarijaat-debian-vm:5000
sudo cp certs/registry.crt /etc/docker/certs.d/paarijaat-debian-vm:5000/ca.crt

# restart docker 
sudo systemctl daemon-reload
sudo systemctl restart docker
```

### Check registry by logging in via docker client and curl

```bash
docker login -u paarijaat -p paari paarijaat-debian-vm:5000
# The following should not give you a SSL error
curl https://paarijaat:paari@paarijaat-debian-vm:5000/v2/_catalog
```

### Push some docker images to local registry

```bash
docker pull paarijaat/helloworld-python
docker tag paarijaat/helloworld-python paarijaat-debian-vm:5000/paarijaat/helloworld-python
docker push paarijaat-debian-vm:5000/paarijaat/helloworld-python
curl https://paarijaat:paari@paarijaat-debian-vm:5000/v2/paarijaat/helloworld-python/manifests/latest

docker pull paarijaat/ubuntu
docker tag paarijaat/ubuntu:latest paarijaat-debian-vm:5000/paarijaat/ubuntu:latest
docker push paarijaat-debian-vm:5000/paarijaat/ubuntu:latest
```

## 3. Setup a K3s cluster

### 3.1 Start a single node k3s cluster

```bash
./setup-new-k3s.sh
```

The above script executes the following commands (shown for reference):

```bash
#!/bin/sh

set -ex

echo "Removing k3s"
/usr/local/bin/k3s-killall.sh || true
sleep 2
/usr/local/bin/k3s-uninstall.sh || true

sleep 5
echo "Downloading k3s-setup.sh"
curl -sfL https://get.k3s.io > k3s-setup.sh
chmod +x k3s-setup.sh

echo "Installing single node k3s cluster"
INSTALL_K3S_EXEC='--disable=traefik --disable metrics-server --write-kubeconfig-mode "0644"' ./k3s-setup.sh

echo "Copying k3s config file to ~/.kube/config.k3s"
cp /etc/rancher/k3s/k3s.yaml ~/.kube/config.k3s

echo "Overwriting ~/.kube/config file. Original file stored at ~/kube/config.bak"
cp ~/.kube/config ~/.kube/config.bak || true
cp ~/.kube/config.k3s ~/.kube/config
```

### 3.2 install K9s, lightweight shell GUI (OPTIONAL)

Makes life easy while browsing your k8s setup. Helps you avoid typing kubectl commands for basic 'read' operations.

Downloads available at: <https://github.com/derailed/k9s/releases>

```bash
mkdir k9s; cd k9s
wget https://github.com/derailed/k9s/releases/download/v0.24.14/k9s_Linux_x86_64.tar.gz
tar -zxvf k9s_Linux_x86_64.tar.gz
sudo cp k9s /usr/local/bin/.
mkdir -p ~/.k9s
cp ./.k9s/config.yml ~/.k9s/.   # OPTIONAL files available at: https://github.com/paarijaat/k3s-setup/tree/main/k9s/.k9s
cp ./.k9s/views.yml ~/.k9s/.    # OPTIONAL files available at: https://github.com/paarijaat/k3s-setup/tree/main/k9s/.k9s
k9s
```

### 3.3 Test if LoadBalancer service is working (OPTIONAL)

k3s can create loadbalancer services automatically by deploying a pod that exposes a hostPort and forwards requests to the service cluster ip address

Create `nginx-lb-service.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: nginx
  name: nginx-deploy
spec:
  replicas: 2
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - image: nginx
        name: nginx
---
apiVersion: v1
kind: Service
metadata:
  name: nginx-deploy
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: 80
  selector:
    app: nginx
```

```bash
kubectl apply -f nginx-lb-service.yaml
curl http://localhost:80
kubectl delete -f nginx-lb-service.yaml
```

### 3.4 Test if `local-path-provisioner` is working (OPTIONAL)

`local-path-provisioner` is installed automatically and it set as the default storageclass

```bash
# these commands are not needed

# This installs the local-path-provisioner
$ kubectl apply -f https://raw.githubusercontent.com/rancher/local-path-provisioner/master/deploy/local-path-storage.yaml

# set the local-path-provisioner as the default storage class
$ kubectl patch storageclass local-path -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
```

#### Check local-path-provisioner by creating a volume and a pod that uses it

```bash
wget https://raw.githubusercontent.com/rancher/local-path-provisioner/master/examples/pvc/pvc.yaml
```

Edit `pvc.yaml` to remove the storageclass line:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: local-path-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 128Mi
```

```bash
kubectl apply -f pvc.yaml
kubectl create -f https://raw.githubusercontent.com/rancher/local-path-provisioner/master/examples/pod/pod.yaml
kubectl get pvc
  # should show status as BOUND

kubectl delete -f https://raw.githubusercontent.com/rancher/local-path-provisioner/master/examples/pod/pod.yaml
kubectl delete -f pvc.yaml
```

### 3.5 Check if k8s can pull from the local registry (OPTIONAL)

#### Create imagePullSecret credential

**_IMPORTANT: the secret should have the hostname with port number_**

```bash
$ kubectl create secret docker-registry regcred --docker-server=paarijaat-debian-vm:5000 --docker-username=paarijaat --docker-password=paari --docker-email=paarijaat@gmail.com

# Add the imagePullSecrets to the default service account in the default namespace
kubectl patch serviceaccount default -p "{\"imagePullSecrets\": [{\"name\": \"regcred\"}]}"

# To use this secret in k8s resources, e.g. Deployments, 
# the following should be added parallel to the 'containers' spec 
imagePullSecrets:
- name: regcred
```

#### Create a sample deployment and service to pull from local registy

```bash
$ curl https://paarijaat:paari@paarijaat-debian-vm:5000/v2/_catalog
# {"repositories":["paarijaat/helloworld-python","paarijaat/ubuntu"]}
```

Create `helloworld-python-local-docker-registry.yaml`:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: helloworld
  labels:
    app: helloworld
    service: helloworld
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: 8080
    name: http
  selector:
    app: helloworld
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: helloworld-python
  labels:
    app: helloworld

spec:
  replicas: 1
  selector:
    matchLabels:
      app: helloworld
  template:
    metadata:
      labels:
        app: helloworld
    spec:
      imagePullSecrets:
      - name: regcred
      containers:
      - name: helloworld
        image: paarijaat-debian-vm:5000/paarijaat/helloworld-python
        resources:
          requests:
            cpu: "100m"
        imagePullPolicy: IfNotPresent #Always
        env:
          - name: TARGET
            value: "Python Sample v1"
          - name: PORT
            value: "8080"
        ports:
        - containerPort: 8080
```

```bash
$ kubectl apply -f helloworld-python-local-docker-registry.yaml
$ curl http://localhost
Hello Python Sample v1!
$ kubectl delete -f helloworld-python-local-docker-registry.yaml
```

## 4. Install istio

### 4.1 Install Istio
```bash
# NOTE istio 1.10.3 is messed up. kube-inject does not work
$ export ISTIO_VERSION=1.9.7
$ curl -L https://istio.io/downloadIstio | sh -
# After installation, istio will recommend you adding the path to 'istioctl' in your PATH variable. Copy-paste the command it displays
$ export PATH="$PATH:/local/work/k3s-setup/istio-1.9.7/bin"   # NOTE: your directory structure may vary

# I followed instructions at: https://istio.io/latest/docs/setup/install/istioctl/
```

Create `diff-istio-install-no-mtls-no-inject-debug-logs.yaml`. This will disable auto-injection, disable AutoMtls

```yaml
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  meshConfig:
    accessLogFile: /dev/stdout
    enableAutoMtls: false
  values:
    global:
      logging:
        level: default:info
      proxy:
        autoInject: disabled
        logLevel: info
```

```bash
# Dump the default profile, for reference
istioctl profile dump -f diff-istio-install-no-mtls-no-inject-debug-logs.yaml > profile-default-istio-install-no-mtls-no-inject-debug-logs.yaml

# Generate a manifest file from the profile, for reference
istioctl manifest generate -f profile-default-istio-install-no-mtls-no-inject-debug-logs.yaml > manifest-default-istio-install-no-mtls-no-inject-debug-logs.yaml

# Actually install istio
istioctl install -f diff-istio-install-no-mtls-no-inject-debug-logs.yaml

# Verify the install
istioctl verify-install -f profile-default-istio-install-no-mtls-no-inject-debug-logs.yaml

# [NOT NEEDED] Capture cluster configuration for later use with kube-inject, for reference
$ kubectl -n istio-system get cm istio-sidecar-injector  -o jsonpath="{.data.config}" > injectConfigFile.yaml
$ kubectl -n istio-system get cm istio -o jsonpath="{.data.mesh}" > meshConfigFile.yaml
$ kubectl -n istio-system get cm istio-sidecar-injector -o jsonpath="{.data.values}" > valuesFile.json

# [NOT NEEDED] Use kube-inject based on captured configuration
$ istioctl kube-inject -f samples/helloworld/helloworld.yaml \
    --injectConfigFile injectConfigFile.yaml \
    --meshConfigFile meshConfigFile.yaml \
    --valuesFile valuesFile.json
```

### 4.2 Test istio with sample app

```bash
cd samples/helloworld
istioctl kube-inject -f helloworld.yaml -o helloworld-injected.yaml
kubectl apply -f helloworld-injected.yaml
kubectl apply -f helloworld-gateway.yaml
curl http://localhost/hello
```

## 5. Knative

I have followed instructions from:
* [Installing Knative Serving using YAML files](https://knative.dev/docs/admin/install/serving/install-serving-with-yaml/)
* [Image pull secret added to default service account](https://knative.dev/docs/developer/serving/deploying-from-private-registry/)
* [Patching knative deployment controller, service deployed failed with image in local private registry](https://github.com/knative/serving/issues/5126)
* <https://github.com/knative/serving/issues/2136>
* [Mount SSL certificates in the Pod with Kubernetes secret](https://faun.pub/mount-ssl-certificates-in-kubernetes-pod-with-secret-8aca220896e6)
* [How to mount multiple files / secrets into common directory in kubernetes?](https://stackoverflow.com/questions/59079318/how-to-mount-multiple-files-secrets-into-common-directory-in-kubernetes)


### 5.1 Download Knative YAMLs

```bash
export KNATIVE_VERSION=0.24.0
mkdir -p knative-${KNATIVE_VERSION}
cd knative-${KNATIVE_VERSION}
wget https://github.com/knative/serving/releases/download/v${KNATIVE_VERSION}/serving-crds.yaml
wget https://github.com/knative/serving/releases/download/v${KNATIVE_VERSION}/serving-core.yaml
wget https://github.com/knative/net-istio/releases/download/v${KNATIVE_VERSION}/net-istio.yaml
wget https://github.com/knative/serving/releases/download/v${KNATIVE_VERSION}/serving-hpa.yaml
wget https://github.com/knative/serving/releases/download/v${KNATIVE_VERSION}/serving-default-domain.yaml
```

### 5.2 Create a secret for the registry.crt (OPTIONAL)

```bash
cd certs
CA=$(base64 -w0 registry.crt)
kubectl apply -f - <<__EOF__
apiVersion: v1
kind: Secret
metadata:
  name: paarijaat-debian-vm
  namespace: knative-serving
type: Opaque
data:
  paarijaat-debian-vm.crt: ${CA}
__EOF__
```

### 5.3 Edit config maps in `serving-core.yaml`

Update the domain name of knative services

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: config-domain
  namespace: knative-serving
  labels:
    serving.knative.dev/release: "v0.24.0"
  annotations:
    knative.dev/example-checksum: "74c3fc6a"
data:
  10.0.2.15.sslip.io: ""   # <-- THIS   (this should be the loadBalancerIP of istio-ingressgateway)
```

or

```bash
kubectl -n knative-serving patch cm config-domain --type "json" --patch '[{"op":"add","path":"/data","value":{"10.0.2.15.sslip.io": ""}}]'
```

### 5.4 Update the url structure for knative services

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: config-network
  namespace: knative-serving
  labels:
    serving.knative.dev/release: "v0.24.0"
  annotations:
    knative.dev/example-checksum: "eaf5fb3f"
data:
  domainTemplate: "{{.Name}}.{{.Domain}}"    # <-- THIS
  _example: |
```

### 5.5 Set logs level to debug (OPTIONAL, debug logs should only be enabled for debug purposes)

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: config-logging
  namespace: knative-serving
  labels:
    serving.knative.dev/release: "v0.24.0"
  annotations:
    knative.dev/example-checksum: "56c59cba"
data:
  loglevel.autoscaler: "debug"
  loglevel.hpaautoscaler: "debug"
  loglevel.controller: "debug"
  loglevel.queueproxy: "debug"
  loglevel.webhook: "debug"
  loglevel.activator: "debug"
  loglevel.certcontroller: "debug"
  loglevel.istiocontroller: "debug"
  loglevel.nscontroller: "debug"
  _example: |
```

### 5.6 Edit the `Deployment` of the `controller` (OPTIONAL, only needed if you want to pull from local registry)

* to mount the secret containing registry certificate as a volume
* to set the `SSL_CERT_DIR` env variable

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: controller
  namespace: knative-serving
  labels:
    serving.knative.dev/release: "v0.24.0"
spec:
  template:
    spec:
      containers:
        - name: controller
          # This is the Go import path for the binary that is containerized
          # and substituted here.
          image: gcr.io/knative-releases/knative.dev/serving/cmd/controller@sha256:e9b9cc7a2a091194161a7a256d6fc342317080f1a31e1f1a33993a5e1993fdb5
          volumeMounts:  # <<-- THIS
          - name: allcerts
            mountPath: "/etc/certs"
            readOnly: true
          env:
            - name: SSL_CERT_DIR      # <<-- THIS
              value: /etc/certs    # <<-- THIS
      volumes:
      - name: allcerts  # <<-- THIS
        projected:
          sources:
          - secret:
              name: paarijaat-debian-vm
```

or
Use [YAML Validator](https://yamlvalidator.com/) to verify that the json below are correct

```bash
kubectl -n knative-serving patch deployment controller --type "json" --patch '[{
    "op":"add",
    "path":"/spec/template/spec/containers/0/volumeMounts",
    "value":
      [{
        "name": "allcerts",
        "mountPath": "/etc/certs",
        "readOnly": True
      }]
    
  },{
    "op":"add",
    "path":"/spec/template/spec/containers/0/env/-",
    "value":{
      "name": "SSL_CERT_DIR",
      "value": "/etc/certs"
    }
  },{
    "op":"add",
    "path":"/spec/template/spec/volumes",
    "value": [{
      "name": "allcerts",
      "projected": {
        "sources": [{
          "secret": {
            "name": "paarijaat-debian-vm"
          }
        }]
      }
    }]
  }]'
```

### 5.7 [NOT NEEDED] Edit `net-istio.yaml`

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: config-istio
  namespace: knative-serving
  labels:
    serving.knative.dev/release: "v0.24.0"
    networking.knative.dev/ingress-provider: istio
data:
  gateway.knative-serving.knative-ingress-gateway: "istio-ingressgateway.istio-system.svc.cluster.local"
  local-gateway.knative-serving.knative-local-gateway: "istio-ingressgateway.istio-system.svc.cluster.local"
  _example: |
```

or

```bash
kubectl -n knative-serving patch cm config-istio --type "json" --patch '[{"op":"add","path":"/data/gateway.knative-serving.knative-ingress-gateway","value": "istio-ingressgateway.istio-system.svc.cluster.local"},{"op":"add","path":"/data/local-gateway.knative-serving.cluster-local-gateway","value": "istio-ingressgateway.istio-system.svc.cluster.local"}]'
```

### 5.8 Actually install knative

```bash
kubectl create ns knative-serving
kubectl apply -f serving-crds.yaml
kubectl apply -f serving-core.yaml
kubectl apply -f net-istio.yaml
kubectl apply -f serving-hpa.yaml
```


### test knative service (OPTIONAL): 

create `helloworld-python-ksvc.yaml`

```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: helloworld-python
spec:
  template:
    spec:
      imagePullSecrets:
      - name: regcred
      containers:
      - name: helloworld
        image: paarijaat/helloworld-python
        resources:
          requests:
            cpu: "100m"
        imagePullPolicy: IfNotPresent #Always
        env:
          - name: TARGET
            value: "Python Sample v1 Knative"
        ports:
        - containerPort: 8080
```

```bash
# Run from the node or a pod:
$ curl -v http://helloworld-python.10.0.2.15.sslip.io
* Expire in 50 ms for 1 (transfer 0x55ac4db2af50)
* Expire in 50 ms for 1 (transfer 0x55ac4db2af50)
* Expire in 50 ms for 1 (transfer 0x55ac4db2af50)
...
*   Trying 10.0.2.15...
* TCP_NODELAY set
* Expire in 200 ms for 4 (transfer 0x55ac4db2af50)
* Connected to helloworld-python.10.0.2.15.sslip.io (10.0.2.15) port 80 (#0)
> GET / HTTP/1.1
> Host: helloworld-python.10.0.2.15.sslip.io
> User-Agent: curl/7.64.0
> Accept: */*
>
< HTTP/1.1 200 OK
< content-length: 32
< content-type: text/html; charset=utf-8
< date: Fri, 06 Aug 2021 20:51:53 GMT
< server: istio-envoy
< x-envoy-upstream-service-time: 3
<
Hello Python Sample v1 Knative!
* Connection #0 to host helloworld-python.10.0.2.15.sslip.io left intact
```

### [NOT NEEDED, For understanding only] Projected volume test

[Configure a Pod to Use a Projected Volume for Storage](https://kubernetes.io/docs/tasks/configure-pod-container/configure-projected-volume-storage/)

Create two secrets containing cetificates

```bash
CA=$(base64 -w0 registry.crt)
kubectl apply -f - <<__EOF__
apiVersion: v1
kind: Secret
metadata:
  name: paarijaat-debian-vm
  namespace: knative-serving
type: Opaque
data:
  paarijaat-debian-vm.crt: ${CA}
  paarijaat-debian-vm-3.crt: ${CA}
__EOF__

kubectl apply -f - <<__EOF__
apiVersion: v1
kind: Secret
metadata:
  name: paarijaat-debian-vm-2
  namespace: knative-serving
type: Opaque
data:
  paarijaat-debian-vm-2.crt: ${CA}
__EOF__
```

Create `projected-volume-pod.yaml`:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: test-projected-volume
  namespace: knative-serving
spec:
  containers:
  - name: test-projected-volume
    image: busybox
    args:
    - sleep
    - "86400"
    volumeMounts:
    - name: all-in-one
      mountPath: "/etc/certs"
      readOnly: true
  volumes:
  - name: all-in-one
    projected:
      sources:
      - secret:
          name: paarijaat-debian-vm
      - secret:
          name: paarijaat-debian-vm-2
```

```bash
kubectl apply -f projected-volume-pod.yaml
```

Note the files created under /etc/certs

```bash
$ kubectl -n knative-serving exec -it test-projected-volume -- sh

$ cd /etc/certs
$ ls -la
total 4
drwxrwxrwt    3 root     root           120 Aug  6 12:06 .
drwxr-xr-x    1 root     root          4096 Aug  6 12:06 ..
drwxr-xr-x    2 root     root            80 Aug  6 12:06 ..2021_08_06_12_06_28.719942564
lrwxrwxrwx    1 root     root            31 Aug  6 12:06 ..data -> ..2021_08_06_12_06_28.719942564
lrwxrwxrwx    1 root     root            32 Aug  6 12:06 paarijaat-debian-vm-2.crt -> ..data/paarijaat-debian-vm-2.crt
lrwxrwxrwx    1 root     root            30 Aug  6 12:06 paarijaat-debian-vm.crt -> ..data/paarijaat-debian-vm.crt
$ cat paarijaat-debian-vm.crt
-----BEGIN CERTIFICATE-----
MIIGETCCA/mgAwIBAgIUKjhNeMpB6gJc/TD8VrpFnBuPeBwwDQYJKoZIhvcNAQEL
BQAwgYcxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQK
DBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQxHDAaBgNVBAMME3BhYXJpamFhdC1k
ZWJpYW4tdm0xIjAgBgkqhkiG9w0BCQEWE3BhYXJpamFhdEBnbWFpbC5jb20wHhcN
MjEwODA1MDk1MTEyWhcNMzEwODAzMDk1MTEyWjCBhzELMAkGA1UEBhMCQVUxEzAR
BgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoMGEludGVybmV0IFdpZGdpdHMgUHR5
IEx0ZDEcMBoGA1UEAwwTcGFhcmlqYWF0LWRlYmlhbi12bTEiMCAGCSqGSIb3DQEJ
ARYTcGFhcmlqYWF0QGdtYWlsLmNvbTCCAiIwDQYJKoZIhvcNAQEBBQADggIPADCC
AgoCggIBAOJ8MfPhb4D9sCzB6JpK4rAWVkyPmOTaBVpI26+hf0J5LNB9gyi5OZpu
tWzLi/CWlsp0b+vP6qTG2CdK/XbLEqocLwnV6HKkJAdiltJQuZGzOoXNsxWdhQoB
3/ILsdXoUlBZC+lPbqvmu4dcdvc/U2zYk2HxtwGH1VF+AaoqYvIIpC950Wvh3JXk
iQdGTjJGOrl9AcxXJgCcvyvMkURTc8/8dKgtwP7sG5eRak42rKTsUr+oqWXOv3kH
UxcrPX64Ca8j6D7ic+OpQ6JEkvFrs2HddrMD+Bn+EqKTVxstsGmdX9uK/uQMnHPF
4Ipq6L3iEPyOoc+vsRZ23T6l4HvyQIbgS2bVsUfZ5gnRwTEaa4cTZ28HxC8PxK+V
TXr3zX0lpsxsAZujPA6CH/nh9qCY5Cozy5L6vlA2bpDdY1A19PWagWMY5ZFEbGRM
Xs+fEw2VAoyzLlQ84paBJRK9P+zrH+7TBIh1LaUTzIH5d+xjMFGcgtLILDARskYa
UBRnSBiXx7gExozXh41DWHUAUFwE9hOPF3ML9kAKhXwzcX/r1jkb/I9s6r+c2TAo
d1oo9VIHa6udcTWIWNLUwH1VsPgpUlNJiSTLHo+l34R4QiycbJ6dVHuzzXBkVU5g
sx9N7lotFnx/+TiwbQEG3+1UtyT6JPiuBAHHP7X1uwqfoaz9UlKTAgMBAAGjczBx
MB0GA1UdDgQWBBRH//xfOy2WvtNQYkukYwzFT+QzCTAfBgNVHSMEGDAWgBRH//xf
Oy2WvtNQYkukYwzFT+QzCTAPBgNVHRMBAf8EBTADAQH/MB4GA1UdEQQXMBWCE3Bh
YXJpamFhdC1kZWJpYW4tdm0wDQYJKoZIhvcNAQELBQADggIBAJwZzP0/SInF3LUe
fSwLE6efTnin5zEgh9Tszpt5vb7ZhC5Gc6QlZm05CfyyfswAmhFQsLBuCdQGDJ4E
9wu1lg7YItLc9cQWzWNaxgYfij2YOViF6s1sMYUgKTTVTbuj+zDb9IrVQD+wsAtl
JsnG6g4o/q6mhFrCUJccf2LZ0geTwlPmncap/KTqWspkP4RTY9A1qY6yIU5XKOKU
j4YM5ieGBj38FsonSqiBynBiNHxd8AKnBdutF5sFDlOZ/ABiW2yYXgpOt127xP5B
6Gd189HaFibPtDd21W5/XyVSW7eL6oz27uBRSw43yf5h8vyktVv0cjMT2S99hkvC
aH+90ef5UfQS/TZx5TwR4uftF55KYxxMbLh9KFPUQinTuD465jFCOOO9eBRNoqxM
2kSZaF/iwlG9c1+SJuCl2XQ3jx4em5AXhxnN8hXOJRF0cqkIfIVeZwfgOsscALX/
eyvZ/RsZvKvmTJ+fAReeP7Mfe8GoNC2vP6KMzhQ9KhB361JstpzF7MnzlCDaE/+8
yiv7Qamsm/Wl/gNXWvjNAU/pynPUwffPxtlyzChqrJnCpjewrsecCqsvkmpA52Vv
Jej2xW6QvhtVqKIL/jfAvMD9PAt8iOKY4Zy7KZKy1b5+5a14pJKHs/soIGLPmMD7
Dypm5HnNMkpY9unki9WqUXr74fjF
-----END CERTIFICATE-----
```



**_[NOT NEEDED, since we are using a loadBalancer IP. Otherwise we will need to do this]_** In another shell run `sudo kubectl port-forward --address 0.0.0.0 -n istio-system svc/istio-ingressgateway 80:80`

## 6. Prometheus

### 6.1 Install Prometheus
Instructions take from [kube-prometheus-stack](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack#configuration)

```bash
helm repo add stable https://charts.helm.sh/stable
helm repo update
mkdir prometheus; cd prometheus
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
kubectl create ns prometheus
helm install prometheus prometheus-community/kube-prometheus-stack --namespace prometheus --set defaultRules.rules.alertmanager=false --set alertmanager.enabled=false
```

### 6.2 Change grafana service to LoadBalancer service

Edit the grafana service to make it a LoadBalancer service listening on port 82

```bash
kubectl -n prometheus edit svc prometheus-grafana
(changes marked as # <--- THIS)
```

```yaml
  ports:
  - name: service
    nodePort: 30646
    port: 82   # <--- THIS
    protocol: TCP
    targetPort: 3000
  selector:
    app.kubernetes.io/instance: prometheus
    app.kubernetes.io/name: grafana
  sessionAffinity: None
  type: LoadBalancer    # <--- THIS
```


After this you can access grafana at: `http://10.0.2.15:82/`
`user: admin`
`pass: prom-operator`

### 6.3 Access grafana via port forwarding (OPTIONAL)

```bash
kubectl -n prometheus --address 0.0.0.0 port-forward service/prometheus-grafana 3000:82
```

Open browser: `http://localhost:3000/`

---
---
---
## Ignore stuff Below, just backup info not used

**[NOT NEEDED] edit manifest-default-no-inject-no-mtls-debug-logs.yaml:**

```yaml
# THIS IS NOT NEEDED
apiVersion: apps/v1
kind: Deployment
metadata:
  name: istio-ingressgateway
  namespace: istio-system
spec:
  replicas: 3   # <-- ADD THIS  (optional)
  template:
    spec:
      affinity:
        # <-- ADD THIS SECTION --   Optional, Needed only when you are adding hostPort below>
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchLabels:
                app: istio-ingressgateway
            topologyKey: kubernetes.io/hostname

      containers:
        image: docker.io/istio/proxyv2:1.8.2
        name: istio-proxy
        ports:
        - containerPort: 15021
          protocol: TCP
        - containerPort: 8080
          hostPort: 80  # <-- ADD THIS  (optional)
          protocol: TCP
        - containerPort: 8443
          hostPort: 443  # <-- ADD THIS  (optional)
```

* https://docs.docker.com/registry/deploying/
* https://www.linuxtechi.com/setup-private-docker-registry-kubernetes/
* https://docs.docker.com/registry/insecure/
* https://devopstales.github.io/kubernetes/k8s-imagepullsecret-patcher/
* https://fabianlee.org/2018/02/17/ubuntu-creating-a-trusted-ca-and-san-certificate-using-openssl-on-ubuntu/

```bash
openssl req -new -x509 -newkey rsa:2048 -sha256 -nodes -keyout ./certs/registry.key -days 3560 -out ./certs/registry.crt -config ./certs/paarijaat-debian-vm.cnf

openssl req \
-x509 \
-newkey rsa:4096 \
-sha256 \
-days 3560 \
-nodes \
-keyout ./certs/registry.key \
-out ./certs/registry.crt \
-subj '/CN=paarijaat-debian-vm' \
-extensions san \
-config <( \
  echo '[bs_section]'; \
  echo 'CA=true'; \
  echo '[req]'; \
  echo 'distinguished_name=req'; \
  echo '[san]'; \
  echo 'subjectAltName=DNS:paarijaat-debian-vm')

openssl x509  -noout -text -in ./certs/registry.crt


openssl req -newkey rsa:4096 -nodes -sha256 -keyout ./certs/registry.key -x509 -days 365 -out ./certs/registry.crt

Generating a RSA private key
................................................................................................++++
....++++
writing new private key to './certs/registry.key'

You are about to be asked to enter information that will be incorporated
into your certificate request.
What you are about to enter is what is called a Distinguished Name or a DN.
There are quite a few fields but you can leave some blank
For some fields there will be a default value,
If you enter '.', the field will be left blank.
-----
Country Name (2 letter code) [AU]:DE
State or Province Name (full name) [Some-State]:
Locality Name (eg, city) []:
Organization Name (eg, company) [Internet Widgits Pty Ltd]:
Organizational Unit Name (eg, section) []:
Common Name (e.g. server FQDN or YOUR name) []:paarijaat-debian-vm
Email Address []:



docker run -d \
  -p 5000:5000 \
  --restart=always \
  --name registry \
  -v "$(pwd)"/auth:/auth \
  -e "REGISTRY_HTTP_ADDR=0.0.0.0:5000"
  -e "REGISTRY_AUTH=htpasswd" \
  -e "REGISTRY_AUTH_HTPASSWD_REALM=Registry Realm" \
  -e REGISTRY_AUTH_HTPASSWD_PATH=/auth/htpasswd \
  -v "$(pwd)"/certs:/certs \
  -e REGISTRY_HTTP_TLS_CERTIFICATE=/certs/registry.crt \
  -e REGISTRY_HTTP_TLS_KEY=/certs/registry.key \
  -v "$(pwd)"/registry:/var/lib/registry \
  registry:latest


docker run -d \
  -p 5000:5000 \
  --restart=always \
  --name registry \
  -v "$(pwd)"/auth:/auth \
  -e "REGISTRY_AUTH=htpasswd" \
  -e "REGISTRY_AUTH_HTPASSWD_REALM=Registry Realm" \
  -e REGISTRY_AUTH_HTPASSWD_PATH=/auth/htpasswd \
  -v "$(pwd)"/certs:/certs \
  -e REGISTRY_HTTP_TLS_CERTIFICATE=/certs/paarijaat-debian-vm.crt \
  -e REGISTRY_HTTP_TLS_KEY=/certs/paarijaat-debian-vm.key.pem \
  -v "$(pwd)"/registry:/var/lib/registry \
  registry:latest

```
