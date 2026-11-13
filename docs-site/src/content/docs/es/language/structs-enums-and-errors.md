---
title: Estructuras, enums y switch
description: Aprende estructuras de Fairy C, abreviatura de campos, sintaxis Type::method, enums, coincidencia exhaustiva con switch y propagación de errores con Option, Result y ?.
---

Cuando los programas crecen más allá de unos cuantos enteros, querrás agrupar campos y modelar múltiples resultados de forma limpia. Fairy C combina `struct`, `enum`, `switch` exhaustivo y `Option` / `Result` para esto.

## 1. Estructuras y abreviatura de campos

Define un registro simple con `struct`. Añade `pub` a los campos que quieras que sean accesibles fuera del módulo que los define:

```c
pub struct Point {
    pub i32 x;
    pub i32 y;
}

fn i32 demo() {
    let i32 x = 10;
    let i32 y = 32;

    // .y es abreviatura de .y = y
    let Point p = Point { .x = x, .y };
    return p.x + p.y;
}
```

Siempre que una variable local tenga el mismo nombre que el campo, `.field` funciona como abreviatura. Todos los campos deben proporcionarse en el orden de declaración.

### Métodos con `Type::method`

Asocia funciones con un `struct` o `resource` prefijando el nombre con `Type::`. Cuando el primer parámetro se llama `this` (`ref T this`, `ref mut T this` o `T this`), puedes llamarlo con sintaxis de punto:

```c
struct Counter {
    i32 value;
}

fn i32 Counter::read(ref Counter this) {
    return this.value;
}

fn void Counter::add(ref mut Counter this, i32 delta) {
    this.value += delta;
}

fn i32 main() {
    var Counter c = Counter { .value = 75 };
    c.add(2);
    return c.read();
}
```

## 2. Enums y enums compatibles con C

Usa `enum` para representar variantes mutuamente excluyentes, que opcionalmente llevan campos inertes con nombre como carga útil:

```c
enum ParseError {
    Empty;
    InvalidByte { usize offset; u8 byte; }
}
```

Cuando el tipo esperado se conoce por el contexto, construye variantes empezando con un punto: `.Empty` o `.InvalidByte { .offset = 2, .byte = 255 }`.

Para enums enteros simples con diseño de C dentro de registros `@repr(C)`, anota el tipo entero de C subyacente:

```c
@repr(c::uint)
enum Status {
    Ready;
    Busy;
}
```

## 3. `switch` exhaustivo

`switch` te obliga a cubrir cada variante explícitamente. No existe paso implícito (fallthrough) al estilo de C entre casos.

```c
fn i32 code_of(ParseError err) {
    return switch (err) {
        case .Empty: { 1 }
        case .InvalidByte { .offset = pos, .byte = _ }: {
            (pos as i32) + 10
        }
    };
}
```

- Cada `case` requiere un bloque `{ ... }`.
- Los campos sin usar se pueden ignorar con `_`, y dejar una expresión final sin `;` permite que el propio `switch` devuelva un valor.

## 4. `Option<T>`, `Result<T, E>` y `?`

- **`Option<T>`**: `.some(value)` o `.none`
- **`Result<T, E>`**: `.ok(value)` o `.err(error)`

Para propagar errores hacia arriba sin escribir `switch` repetitivos, añade el sufijo `?`:

```c
fn Result<i32, ParseError> add_one(Result<i32, ParseError> input) {
    let i32 n = input?;    // devuelve .err(...) anticipadamente si falla
    return .ok(n + 1);
}
```

### `@must_use` y `discard`

Ignorar un `Result` sin manejarlo es un error de compilación. Si quieres descartar intencionadamente un `Result` (o cualquier función marcada con `@must_use`), reconócelo explícitamente con `discard`:

```c
discard add_one(.err(.Empty));

@must_use
fn usize bytes_written() {
    return 16;
}
```

---

- Anterior: [Funciones y flujo de control](../control-flow-and-functions/)
- Siguiente: [Arreglos, slices y vistas](../arrays-and-slices/)
