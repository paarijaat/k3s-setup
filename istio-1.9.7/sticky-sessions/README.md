# How to create sticky sessions in istio

## References

* [What are sticky sessions and how to configure them with Istio?](https://dev.to/peterj/what-are-sticky-sessions-and-how-to-configure-them-with-istio-1e1a)
* [Introducing the Istio v1alpha3 routing API](https://preliminary.istio.io/latest/blog/2018/v1alpha3-routing/)
* [How to Manage Traffic Using Istio on Kubernetes](https://betterprogramming.pub/how-to-manage-traffic-using-istio-on-kubernetes-cd4b96e00b57)
* [Exploring Istio - The DestinationRule resource](https://octopus.com/blog/istio/istio-destinationrule)
* [Make Pod IPs addressable directly even in the mesh #23494](https://github.com/istio/istio/issues/23494)
* [503 between pod to pod communication (1.5.1)](https://discuss.istio.io/t/503-between-pod-to-pod-communication-1-5-1/6121/16)
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