<h1 align="center">Khulnasoft Security Operator</h1>
<p align="center">
  <img width="150px" height="150px" src="https://avatars.githubusercontent.com/u/43526139?s=400"/>
</p>

## About

The **khulnasoft-operator** is a group of controllers that runs within a Kubernetes or OpenShift cluster. It provides a means to deploy and manage an Khulnasoft Security cluster and components:
* Server (Console)
* Gateway
* Database (not recommended for production environments)
* Server components (package of Server, Gateway and Database)
* Khulnasoft Enforcer
* Scanner CLI

**Use the khulnasoft-operator to:**
 * Deploy Khulnasoft Enterprise components in OpenShift clusters
 * Scale up Khulnasoft Enterprise components with extra replicas
 * Assign metadata tags to Khulnasoft Enterprise components

## Deployment requirements

The Operator is designed for OpenShift clusters.

* **OpenShift:** 4.0 +

## Documentation

The following documentation is available:

- [OpenShift installation and examples](docs/DeployOpenShiftOperator.md)
- [Khulnasoft Enterprise documentation portal](https://docs.khulnasoft.com/)
- [Khulnasoft Security Operator page on OperatorHub.io](https://operatorhub.io/operator/khulnasoft)

## Issues and feedback

If you encounter any problems or would like to give us feedback on deployments, we encourage you to raise issues here on GitHub.
