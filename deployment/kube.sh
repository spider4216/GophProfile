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
kubectl apply -f deployment/rabbitmq/rabbitmq-deployment.yaml -n gophprofile


# Проброс портов (cheetsheet)
# kubectl port-forward service/minio 9001:9001 -n gophprofile