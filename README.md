# Тестовое задание

Требуется написать две программы на Go.

## 1. Вычислитель

Программа принимает HTTP-запросы к маршруту:

```http
POST /calc?num=X
```

и максимально быстро отвечает HTTP-кодом `200 OK`.

Для каждого числа `num` выполняется расчёт:

- для C-библиотеки рассчитывается сумма: `sum = sum + num`;
- для Rust-библиотеки рассчитывается разность: `sub = sub - num`.

Раз в 5 секунд программа выводит текущие значения `sum` и `sub`.

При прерывании программы по `SIGINT` нужно вывести последние значения `sum` и `sub`.

Дополнительно требуется реализовать маршрут:

```http
GET /metrics
```

Маршрут должен возвращать в формате Prometheus:

- RPS HTTP-запросов за каждую секунду последней минуты — 60 значений;
- p95 и p99 времени выполнения вызова функции на C;
- p95 и p99 времени выполнения вызова функции на Rust.

## 2. Генератор

Программа в `N` потоков генерирует случайное число от `-100` до `100` и вызывает по HTTP вычислитель:

```http
POST /calc?num=X
```

## C-библиотека

`calculator.h`:

```c
#ifndef CALCULATOR_H
#define CALCULATOR_H

#include <stdint.h>

int64_t add(int64_t a, int64_t b);

#endif
```

`calculator.c`:

```c
#include "calculator.h"

int64_t add(int64_t a, int64_t b) {
    volatile uint64_t x = (uint64_t)(a + b);

    for (int i = 0; i < 10000; i++) {
        x ^= x << 13;
        x ^= x >> 17;
        x ^= x << 5;
    }

    return a + b;
}
```

## Rust-библиотека

`Cargo.toml`:

```toml
[package]
name = "calculator_rust"
version = "0.1.0"
edition = "2021"

[lib]
name = "calculator_rust"
crate-type = ["staticlib"]
```

`src/lib.rs`:

```rust
#[no_mangle]
pub extern "C" fn sub(a: i64, b: i64) -> i64 {
    let mut x = (a - b) as u64;

    for _ in 0..10_000 {
        x ^= x << 13;
        x ^= x >> 17;
        x ^= x << 5;
    }

    a - b
}
```

# Запуск проекта

Для запуска нужен Docker Desktop с поддержкой Linux containers.

В корневой папке проекта выполните:

```sh
docker compose up --build
```

После запуска вычислитель доступен по адресу `http://localhost:8080`.

Примеры запросов:

```sh
curl -X POST "http://localhost:8080/calc?num=10"
curl "http://localhost:8080/metrics"
```
