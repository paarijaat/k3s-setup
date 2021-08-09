# How to create sticky sessions in istio

## References

* [What are sticky sessions and how to configure them with Istio?](https://dev.to/peterj/what-are-sticky-sessions-and-how-to-configure-them-with-istio-1e1a)
* [Introducing the Istio v1alpha3 routing API](https://preliminary.istio.io/latest/blog/2018/v1alpha3-routing/)
* [How to Manage Traffic Using Istio on Kubernetes](https://betterprogramming.pub/how-to-manage-traffic-using-istio-on-kubernetes-cd4b96e00b57)
* [Exploring Istio - The DestinationRule resource](https://octopus.com/blog/istio/istio-destinationrule)
* [Make Pod IPs addressable directly even in the mesh #23494](https://github.com/istio/istio/issues/23494)
* [503 between pod to pod communication (1.5.1)](https://discuss.istio.io/t/503-between-pod-to-pod-communication-1-5-1/6121/16)
* [Making Istio Work with Kubernetes StatefulSet and Headless Services](https://medium.com/airy-science/making-istio-work-with-kubernetes-statefulset-and-headless-services-d5725c8efcc9)
* [Original destination](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/upstream/load_balancing/original_dst#arch-overview-load-balancing-types-original-destination)
* <https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/cluster/v3/cluster.proto#envoy-v3-api-field-config-cluster-v3-cluster-originaldstlbconfig-use-http-header>


## EnvoyFilter references

* [EnvoyFilter Samples](https://github.com/istio/istio/wiki/EnvoyFilter-Samples)
* [EnvoyFilter](https://istio.io/latest/docs/reference/config/networking/envoy-filter/)
* [Istio EnvoyFilter HTTP_ROUTE example](https://stackoverflow.com/questions/67823367/istio-envoyfilter-http-route-example)
* [aeraki-framework/dubbo-envoyfilter-example](https://github.com/aeraki-framework/dubbo-envoyfilter-example/tree/master/istio)
* [An example of configuring aggregate cluster using EnvoyFilter.](https://gist.github.com/howardjohn/95607bc10edf9c5123bebc57d1e5e61c)
* [Hoot - Extending Istio with the EnvoyFilter CRD](https://www.solo.io/wp-content/uploads/2021/01/Extending-Istio-with-the-EnvoyFilter-CRD.pdf)
* [google search - envoyfilter cluster](https://www.google.com/search?q=envoyfilter+cluster)



## The sticky app

```golang
package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

var hostnameCache string
var ipAddressCache string
var targetStringCache string

func init() {
	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	hostnameCache = hostname

	// Get IP address
	conn, error := net.Dial("udp", "8.8.8.8:80")  
	if error != nil {  
		fmt.Println(error)
		os.Exit(1)
	}  
	defer conn.Close()  
	ipAddress := conn.LocalAddr().(*net.UDPAddr).IP 
	ipAddressCache = ipAddress.String()

	targetStringCache = os.Getenv("TARGET")

	fmt.Printf("[%s] Starting, hostname: %s, IP: %s, port: 8080, %s\n", 
		getCurrentTime(), hostnameCache, ipAddressCache, targetStringCache)

}

func main() {
	http.HandleFunc("/", mainHandler)
	http.ListenAndServe(":8080", nil)
}

func getCurrentTime() string {
	dt := time.Now()
	return dt.Format("01-01-2006 15:04:05.000")
}

func mainHandler(res http.ResponseWriter, req *http.Request) {
	outputStr := fmt.Sprintf("[%s] Hello from hostname: %s, IP: %s, %s\n", 
		getCurrentTime(), hostnameCache, ipAddressCache, targetStringCache)
	fmt.Printf(outputStr)
	data := []byte(outputStr)
	res.WriteHeader(200)
	res.Write(data)
}
```

## Sticky app deployment and service
```yaml
apiVersion: v1
kind: Service
metadata:
  name: stickyapp
  labels:
    app: stickyapp
    service: stickyapp
spec:
  #type: LoadBalancer
  ports:
  - port: 8080
    name: http
    targetPort: 8080
  selector:
    app: stickyapp
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: stickyapp
  labels:
    app: stickyapp
spec:
  replicas: 3
  selector:
    matchLabels:
      app: stickyapp
  template:
    metadata:
      labels:
        app: stickyapp
    spec:
      containers:
      - name: stickyapp
        image: paarijaat-debian-vm:5000/paarijaat/stickyapp
        env:
          - name: TARGET
            value: "Sticky app v1"
        resources:
          requests:
            cpu: "50m"
        imagePullPolicy: IfNotPresent #Always
        ports:
        - containerPort: 8080
```

## Consistent hash loadbalancing

NOTE: This is working.
Create `stickyapp-gateway-consistent.yaml`:

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: Gateway
metadata:
  name: stickyapp-gateway
spec:
  selector:
    istio: ingressgateway # use istio default controller
  servers:
  - port:
      number: 80
      name: http
      protocol: HTTP
    hosts:
    - "*"
---
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: stickyapp-consistent
spec:
  hosts:
  - "*"
  gateways:
  - stickyapp-gateway
  http:
  - match:
    - uri:
        prefix: "/stickyapp"
    rewrite:
      uri: "/"
    route:
    - destination:
        host: stickyapp.default.svc.cluster.local
        port:
          number: 8080
---
apiVersion: networking.istio.io/v1alpha3
kind: DestinationRule
metadata:
    name: stickyapp-consistent
spec:
    host: stickyapp.default.svc.cluster.local
    trafficPolicy:
      loadBalancer:
        consistentHash:
          httpHeaderName: x-user
```

```bash
curl -H "x-user: paarijaat" http://localhost/stickyapp
```


## Selectively Passthrough (hybrid) loadbalancing

Keeping the same `Deployment` and `Service`. Create `stickyapp-gateway-hybrid.yaml`:

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: Gateway
metadata:
  name: stickyapp-gateway
spec:
  selector:
    istio: ingressgateway # use istio default controller
  servers:
  - port:
      number: 80
      name: http
      protocol: HTTP
    hosts:
    - "*"
---
apiVersion: networking.istio.io/v1alpha3
kind: DestinationRule
metadata:
  name: stickyapp-hybrid
spec:
  host: stickyapp.default.svc.cluster.local
  subsets:
  - name: normal
    trafficPolicy:
      loadBalancer:
        simple: ROUND_ROBIN
  - name: direct
    trafficPolicy:
      loadBalancer:
        simple: PASSTHROUGH
---
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: stickyapp-hybrid
spec:
  hosts:
  - "*"
  gateways:
  - stickyapp-gateway
  http:
  - match:
    - uri:
        prefix: "/stickyapp"
      headers:
        use-direct:
          exact: "true"
    rewrite:
      uri: "/"
    route:
    - destination:
        host: stickyapp.default.svc.cluster.local
        subset: direct
        port:
          number: 8080
  - match:
    - uri:
        prefix: "/stickyapp"
    rewrite:
      uri: "/"
    route:
    - destination:
        host: stickyapp.default.svc.cluster.local
        subset: normal
        port:
          number: 8080
```

### EnvoyFilter to enable the use_http_header: true

Create `stickyapp-envoyfilter-hybrid.yaml`:

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: EnvoyFilter
metadata:
  name: stickyapp-envoyfilter-hybrid
  namespace: istio-system
spec:
  configPatches:
  - applyTo: CLUSTER
    match:
      context: GATEWAY
      cluster:
        name: "outbound|8080|direct|stickyapp.default.svc.cluster.local"
    patch:
      operation: MERGE
      value: 
        original_dst_lb_config:
          use_http_header: true
```

```bash
kubectl apply -f stickyapp-envoyfilter-hybrid.yaml
```

### Try out the selectively passthrough loadbalancing

```bash
# this goes to the 'normal' round-robin loadbalanced subset
curl -v http://localhost/stickyapp

# this goes to the 'direct' passthrough loadbalanced subset
$ curl -v -H "use-direct: true" -H "x-envoy-original-dst-host: 10.42.0.94:8080" http://localhost/stickyapp
* Expire in 1 ms for 1 (transfer 0x560f0223ef50)
*   Trying ::1...
* TCP_NODELAY set
* Expire in 149997 ms for 3 (transfer 0x560f0223ef50)
* Expire in 200 ms for 4 (transfer 0x560f0223ef50)
* connect to ::1 port 80 failed: Connection refused
*   Trying 127.0.0.1...
* TCP_NODELAY set
* Expire in 149997 ms for 3 (transfer 0x560f0223ef50)
* Connected to localhost (127.0.0.1) port 80 (#0)
> GET /stickyapp HTTP/1.1
> Host: localhost
> User-Agent: curl/7.64.0
> Accept: */*
> use-direct: true
> x-envoy-original-dst-host: 10.42.0.94:8080
>
< HTTP/1.1 200 OK
< date: Mon, 09 Aug 2021 13:39:06 GMT
< content-length: 105
< content-type: text/plain; charset=utf-8
< x-envoy-upstream-service-time: 0
< server: istio-envoy
<
[08-08-2021 15:39:06.957] Hello from hostname: stickyapp-6d7bb596bc-bq4z2, IP: 10.42.0.94, Sticky app v1
* Connection #0 to host localhost left intact
```

## Inspect Istio and Envoy configurations (for learning only)

[Using the Istioctl Command-line Tool](https://istio.io/latest/docs/ops/diagnostic-tools/istioctl/)
[Debugging Envoy and Istiod](https://istio.io/latest/docs/ops/diagnostic-tools/proxy-cmd/)


### Test istio with sample app

```bash
cd sticky-sessions/stickyapp
kubectl apply -f stickyapp.yaml
kubectl apply -f stickyapp-gateway-hybrid.yaml
```

### How to get configuration data from istio
```bash
$ kubectl -n istio-system get pods -o wide
NAME                                    READY   STATUS    RESTARTS   AGE   IP           NODE                  NOMINATED NODE   READINESS GATES
svclb-istio-ingressgateway-9j666        5/5     Running   5          41h   10.42.0.63   paarijaat-debian-vm   <none>           <none>
istiod-68969ddb99-rnkc9                 1/1     Running   1          41h   10.42.0.55   paarijaat-debian-vm   <none>           <none>
istio-ingressgateway-569f8cdc97-mjml7   1/1     Running   1          41h   10.42.0.58   paarijaat-debian-vm   <none>           <none>

$ kubectl -n istio-system get svc
NAME                    TYPE           CLUSTER-IP     EXTERNAL-IP   PORT(S)                                                                      AGE
istiod                  ClusterIP      10.43.96.159   <none>        15010/TCP,15012/TCP,443/TCP,15014/TCP                                        41h
istio-ingressgateway    LoadBalancer   10.43.137.34   10.0.2.15     15021:30691/TCP,80:30974/TCP,443:32669/TCP,15012:32368/TCP,15443:32699/TCP   41h
knative-local-gateway   ClusterIP      10.43.60.15    <none>        80/TCP                                                                       40h

# HostPort entries will show up in iptables as DNAT entries
$ sudo iptables -t nat -L -v --line-numbers | grep DNAT
3        3   132 CNI-HOSTPORT-DNAT  all  --  any    any     anywhere             anywhere             ADDRTYPE match dst-type LOCAL
3      176 10560 CNI-HOSTPORT-DNAT  all  --  any    any     anywhere             anywhere             ADDRTYPE match dst-type LOCAL
4        0     0 DNAT       tcp  --  !docker0 any     anywhere             anywhere             tcp dpt:5000 to:172.17.0.2:5000
2       20  1200 DNAT       tcp  --  any    any     anywhere             anywhere             tcp /* default/kubernetes:https */ to:10.0.2.15:6443
Chain CNI-HOSTPORT-DNAT (2 references)
3        0     0 DNAT       tcp  --  any    any     anywhere             anywhere             tcp dpt:15021 to:10.42.0.63:15021
6        0     0 DNAT       tcp  --  any    any     anywhere             anywhere             tcp dpt:http to:10.42.0.63:80   # <-- THIS (IP address of the pod where connections to port 80 will be forwarded)
9        0     0 DNAT       tcp  --  any    any     anywhere             anywhere             tcp dpt:https to:10.42.0.63:443
...
...
2        0     0 DNAT       tcp  --  any    any     anywhere             anywhere             tcp /* default/helloworld:http */ to:10.42.0.66:5000
2        0     0 DNAT       tcp  --  any    any     anywhere             anywhere             tcp /* default/helloworld:http */ to:10.42.0.67:5000


$ istioctl proxy-status
NAME                                                   CDS        LDS        EDS        RDS        ISTIOD                      VERSION
istio-ingressgateway-569f8cdc97-mjml7.istio-system     SYNCED     SYNCED     SYNCED     SYNCED     istiod-68969ddb99-rnkc9     1.9.7


# Retrieve information about proxy configuration from an Envoy instance.
istioctl proxy-config <clusters|listeners|routes|endpoints|bootstrap|log|secret> <pod-name[.namespace]>


istioctl proxy-config listeners istio-ingressgateway-569f8cdc97-mjml7.istio-system
istioctl proxy-config routes istio-ingressgateway-569f8cdc97-mjml7.istio-system
istioctl proxy-config clusters istio-ingressgateway-569f8cdc97-mjml7.istio-system
# istioctl proxy-config listener helloworld-v1-6f79f9649b-dnd5w.default
```

```bash
istioctl proxy-config cluster istio-ingressgateway-569f8cdc97-mjml7.istio-system -o json > clusters.json
```

Inspect clusters created for stickyapp

```json
[
    {
        "name": "outbound|8080||stickyapp.default.svc.cluster.local",
        "type": "EDS",
        "edsClusterConfig": {
            "edsConfig": {
                "ads": {},
                "resourceApiVersion": "V3"
            },
            "serviceName": "outbound|8080||stickyapp.default.svc.cluster.local"
        },
        "connectTimeout": "10s",
        "circuitBreakers": {
            "thresholds": [
                {
                    "maxConnections": 4294967295,
                    "maxPendingRequests": 4294967295,
                    "maxRequests": 4294967295,
                    "maxRetries": 4294967295
                }
            ]
        },
        "metadata": {
            "filterMetadata": {
                "istio": {
                    "config": "/apis/networking.istio.io/v1alpha3/namespaces/default/destination-rule/stickyapp-hybrid",
                    "default_original_port": 8080,
                    "services": [
                        {
                            "host": "stickyapp.default.svc.cluster.local",
                            "name": "stickyapp",
                            "namespace": "default"
                        }
                    ]
                }
            }
        },
        "filters": [
            {
                "name": "istio.metadata_exchange",
                "typedConfig": {
                    "@type": "type.googleapis.com/udpa.type.v1.TypedStruct",
                    "typeUrl": "type.googleapis.com/envoy.tcp.metadataexchange.config.MetadataExchange",
                    "value": {
                        "protocol": "istio-peer-exchange"
                    }
                }
            }
        ]
    },
    {
        "name": "outbound|8080|direct|stickyapp.default.svc.cluster.local",
        "type": "ORIGINAL_DST",
        "connectTimeout": "10s",
        "lbPolicy": "CLUSTER_PROVIDED",
        "circuitBreakers": {
            "thresholds": [
                {
                    "maxConnections": 4294967295,
                    "maxPendingRequests": 4294967295,
                    "maxRequests": 4294967295,
                    "maxRetries": 4294967295
                }
            ]
        },
        "metadata": {
            "filterMetadata": {
                "istio": {
                    "config": "/apis/networking.istio.io/v1alpha3/namespaces/default/destination-rule/stickyapp-hybrid",
                    "default_original_port": 8080,
                    "services": [
                        {
                            "host": "stickyapp.default.svc.cluster.local",
                            "name": "stickyapp",
                            "namespace": "default"
                        }
                    ],
                    "subset": "direct"
                }
            }
        },
        "filters": [
            {
                "name": "istio.metadata_exchange",
                "typedConfig": {
                    "@type": "type.googleapis.com/udpa.type.v1.TypedStruct",
                    "typeUrl": "type.googleapis.com/envoy.tcp.metadataexchange.config.MetadataExchange",
                    "value": {
                        "protocol": "istio-peer-exchange"
                    }
                }
            }
        ]
    },
    {
        "name": "outbound|8080|normal|stickyapp.default.svc.cluster.local",
        "type": "EDS",
        "edsClusterConfig": {
            "edsConfig": {
                "ads": {},
                "resourceApiVersion": "V3"
            },
            "serviceName": "outbound|8080|normal|stickyapp.default.svc.cluster.local"
        },
        "connectTimeout": "10s",
        "circuitBreakers": {
            "thresholds": [
                {
                    "maxConnections": 4294967295,
                    "maxPendingRequests": 4294967295,
                    "maxRequests": 4294967295,
                    "maxRetries": 4294967295
                }
            ]
        },
        "metadata": {
            "filterMetadata": {
                "istio": {
                    "config": "/apis/networking.istio.io/v1alpha3/namespaces/default/destination-rule/stickyapp-hybrid",
                    "default_original_port": 8080,
                    "services": [
                        {
                            "host": "stickyapp.default.svc.cluster.local",
                            "name": "stickyapp",
                            "namespace": "default"
                        }
                    ],
                    "subset": "normal"
                }
            }
        },
        "filters": [
            {
                "name": "istio.metadata_exchange",
                "typedConfig": {
                    "@type": "type.googleapis.com/udpa.type.v1.TypedStruct",
                    "typeUrl": "type.googleapis.com/envoy.tcp.metadataexchange.config.MetadataExchange",
                    "value": {
                        "protocol": "istio-peer-exchange"
                    }
                }
            }
        ]
    }    
]
```

After applying `stickyapp-envoyfilter-hybrid.yaml`

```bash
kubectl apply -f stickyapp-envoyfilter-hybrid.yaml
istioctl proxy-config cluster istio-ingressgateway-569f8cdc97-mjml7.istio-system -o json > clusters_envoyfilter.json
```

Inspect the output of the `clusters_envoyfilter.json` file. 
You should see the `outbound|8080|direct|stickyapp.default.svc.cluster.local` cluster updated

```json
[
    {
        "name": "outbound|8080|direct|stickyapp.default.svc.cluster.local",
        "type": "ORIGINAL_DST",
        "connectTimeout": "10s",
        "lbPolicy": "CLUSTER_PROVIDED",
        "circuitBreakers": {
            "thresholds": [
                {
                    "maxConnections": 4294967295,
                    "maxPendingRequests": 4294967295,
                    "maxRequests": 4294967295,
                    "maxRetries": 4294967295
                }
            ]
        },

        // NOTE: THIS WAS ADDED because of the EnvoyFilter
        "originalDstLbConfig": {
            "useHttpHeader": true    
        },
        "metadata": {
            "filterMetadata": {
                "istio": {
                    "config": "/apis/networking.istio.io/v1alpha3/namespaces/default/destination-rule/stickyapp-hybrid",
                    "default_original_port": 8080,
                    "services": [
                        {
                            "host": "stickyapp.default.svc.cluster.local",
                            "name": "stickyapp",
                            "namespace": "default"
                        }
                    ],
                    "subset": "direct"
                }
            }
        },
        "filters": [
            {
                "name": "istio.metadata_exchange",
                "typedConfig": {
                    "@type": "type.googleapis.com/udpa.type.v1.TypedStruct",
                    "typeUrl": "type.googleapis.com/envoy.tcp.metadataexchange.config.MetadataExchange",
                    "value": {
                        "protocol": "istio-peer-exchange"
                    }
                }
            }
        ]
    }
]
```
