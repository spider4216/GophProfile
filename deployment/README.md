# Шпаргалка по управлению кластером

## Меняем контекст

```bash
kubectl config use rancher-desktop
```

## Создаем пространство имен для сервиса

```bash
kubectl create ns gophprofile
```

## Apply

```bash
kubectl apply -f deployment/postgres/secret.yaml -n gophprofile
```

## Собираем приложение

```bash
docker build -t gophprofile .
docker tag gophprofile gophprofile:1.0
```

## Импортируем собранный образ в кластер

```bash
k3d image import gophprofile:1.0 -c local
```

## Проброс портов для web-морд (при необходимости)

```bash
kubectl port-forward service/minio 9001:9001 -n gophprofile
kubectl port-forward service/rabbitmq 15672:15672 -n gophprofile
kubectl port-forward service/jaeger 16686:16686 -n gophprofile
kubectl port-forward service/prometheus 9090:9090 -n gophprofile
kubectl port-forward service/grafana 3000:3000 -n gophprofile
```

## Проверить доступность из одной поды в другую

```bash
nc -vz postgres 5432
```

## Линтим Chart

```bash
helm lint .
```

## Смотрим шаблон без применения

```bash
helm template gophprofile . -n gophprofile
```

## Установка

```bash
helm upgrade --install gophprofile . --namespace gophprofile --create-namespace
```

## Обновление

```
helm upgrade gophprofile . -n gophprofile
```

## Посмотреть установленный Chart

```bash
helm list -n gophprofile
```

## Удаление релиза

```bash
helm uninstall gophprofile -n gophprofile
```