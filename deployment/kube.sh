#!/bin/sh

# Меняем контекст
kubectl config use rancher-desktop

# Создаем пространство имен для сервиса
kubectl create ns gophprofile

kubectl apply -f deployment/postgres/deployment.yaml --namespace=gophprofile
kubectl apply -f deployment/postgres/service.yaml --namespace=gophprofile
kubectl apply -f deployment/postgres/configmap.yaml --namespace=gophprofile