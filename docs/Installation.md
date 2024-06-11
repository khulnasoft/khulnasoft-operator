# Installation

## Simple Installation

requirements:
* kubectl configured with your cluster

Create khulnasoft namespace

```shell
kubectl create namespace khulnasoft
```

Install Custom CRDs

```shell
kubectl create -f  config/crd/operator.khulnasoft.com_khulnasoftdatabases_crd.yaml 
kubectl create -f  config/crd/operator.khulnasoft.com_khulnasoftgateways_crd.yaml 
kubectl create -f  config/crd/operator.khulnasoft.com_khulnasoftservers_crd.yaml 
kubectl create -f  config/crd/operator.khulnasoft.com_khulnasoftenforcers_crd.yaml
kubectl create -f  config/crd/operator.khulnasoft.com_khulnasoftcsps_crd.yaml
kubectl create -f  config/crd/operator.khulnasoft.com_khulnasoftscanners_crd.yaml
kubectl create -f  config/crd/operator.khulnasoft.com_khulnasoftkubeenforcers_crd.yaml
```

Install operator with version in the [Operator YAML](../config/manifests/operator.yaml)

```shell
kubectl create -f config/rbac/service_account.yaml -n khulnasoft
kubectl create -f config/rbac/role.yaml
kubectl create -f config/rbac/role_binding.yaml
kubectl create -f config/manifests/operator.yaml -n khulnasoft
```
