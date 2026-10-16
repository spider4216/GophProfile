#!/bin/sh

# Меняем контекст
kubectl config use rancher-desktop

# Создаем пространство имен для сервиса
kubectl create ns gophprofile

# Apply для Postgres
kubectl apply -f deployment/postgres/deployment.yaml --namespace=gophprofile
kubectl apply -f deployment/postgres/service.yaml --namespace=gophprofile
kubectl apply -f deployment/postgres/configmap.yaml --namespace=gophprofile

# Apply для Minio
kubectl apply -f deployment/minio/minio-pvc.yaml --namespace=gophprofile
kubectl apply -f deployment/minio/configmap.yaml --namespace=gophprofile
kubectl apply -f deployment/minio/deployment.yaml --namespace=gophprofile
kubectl apply -f deployment/minio/service.yaml --namespace=gophprofile

# Apply для Rabbitmq
kubectl apply -f deployment/rabbitmq/rabbitmq-pvc.yaml -n gophprofile
kubectl apply -f deployment/rabbitmq/configmap.yaml -n gophprofile
kubectl apply -f deployment/rabbitmq/deployment.yaml -n gophprofile
kubectl apply -f deployment/rabbitmq/service.yaml -n gophprofile

# Apply для node-exporter
kubectl apply -f deployment/node_exporter/deployment.yaml -n gophprofile
kubectl apply -f deployment/node_exporter/service.yaml -n gophprofile

# Apply для Jaeger
kubectl apply -f deployment/jaeger/deployment.yaml -n gophprofile
kubectl apply -f deployment/jaeger/service.yaml -n gophprofile

# Apply для Prometheus
kubectl apply -f deployment/prometheus/configmap.yaml -n gophprofile
kubectl apply -f deployment/prometheus/deployment.yaml -n gophprofile
kubectl apply -f deployment/prometheus/service.yaml -n gophprofile

# Apply для Grafana
kubectl apply -f deployment/grafana/grafana-pvc.yaml -n gophprofile
kubectl apply -k . -n gophprofile

# Apply для Loki
kubectl apply -f deployment/loki/loki-pvc.yaml -n gophprofile
kubectl apply -f deployment/loki/configmap.yaml -n gophprofile
kubectl apply -f deployment/loki/service.yaml -n gophprofile


# Проброс портов (cheetsheet)
# kubectl port-forward service/minio 9001:9001 -n gophprofile
# kubectl port-forward service/rabbitmq 15672:15672 -n gophprofile
# kubectl port-forward service/jaeger 16686:16686 -n gophprofile
# kubectl port-forward service/prometheus 9090:9090 -n gophprofile
# kubectl port-forward service/grafana 3000:3000 -n gophprofile