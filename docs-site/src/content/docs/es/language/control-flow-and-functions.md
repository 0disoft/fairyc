---
title: Funciones y flujo de control
description: Aprende declaraciones de funciones, etiquetas de argumentos, genéricos escalares, expresiones if, while, for estilo C, iteración for-in y defer en Fairy C.
---

Ahora que las variables y los operadores están claros, veamos las funciones y el flujo de control. Fairy C mantiene un ritmo parecido al de C, añadiendo comodidades como `if` que devuelve valores, bucles `for-in` seguros y `defer`.

## 1. Funciones y etiquetas de argumentos

Las funciones siguen la forma `fn TipoRetorno nombre(Parametros)`. Usa `void` cuando no se devuelva ningún valor.

```c
fn i32 clamp(i32 value, i32 minimum, i32 maximum) {
    if (value < minimum) { return minimum; }
    if (value > maximum) { return maximum; }
    return value;
}

fn void log_step() {
    // las funciones void pueden omitir un return explícito;
}
```

A diferencia de Rust, **los cuerpos de funciones que no son `void` no devuelven implícitamente su última línea sin `return`**. Escribe siempre `return value;`.

### Etiquetas de argumentos que conservan la posición

Al llamar a una función local del módulo que no sea genérica, puedes aclarar los argumentos usando `.nombre_parametro = valor`:

```c
let i32 result = clamp(77, .minimum = 0, .maximum = 100);
```

Las etiquetas no pueden reordenar argumentos ni proporcionar valores por defecto —cada etiqueta debe coincidir con la posición exacta de ese parámetro—, por lo que sirven estrictamente como guía de legibilidad.

### Indicación `@inline` en funciones

Las funciones auxiliares no genéricas pueden llevar `@inline` una sola vez (sin argumentos) para permitir la expansión en línea local o entre módulos; dentro de su cuerpo solo se admiten llamadas simples del mismo módulo sin `::`.

```c
@inline
pub fn i32 combine(i32 a, i32 b) {
    return a + b;
}
```

## 2. Funciones genéricas escalares (`<T>`)

Las funciones ordinarias pueden aceptar parámetros de tipo escalar explícitos `<T>` y llamarse como `fn_name<i32>(...)`:

```c
fn T pick_first<T>(T first, T second) {
    return first;
}

fn i32 main() {
    return pick_first<i32>(77, 0);
}
```

Actualmente los argumentos de tipo cubren los escalares básicos (`bool`, `u8`, `i32`, `u32`, `usize`) y se especializan en tiempo de compilación. (La inferencia de tipos en las llamadas y los `struct`/`enum` genéricos definidos por el usuario no forman parte de Stage 0.)

### Valores de función y cierres acotados (`fn i32(i32)`)

Stage 0 admite valores de función acotados y cierres que capturan valores por copia con la firma `fn i32(i32)`.

```c
fn i32 apply_once(fn i32(i32) f, i32 x) {
    return f(x);
}

fn i32 make_result() {
    i32 offset = 7;
    fn i32(i32) add_offset = fn [i32 offset] i32(i32 x) {
        return x + offset;
    };
    return apply_once(add_offset, 70);
}
```

- Las listas de captura (`[i32 offset]`) permiten capturar por copia hasta 4 valores `i32` inmutables; estos cierres se pueden pasar, devolver y guardar en campos de un `struct` normal.
- El cuerpo del cierre debe ser puro (sin `var`, efectos ni llamadas externas) y el tipo de cierre no puede usarse dentro de `Option`, `Result`, `ref`, `ptr`, arreglos, `struct` `@repr(C)`, `resource` ni firmas `extern C`.

## 3. `if / else` como expresiones

Las condiciones siempre requieren paréntesis `(...)`. Si cada rama termina con una expresión sin punto y coma, `if` se evalúa directamente a un valor:

```c
let i32 score = 85;
let i32 grade = if (score >= 90) {
    1
} else if (score >= 80) {
    2
} else {
    3
};
```

Ambas ramas deben producir exactamente el mismo tipo. De igual modo, los bloques independientes `{ let i32 a = 70; a + 7 }` también actúan como expresiones cuando terminan con una expresión sin `;`.

## 4. `while` y dos formas de `for`

### `while` y el `for` clásico

```c
var i32 n = 0;
while (n < 5) {
    n += 1;
}

var i32 sum = 0;
for (var i32 i = 0; i < 10; i += 1) {
    if (i == 3) { continue; }
    if (i == 8) { break; }
    sum += i;
}
```

Cualquier cláusula de cabecera en `for (init; cond; post)` puede omitirse (una condición vacía significa `true`). Con `continue`, la cláusula `post` se ejecuta antes de la siguiente comprobación de la condición.

### `for-in` sobre arreglos y vistas de slice

Para evitar errores de índice por uno de más o de menos al recorrer un arreglo o una vista, usa `for-in`:

```c
var i32[4] items = [10, 20, 30, 40];

// Copia cada elemento para leerlo
var i32 total = 0;
for (let i32 item in items) {
    total += item;
}

// Toma prestado cada elemento de forma mutable en su sitio
for (let ref mut i32 item in items) {
    *item += 1;
}
```

También puedes iterar directamente sobre subrangos, como `for (let u8 b in data[1..4]) { ... }`, o recorrer un `Vec<T>` del SDK en solo lectura mediante `for (let ref i32 item in vec) { ... }`.

## 5. `assert`, `defer` y `source_location()`

- **`assert(cond);`**: Detiene la ejecución (trap) inmediatamente si `cond` es falso.
- **`defer call(...);`**: Programa una llamada interna a una función `void` (con argumentos escalares evaluados en el momento del registro) para que se ejecute justo antes de salir del bloque actual, antes de las liberaciones implícitas de recursos o los reinicios de `region`.
- **`source_location()`**: Intrínseco integrado sin argumentos que devuelve `"path:line:col@offset"` como `ref str`, muy útil para diagnósticos y ayudantes de registro.

---

- Anterior: [Variables, tipos y operadores](../basics/)
- Siguiente: [Estructuras, enums y switch](../structs-enums-and-errors/)
