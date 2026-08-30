# TESSERA v0 on kind

Build the local images and load them into kind before applying the manifest:

```bash
docker build --target gateway -t tessera-gateway:dev .
docker build --target mock-model -t tessera-mock-model:dev .
kind load docker-image tessera-gateway:dev tessera-mock-model:dev
kubectl apply -f deploy/k8s/v0.yaml
kubectl port-forward -n tessera svc/gateway 8080:8080
```

The PostgreSQL schema is mounted as an init script. Delete the PostgreSQL pod and its data volume before re-running schema changes in a disposable kind cluster.
