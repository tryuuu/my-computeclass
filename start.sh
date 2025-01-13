#! /bin/bash
gcloud container clusters get-credentials sreake-intern-tryu-gke --region asia-northeast1
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.16.2/cert-manager.yaml

sleep 5
kubectl apply -k config/certmanager
kubectl apply -k config/default
make deploy
kubectl apply -k config/certmanager
kubectl apply -k config/default
make deploy
kubectl apply -k config/certmanager

kubectl annotate serviceaccount my-computeclass-controller-manager \
    -n my-computeclass-system \
    iam.gke.io/gcp-service-account=174043275706-compute@developer.gserviceaccount.com

sleep 45
kubectl apply -k config/certmanager
kubectl get pods -n my-computeclass-system
kubectl apply -f config/samples/sample.yaml