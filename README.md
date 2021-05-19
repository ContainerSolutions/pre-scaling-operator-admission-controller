# Admission controller for the Pre-Scaling Operator

This admission controller can be used in combination with the [Pre-Scaling operator](https://github.com/ContainerSolutions/pre-scaling-operator) to ensure smooth, frictionless and secure operations. 

The primary purpose of this controller is to block update events on workloads that have opted in to be managed by the operator. The controller blocks only changes on the replicas field and allows all other updates of the object.

## Deploying the controller

The admission controller needs to be deployed separately of the operator. Currently, it's considering resources from the whole cluster, so it doesn't matter in which namespace it is deployed. 

Firstly, the admission controller needs to communicate securely via TLS with the API server. For that, you need to generate the right certs. You can do this by executing the gen_certs.sh script, which will populate the certs directory.

` ./gen_certs.sh `

Then, you'll need to create a Kubernetes secret using the generated certificates and in the same namespace as the admission controller (to be generic enough, let's assume default namespace. Still, try to have a specific namespace for the controller).

```

kubectl create secret generic acsecret -n default \
  --from-file=key.pem=certs/ac-key.pem \
  --from-file=cert.pem=certs/ac-crt.pem

```

After the secret is created, we can apply the manifest.yaml, which includes the service, deployment and ValidatingWebhookConfiguration for the admission controller.

That should be enough to have the controller up and running!