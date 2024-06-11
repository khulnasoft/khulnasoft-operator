The Khulnasoft Security Operator is used to deploy and manage the Khulnasoft Security platform and its components:
* Server (aka “console”)
* Database (optional, you can map an external database as well) 
* Gateway 
* Enforcer (aka “agent”)
* Scanner
* CSP (a simple package that contains the Server, Database, and Gateway)

Use the Khulnasoft-Operator to: 
* Deploy the Khulnasoft Security platform on OpenShift
* Scale up Khulnasoft Security components with extra replicas
* Assign metadata tags to Khulnasoft Security components
* Automatically scale the number of Khulnasoft scanners based on the number of images in the scan queue
	
The Khulnasoft Operator provides a few [Custom Resources](https://github.com/khulnasoft/khulnasoft-operator/tree/master/config/crd) to manage the Khulnasoft platform. 
Please make sure to read the Khulnasoft installation manual (https://docs.khulnasoftsec.com/docs) before using the Operator. 
For advance configurations please consult with Khulnasoft's support team.
    
How to deploy Khulnasoft using the Operator -
1. Install the Khulnasoft Operator from RH's OperatorHub
2. Manage all the prerequisites as covered in the instructions below (see "Before you begin using the Operator CRDs")
3. Use the KhulnasoftCSP CRD to install Khulnasoft in your cluster. KhulnasoftCSP CRD defines how to deploy the Console, Database, Scanner, and Gateway Custom Resources
4. You can also use the KhulnasoftCSP CRD to deploy the default Enforcers, and an OpenShift Route to access the console 
5. You can install the Enforcers using the KhulnasoftEnforcer CRD, but you will need to first get a security token.  Access Khulnasoft console and create a new Enforcer Group. Copy the group's 'token' and use it in the KhulnasoftEnforcer CRD
	

## Before You Begin Using the Operator CRDs
Install the Khulnasoft Operator and obtain access to the Khulnasoft registry - https://www.khulnasoftsec.com/about-us/contact-us/
You will need to supply two secrets during the installation - 
* A secret for the Docker registry
* A secret for the database

You can list the secrets in the YAML files or, you can define secrets in the OpenShift project (see example below) -
```bash
oc create project khulnasoft 
oc create secret docker-registry khulnasoft-registry --docker-server=registry.khulnasoftsec.com --docker-username=<KHULNASOFT_USERNAME> --docker-password=<KHULNASOFT_PASSWORD> --docker-email=<user email> -n khulnasoft
oc create secret generic khulnasoft-database-password --from-literal=db-password=<password> -n khulnasoft
oc secrets add khulnasoft-sa khulnasoft-registry --for=pull -n khulnasoft
```

## Installing KhulnasoftCSP
There are multiple options to deploy the KhulnasoftCSP CR. You can review the different options in the following [file](https://github.com/khulnasoft/khulnasoft-operator/blob/master/config/crd/operator_v1alpha1_khulnasoftcsp_cr.yaml). 

Here is an example of a simple deployment  - 
```yaml
---
apiVersion: operator.khulnasoftsec.com/v1alpha1
kind: KhulnasoftCsp
metadata:
  name: khulnasoft
  namespace: khulnasoft
spec:
  infra:                                    
    serviceAccount: "khulnasoft-sa"               
    namespace: "khulnasoft"                       
    version: "2022.4"                          
    requirements: true                      
  common:
    imagePullSecret: "khulnasoft-registry"        # Optional: if already created image pull secret then mention in here
    dbDiskSize: 10       
  database:                                 
    replicas: 1                            
    service: "ClusterIP"                    
  gateway:                                  
    replicas: 1                             
    service: "ClusterIP"                    
  server:                                   
    replicas: 1                             
    service: "NodePort" 
  enforcer:                                 # Optional: if defined the Operator will create the default Enforcer 
    enforcerMode: true                      # Defines weather the default enforcer will work in 'enforce' or 'audit' more 
  route: true                               # Optional: if defines and set to true, the operator will create a Route to enable access to the console
```

If you haven't used the Route option in the KhulnasoftCsp CRD, you should define the Route manually to enable external access to Khulnasoft's console.

## Installing KhulnasoftEnforcer
If you haven't deployed the enforcer yet, or if you want to deploy additional enforcers, please follow the instruction below:
You can review the different options to implement KhulnasoftEnforcer in the following [file](https://github.com/khulnasoft/khulnasoft-operator/blob/master/config/crd/operator_v1alpha1_khulnasoftenforcer_cr.yaml).

Here is an example of a simple deployment  - 
```yaml
---
apiVersion: operator.khulnasoftsec.com/v1alpha1
kind: KhulnasoftEnforcer
metadata:
  name: khulnasoft
spec:
  infra:                                    
    serviceAccount: "khulnasoft-sa"                
    version: "2022.4"                          # Optional: auto generate to latest version
  common:
    imagePullSecret: "khulnasoft-registry"            # Optional: if already created image pull secret then mention in here
  deploy:                                   # Optional: information about khulnasoft enforcer deployment
    image:                                  # Optional: if not given take the default value and version from infra.version
      repository: "enforcer"                # Optional: if not given take the default value - enforcer
      registry: "registry.khulnasoftsec.com"      # Optional: if not given take the default value - registry.khulnasoftsec.com
      tag: "2022.4"                            # Optional: if not given take the default value - 4.5 (latest tested version for this operator version)
      pullPolicy: "IfNotPresent"            # Optional: if not given take the default value - IfNotPresent
  gateway:                                  # Required: data about the gateway address
    host: khulnasoft-gateway
    port: 8443
  token: "<<your-token>>"                            # Required: enforcer group token also can use an existing secret instead (you can create a token from Khulnasoft's console)
```
