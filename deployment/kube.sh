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

kubectl apply -f deployment/otel_collector/configmap.yaml -n gophprofile
kubectl apply -f deployment/otel_collector/deployment.yaml -n gophprofile
kubectl apply -f deployment/otel_collector/service.yaml -n gophprofile

# Собираем приложение
docker build -t gophprofile .
docker tag gophprofile gophprofile:1.0
# Смотрим список кластеров
# k3d cluster list
# Импортируем собранный образ в кластер
k3d image import gophprofile:1.0 -c local

# Apply для Worker
kubectl apply -f deployment/worker/configmap.yaml -n gophprofile
kubectl apply -f deployment/worker/deployment.yaml -n gophprofile

# Apply для самого app
kubectl apply -f deployment/app/configmap.yaml -n gophprofile
kubectl apply -f deployment/app/deployment.yaml -n gophprofile
kubectl apply -f deployment/app/service.yaml -n gophprofile
kubectl apply -f deployment/app/ingress.yaml -n gophprofile

# Проброс портов (cheetsheet)
# kubectl port-forward service/minio 9001:9001 -n gophprofile
# kubectl port-forward service/rabbitmq 15672:15672 -n gophprofile
# kubectl port-forward service/jaeger 16686:16686 -n gophprofile
# kubectl port-forward service/prometheus 9090:9090 -n gophprofile
# kubectl port-forward service/grafana 3000:3000 -n gophprofile

# Рестарт
# kubectl rollout restart deployment gophprofile-app -n gophprofile

# Заметки
# Внести "127.0.0.1 gophprofile.loc" в /etc/hosts