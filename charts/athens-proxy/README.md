# Athens Proxy Helm Chart

## What is Athens?

[Athens](https://docs.gomods.io) is a repository for packages used by your go packages.

Athens provides a repository for [Go Modules](https://github.com/golang/go/wiki/Modules) that you can run. It serves public code and your private code for you, so you don't have to pull directly from a version control system (VCS) like GitHub or GitLab.

## Prerequisites

* Kubernetes 1.10+

## Requirements

- A running Kubernetes cluster
- [Kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) installed and setup to use the cluster
- [Helm](https://helm.sh/) [installed](https://github.com/helm/helm#install) and setup to use the cluster (helm init) or [Tillerless Helm](https://github.com/rimusz/helm-tiller)

## Deploy Athens

The fastest way to install Athens using Helm is to deploy it from our public Helm chart repository. First, add the repository with this command:

```console
$ helm repo add gomods https://athens.blob.core.windows.net/charts
$ helm repo update
```

Next, install the chart with default values to `athens` namespace:  

```
$ helm install gomods/athens-proxy -n athens --namespace athens
```

This will deploy a single Athens instance in the `athens` namespace with `disk` storage enabled. Additionally, a `ClusterIP` service will be created.


## Advanced Configuration

For more advanced configuration options please check Athens [docs](https://docs.gomods.io/install/install-on-kubernetes/#advanced-configuration).

Available options:
- [Replicas](https://docs.gomods.io/install/install-on-kubernetes/#replicas)
- [Access to private repositories via Github](https://docs.gomods.io/install/install-on-kubernetes/#give-athens-access-to-private-repositories-via-github-token-optional)
- [Storage Providers](https://docs.gomods.io/install/install-on-kubernetes/#storage-providers)
- [Kubernetes Service](https://docs.gomods.io/install/install-on-kubernetes/#kubernetes-service)
- [Ingress Resource](https://docs.gomods.io/install/install-on-kubernetes/#ingress-resource)
- [Upstream module repository](https://docs.gomods.io/install/install-on-kubernetes/#upstream-module-repository)
- [.netrc file support](https://docs.gomods.io/install/install-on-kubernetes/#netrc-file-support)
- [gitconfig support](https://docs.gomods.io/install/install-on-kubernetes/#gitconfig-support)

### [AWS Load Balancer Controller](https://kubernetes-sigs.github.io/aws-load-balancer-controller/latest/) ssl redirect using annotation
Using `.Values.ingress.extraPaths` you can define extra paths to the each hosts. So now we can define
extraPaths to `use-annotation` like it mentioned [here](https://kubernetes-sigs.github.io/aws-load-balancer-controller/latest/guide/tasks/ssl_redirect/)

The example below shows how to setup ssl-redirect for AWS LoadBalancer Controller.

```yaml
ingress:
  enabled: true
  annotations:
    alb.ingress.kubernetes.io/actions.ssl-redirect: '{"Type": "redirect", "RedirectConfig": { "Protocol": "HTTPS", "Port": "443", "StatusCode": "HTTP_301"}}'
    # legalone.io wildcard
    alb.ingress.kubernetes.io/certificate-arn: ACM ARN
    alb.ingress.kubernetes.io/listen-ports: '[{"HTTP": 80}, {"HTTPS":443}]'
    alb.ingress.kubernetes.io/scheme: internet-facing
    alb.ingress.kubernetes.io/tags: Service=athens-proxy,Environment=k-production
    alb.ingress.kubernetes.io/target-type: instance
    kubernetes.io/ingress.class: alb
  # Provide an array of values for the ingress host mapping
  hosts:
    - athens-proxy.host
  extraPaths:
    # This will be added to each hosts
    - service: ssl-redirect
      port: use-annotation
      path: /*
```
### Pass extra configuration environment variables

You can pass any extra environment variables supported in [config.dev.toml](../../../config.dev.toml).
The example below shows how to set username/password for basic auth:

```yaml
configEnvVars:
  - name: BASIC_AUTH_USER
    value: "some_user"
  - name: BASIC_AUTH_PASS
    value: "some_password"
```

### Private git servers over ssh support

One or more of git servers can added to `sshGitServers`, and the corresponding config files (git config and ssh config) and ssh keys will be created. Athens then will use these configs and keys to download the source from the git servers.

```yaml
sshGitServers: 
  ## Private git servers over ssh
  ## to enable uncomment lines with single hash below
  ## hostname of the git server
  - host: git.example.com
    ## ssh username
    user: git
    ## ssh private key for the user
    privateKey: |
      -----BEGIN RSA PRIVATE KEY-----
      ...
      -----END RSA PRIVATE KEY-----
    ## ssh port
    port: 22
```
