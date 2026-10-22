# GophProfile

## Запуск в кластере

Приложение настроено под запуск в кластере и упаковано в Helm. Для запуска достаточно сделать следующее:

Собрать образ приложения и воркера

```bash
docker build -t gophprofile .
docker tag gophprofile gophprofile:1.0
```

Заием прокинуть образ в k8s

```bash
k3d image import gophprofile:1.0 -c local
```

Затем запустить установку через helm

```bash
helm upgrade --install gophprofile . --namespace gophprofile --create-namespace
```