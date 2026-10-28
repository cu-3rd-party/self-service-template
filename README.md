# self-service-template

Готовый HTTP API для выкладки в cu3rd self-service: сборка контейнера, PostgreSQL,
HTTPS, метрики Prometheus и логи Loki. Основная ветка — `master`.

Развёрнутое приложение: https://self-service-template.dev.cu3rd.ru

## Создать свой сервис

1. Нажмите **Use this template** и создайте репозиторий в `cu3rd` с веткой `master`.
2. Скопируйте `self-service/project.yaml` в `projects/<имя>.yaml` репозитория
   [self-service](https://git.cu3rd.ru/cu3rd/self-service). Поменяйте `name`, `team`,
   `repo`, `image` и домен. Откройте PR и дождитесь merge.
3. В своём репозитории откройте **Settings → Actions → Variables**:

   | Variable | Значение |
   |---|---|
   | `APP_NAME` | Имя вашего приложения |
   | `SELF_SERVICE_PROJECT` | Имя проекта в self-service; по умолчанию имя репозитория |
   | `SELF_SERVICE_ENVIRONMENT` | `dev` по умолчанию |
   | `SELF_SERVICE_APP` | `api` по умолчанию |

4. Получите у платформы `SELF_SERVICE_DELIVERY_TOKEN` и добавьте его в
   **Settings → Actions → Secrets**. Это токен для открытия PR в self-service,
   без доступа к Kubernetes, Postgres admin или Cloudflare.
5. Сделайте push в `master` либо запустите workflow вручную. CI проверит Go-код,
   соберёт и опубликует образ с тегом commit SHA и откроет PR доставки в self-service.
6. После merge PR доставки provisioner применит конфигурацию, Argo CD развернёт
   приложение, cert-manager выпустит сертификат. Статус CI означает публикацию образа
   и создание PR; дождитесь применения self-service и готовности приложения.

Публикация использует организационные `REGISTRY_USER` и `REGISTRY_TOKEN`. Раннеры —
`homelab-docker`, без `ru`. На PR выполняются только тесты, деплой не запускается.
При копировании шаблона содержимое Actions Variables/Secrets не копируется — настройте
их для нового репозитория. Если проекта ещё нет в self-service, delivery завершится
ошибкой; после регистрации проекта запустите workflow повторно.

## Настройки и доступы

- `APP_NAME` приходит из Actions Variable через явный mapping в YAML.
- `DATABASE_URL` создаётся платформой автоматически из `postgres.enabled: true`.
  Его не нужно хранить в Actions Secrets, вручную задавать или выводить в логах.
- Для токенов приложения добавьте mapping `secrets: {MY_TOKEN: MY_TOKEN}` в описание
  приложения и одноимённый Actions Secret. Обновите значение и запустите delivery.
- CPU и память задаются в `values.resources`. Здесь requests и limits — **2m CPU
  и 30Mi памяти**. Go работает с `GOMEMLIMIT=20MiB`, `GOGC=50`, `GOMAXPROCS=1`.
- Приложение работает от UID 10001, без root и service-account token.
- PostgreSQL общий через PgBouncer; приложение держит до двух подключений. `PGSSLMODE`
  по умолчанию `disable` для внутреннего адреса pooler; сетевой путь между узлами
  защищён WireGuard. Для внешней БД задайте соответствующий TLS-режим.

## API

| Путь | Назначение |
|---|---|
| `GET /` | Имя сервиса и статус |
| `GET /api/health` | Проверка HTTP-сервера, не зависит от БД |
| `GET /api/ready` | Реальное подключение к PostgreSQL; 503, если БД недоступна |
| `GET /metrics` | Prometheus: запросы, Go heap и goroutines |

Метрики подключены через `metrics` в описании приложения. Логи пишутся в stdout.
[Дашборд](https://grafana.iamamaev.ru/d/cu3rd-self-service) позволяет выбрать namespace
`cu3rd-self-service-template-dev` и увидеть метрики и логи.

## Локальный запуск

```sh
cp .env.example .env
# Укажите DATABASE_URL для своей локальной БД в .env.
go test -tags nomsgpack ./...
go run -tags nomsgpack .
curl http://localhost:8080/api/health
curl http://localhost:8080/api/ready
```

## Удаление

Удалите соответствующие секции или весь файл проекта в self-service и влейте PR.
Удаление `postgres` удаляет базу **вместе с данными**. Для пересоздания сначала
дождитесь применения удаления, затем верните секцию отдельным PR.
