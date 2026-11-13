---
title: Variables, tipos y operadores
description: Aprende la diferencia entre let y var, const en tiempo de compilación, tipos escalares y de cadena, literales, aritmética verificada y conversiones explícitas con as en Fairy C.
---

Si alguna vez has echado un vistazo a C, la sintaxis de Fairy C te resultará familiar enseguida. Las sentencias terminan con `;` y los bloques usan `{ }`. La diferencia es que el compilador detecta errores habituales —como variables sin inicializar, desbordamientos silenciosos de enteros y conversiones implícitas accidentales— incluso antes de que tu programa se ejecute.

## 1. `let`, `var` y `const`

Usa `let` para enlaces que no vas a reasignar y `var` para variables mutables.

```c
let i32 limit = 100;      // no se puede reasignar después
var i32 count = 0;        // se puede actualizar con count = 1;
let auto_inferred = 77;   // se puede omitir el tipo si queda claro por el valor
```

- **La inicialización es obligatoria**: Escribir `var i32 count;` sin un valor no está permitido, lo que elimina los errores de lectura sin inicializar.
- **`let` solo bloquea la reasignación**: Aunque un `resource` esté enlazado con `let`, aún puedes transferir (mover) su propiedad a otra función.

Para constantes de módulo en tiempo de compilación, usa `const` (actualmente constantes escalares privadas de tipo `bool`, `u8`, `i32` o `u32`):

```c
const i32 MAX_RETRY = 3;
static_assert(MAX_RETRY > 0);
```

`static_assert(...)` comprueba una condición pura en tiempo de compilación mientras se compila el programa.

## 2. Tipos principales y literales

| Categoría | Nombres de tipo | Notas |
| --- | --- | --- |
| Enteros con signo | `i8`, `i16`, `i32`, `i64`, `isize` | `i32` para enteros generales, `i64` para 64 bits |
| Enteros sin signo | `u8`, `u16`, `u32`, `u64`, `usize` | `u8` para bytes, `usize` para índices y longitudes de arreglos |
| Booleano | `bool` | `true`, `false` (nunca se mezclan con enteros) |
| Punto flotante | `f32`, `f64` | Soporte acotado de transporte, como campos `@repr(C)` |
| Escalares de ABI C | `c::int`, `c::uint`, `c::size_t`, ... | Se usan al enlazar funciones `extern C` |
| Vistas de texto y bytes | `ref str`, `ref CStr`, `ref u8[]` | Cadenas UTF-8, cadenas C terminadas en NUL, slices de bytes |

Los literales admiten separadores numéricos (`_`), hexadecimal (`0x`), binario (`0b`) y tres formatos de cadena o bytes:

```c
let i32 million = 1_000_000;
let u32 mask = 0xff_00_aa;
let u8 bits = 0b1010_0101;

let ref str text = "Hello Fairy"; // vista de cadena UTF-8
let ref u8[] raw = b"PK\x03\x04";  // literal de slice de bytes
let ref CStr title = c"Window";   // vista de cadena C terminada en NUL
```

Ten en cuenta que Stage 0 no tiene literales de carácter (`'a'`) ni sufijos numéricos (`10u32`). Los tipos provienen de anotaciones explícitas o del contexto esperado.

## 3. Operadores y aritmética verificada

Están integrados los operadores aritméticos estándar (`+`, `-`, `*`, `/`, `%`), de bits (`~`, `&`, `|`, `^`, `<<`, `>>`) y las asignaciones compuestas (`+=`, `-=`, `*=`, `/=`, `%=`, `<<=`, `>>=`, `&=`, `^=`, `|=`). (No existen `++`, `--` ni `? :`.)

Una regla clave de seguridad:

```c
var u8 x = 250;
x += 10; // A diferencia de C, que daría la vuelta silenciosamente a 4, la ejecución se detiene de inmediato!
```

- **El desbordamiento y la división por cero detienen el programa (trap)**: El desbordamiento aritmético, el subdesbordamiento sin signo, la división por cero y los desplazamientos fuera de rango se detienen de forma segura en lugar de corromper valores.
- **Sin comparaciones encadenadas**: Escribe `(0 < x) && (x < 10)` en lugar de `0 < x < 10` (`E0009`).
- **Las condiciones requieren `bool`**: No puedes escribir `if (count)`; escribe siempre una comparación explícita como `if (count != 0)`.

## 4. Conversiones explícitas de tipo (`as`)

Los tipos de enteros distintos nunca se mezclan implícitamente. Usa `as` para conversiones explícitas de ampliación que preservan el valor:

```c
let u8 byte_val = 40;
let i32 a = byte_val as i32;      // u8 -> i32
let u32 b = byte_val as u32;      // u8 -> u32
let usize c = b as usize;         // u32 -> usize
```

Actualmente `as` seguro cubre pares de ampliación que nunca pierden datos (`u8 -> i32/u32/usize`, `u32 -> usize`, además de `i32 <-> c::int` en destinos con `c::int` de 32 bits). Las conversiones de reducción o cambio de signo usan ayudantes explícitos que pueden fallar, como los de `std::num`.

---

Siguiente: [Funciones y flujo de control](../control-flow-and-functions/)
