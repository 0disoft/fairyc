---
title: Primer programa
description: Crea tu primer paquete de Fairy C, recorre el código generado y ejecuta comprobaciones, pruebas y compilaciones nativas.
---

Vamos a crear un paquete inicial de Fairy C, leer línea por línea los archivos generados y ejecutarlo.

## 1. Crear un paquete nuevo

Elige un directorio vacío fuera del repositorio del compilador y ejecuta `fairyc new` con un `--uid` único:

```powershell
fairyc new hello-fairy --uid example.hello-fairy
cd hello-fairy
```

Esto genera cinco archivos: `fairy.toml`, `src/main.fy`, `tests/main.fy`, `.gitignore` y `README.md`.

Mira `fairy.toml`:

```toml
schema = "fairyc.manifest/v1"

[package]
uid = "example.hello-fairy"
name = "hello-fairy"
version = "0.1.0"
language = "fairyc-language/20"
source = "src"
```

Y mira `src/main.fy`:

```c
module main;

pub fn i32 answer() {
    return 42;
}

fn i32 main() {
    return answer() - 42;
}
```

Un breve repaso:

- `module main;` declara el nombre de módulo de este archivo. El archivo de entrada `src/main.fy` debe ser `module main;`.
- `pub fn i32 answer()` coloca el tipo de retorno (`i32`) antes del nombre de la función, y `pub` permite que otros módulos (como `tests/main.fy`) la llamen.
- `fn i32 main()` es el punto de entrada. El `i32` que devuelva se convierte en el código de salida del proceso (`42 - 42` devuelve `0`).

## 2. Comprobar sintaxis, tipos y préstamos

```powershell
fairyc check
```

Salida esperada:

```text
check ok: 1 instance(s), 1 source file(s), 2 MIR function(s), lock updated
```

El primer `check` crea `fairy.lock`. Después puedes pasar `--locked` para que las siguientes ejecuciones verifiquen que el manifiesto no haya cambiado inesperadamente:

```powershell
fairyc check --locked
```

## 3. Ejecutar la prueba inicial

`tests/main.fy` ya incluye una prueba de ejemplo. Para añadir más pruebas después, marca funciones `fn void` con `@test`:

```c
module tests::main;

import main as app;

@test
fn void answer_is_42() {
    assert(app::answer() == 42);
}
```

Ejecútala con:

```powershell
fairyc test --locked
```

Salida esperada:

```text
test tests::main::answer_is_42 ... ok
test result: ok. 1 passed; 0 failed; lock verified
```

Pasa `--native` (`fairyc test --native --locked`) si quieres compilar las pruebas en binarios nativos en lugar de ejecutarlas en el intérprete.

## 4. Ejecutar y compilar de forma nativa

Compila y ejecuta un binario nativo temporal:

```powershell
fairyc run --locked
```

Como `main()` devuelve `0`, el proceso termina limpiamente con estado de salida `0`. Para conservar el ejecutable compilado en `build/hello-fairy.exe` (Windows) o `build/hello-fairy` (Linux):

```powershell
fairyc build --release --locked
```

## Tropiezos frecuentes al empezar

- **Olvidar `--uid` en `fairyc new`**: `--uid` es obligatorio y necesita al menos dos segmentos en minúsculas kebab-case separados por un punto (como `example.hello-fairy`).
- **Letras mayúsculas en `name` o `uid`**: La validación del manifiesto solo permite minúsculas en kebab-case.
- **Ejecutar `--locked` justo después de editar `fairy.toml`**: Ejecuta `fairyc check` una vez sin `--locked` para actualizar `fairy.lock`.
- **Código de salida `6` en `build` o `run`**: No se encontró el compilador de C del sistema (MSVC + Windows SDK en Windows, o `gcc`/`clang` en Linux) o falló el enlazado.
