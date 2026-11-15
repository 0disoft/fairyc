---
title: Módulos, paquetes y std
description: Aprende la sintaxis de módulos e importaciones de Fairy C, visibilidad pub, espacios de trabajo fairy.toml y los módulos actuales de std.
---

En cuanto dividas el código entre varios archivos o incorpores bibliotecas, usarás el sistema de módulos y paquetes de Fairy C. No hay archivos de cabecera (`#include`) ni macros del preprocesador de C.

## 1. Las rutas de archivo coinciden con los nombres de `module`

Cada archivo `.fy` comienza con una declaración `module` que coincide con su ruta bajo la raíz de código fuente (`src/app/math.fy` -> `module app::math;`):

```c
module app::math;

pub fn i32 add(i32 a, i32 b) {
    return a + b;
}
```

Solo los elementos marcados con `pub` son visibles para otros módulos.

## 2. Importaciones explícitas con alias (`import ... as ...`)

Toda importación requiere un alias (`as name`), manteniendo todos los nombres externos claramente calificados:

```c
module main;

import app::math as math;
import std::io as io;

@effects(console)
fn i32 main() {
    let i32 v = math::add(70, 7);
    discard io::write_i32(v);
    return 0;
}
```

No se permiten importaciones comodín (`*`) ni dependencias circulares entre módulos.

## 3. `fairy.toml`, espacios de trabajo y `fairy.lock`

Define paquetes y dependencias directas en `fairy.toml`:

```toml
schema = "fairyc.manifest/v1"

[package]
uid = "example.my-app"
name = "my-app"
version = "0.1.0"
language = "fairyc-language/20"
source = "src"

[workspace]
members = ["libs/answer"]

[dependencies.answer]
path = "libs/answer"
version = "0.1.0"
features = []
```

- Una sección raíz `[workspace]` te permite gestionar varios paquetes juntos (`examples/workspace_quickstart` muestra esta organización).
- Al ejecutar `fairyc check` o `fairyc test`, se registra el grafo de paquetes resuelto en `fairy.lock`. Pasa `--locked` cuando quieras que las compilaciones fallen si cambian las dependencias.
- Los destinos de Windows que usan configuraciones adicionales del enlazador se emparejan con `fairy.windows.json`.

## 4. Módulos de `std` disponibles

Cuando un paquete enlaza la raíz del SDK `std` (como se ve en los ejemplos del repositorio), están disponibles los siguientes módulos:

| Módulo | Propósito | Funciones o tipos de ejemplo |
| --- | --- | --- |
| `std::io` | E/S de consola (`@effects(console)`) | `write`, `write_i32`, `write_i64`, `write_bytes`, `write_stderr` |
| `std::vec` | Contenedor extensible `Vec<T>` | `empty<T>`, `push<T>`, `pop<T>`, `at<T>`, `truncate<T>`, `clear<T>` |
| `std::text` | Cadenas UTF-8 y utilidades de `CString` | `validate_utf8`, `copy`, `append`, `as_bytes`, `cstring_from_text` |
| `std::bytes` | Utilidades para slices y búferes de bytes | Ayudantes de comparación y manipulación de bytes |
| `std::fs` | Operaciones del sistema de archivos (`@effects(filesystem)`) | `open_read`, `create_truncate`, `read`, `write_all`, `seek_start`, `close` |
| `std::num` | Análisis verificado y conversiones de reducción | `parse_i64_decimal`, `usize_from_i64`, `u8_from_i64` |
| `std::process` | Argumentos del proceso y manejo de salida | `process::arg(...)` |
| `std::alloc` | Interfaz explícita de asignador | Primitivas explícitas de asignación de búferes y recursos |

---

- Anterior: [Propiedad, préstamos y recursos](../memory-and-resources/)
- Siguiente: [Interoperabilidad con C y Win32](../../examples/c-interop-and-win32/)
