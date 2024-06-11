## Kubernetes

Support only Kubernetes 1.11+

### Requirements (Optional)
You can create before using the operator but in Kubernetes the operator able to create all the requirements:
* Namespace
* Service Account
* Docker Pull Image Secret
* Khulnasoft Database Password Secret

> Note: We are recommended to use the automatic requirements generate by the operator in Kubernetes

```shell
kubectl create namespace khulnasoft

kubectl create secret docker-registry khulnasoft-registry-secret --docker-server=registry.khulnasoft.com --docker-username=<user name> --docker-password=<password> --docker-email=<user email> -n khulnasoft

kubectl create secret generic khulnasoft-database-password --from-literal=db-password=123456 -n khulnasoft

kubectl create -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: khulnasoft-sa
  namespace: khulnasoft
imagePullSecrets:
- name: khulnasoft-registry-secret
EOF
```

## Openshift

Support only Openshift 3.11+

### Requirements

First of all you need to create:
* Namespace
* Service Account


```shell
oc new-project khulnasoft

oc create serviceaccount khulnasoft-sa -n khulnasoft

oc adm policy add-cluster-role-to-user cluster-reader system:serviceaccount:khulnasoft:khulnasoft-sa -n khulnasoft
oc adm policy add-scc-to-user privileged system:serviceaccount:khulnasoft:khulnasoft-sa -n khulnasoft
oc adm policy add-scc-to-user hostaccess system:serviceaccount:khulnasoft:khulnasoft-sa -n khulnasoft

```