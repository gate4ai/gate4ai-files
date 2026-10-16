# gate4ai-files — коннектор «Файлы» для gate4.ai

Чтение, список, добавление в конец и запись (по умолчанию — с подтверждением) только в папках из настройки `roots`. Пути проверяет хост: выйти за пределы папок через `..` или симлинки нельзя.

Коннектор — WASM-модуль для хоста [gate4ai](https://github.com/gate4ai/gate4ai): что ему можно, объявлено в [manifest.yaml](manifest.yaml) (`capabilities`) и показывается владельцу при установке; всё остальное хост запрещает (`capability_denied`). Написан на [sdk-go](https://github.com/gate4ai/sdk-go).

## Установка

```sh
gate4ai connectors install files                 # из магазина gate4.ai
# или своя сборка:
gate4ai connectors pack . -o files.g4c && gate4ai connectors install files.g4c --dev
```

Настройки — `connectors.files` в конфигурации gate4ai (секреты — только алиасы Vault, `@имя`).

## Разработка

```sh
go test ./...                                                    # нативно, на тестовом хосте sdk-go/hosttest
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o connector.wasm .
gate4ai connectors install . --watch                             # поставить в свой daemon из исходников и пересобирать на каждое сохранение
```

## Лицензия

[AGPL-3.0-only](LICENSE) с [дополнительным разрешением](LICENSE-EXCEPTION) для хоста gate4ai: коннектор можно исполнять в хосте и распространять вместе с ним, не распространяя AGPL на хост. Изменения самого коннектора остаются под AGPL. Вклад — по DCO (`git commit -s`).
