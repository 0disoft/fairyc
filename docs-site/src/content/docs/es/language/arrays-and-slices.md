---
title: Arreglos, slices y vistas
description: Aprende arreglos fijos T[N], vistas slice ref T[] / ref mut T[], segmentación por rangos, campos de arreglo en línea en struct y diferencias entre str, u8[] y CStr.
---

El código de sistemas trabaja constantemente con bloques contiguos de memoria. Fairy C separa los **arreglos de tamaño fijo** (cuya longitud se conoce en tiempo de compilación) de las **vistas slice** (que toman prestado un rango de memoria sin copiarlo).

## 1. Arreglos fijos (`T[N]`)

Escribe los tipos de arreglo como `ElementType[Length]` (Stage 0 admite longitudes de `1` a `4096`):

```c
let i32[3] scores = [10, 20, 30];
var u8[16] buffer = [0; 16];       // repite 0 dieciséis veces

let usize len = scores.len;        // 3 (usize)
let i32 first = scores[0];
buffer[1] = 65;                    // mutable porque buffer es var
buffer[1] += 1;                    // 66
```

- **Los índices tienen tipo `usize`**: Los literales enteros como `0` toman automáticamente `usize` en posición de índice.
- **Las comprobaciones de límites siempre se ejecutan**: Leer o escribir en un índice `>= len` detiene la ejecución de inmediato con `bounds` en lugar de leer memoria basura.

Las estructuras que no son `@repr(C)` también pueden incrustar arreglos acotados en línea como campos:

```c
struct Packet {
    u8[8] bytes;
    usize length;
}
```

## 2. Vistas slice (`ref T[]`, `ref mut T[]`)

Pasar un arreglo fijo completo por valor copia sus elementos. Cuando solo quieres inspeccionar o mutar elementos en su sitio, toma prestada una vista slice en su lugar:

```c
fn u8 first_byte(ref u8[] view) {
    if (view.is_empty) { return 0; }
    return view[0];
}

fn void clear_head(ref mut u8[] view) {
    if (view.len > 0) {
        view[0] = 0;
    }
}

fn u8 demo() {
    var u8[4] data = [70, 97, 105, 114];
    clear_head(ref mut data);
    return first_byte(ref data);
}
```

- **`ref T[]`**: Vista de solo lectura que admite `.len`, `.is_empty` y lecturas comprobadas `view[index]`.
- **`ref mut T[]`**: Vista mutable exclusiva que permite escribir elementos cuando el almacenamiento subyacente es `var`.

### Segmentación por rangos (`arr[start..end]`)

Segmenta una vista `ref u8[]` (o toma prestado un rango directamente desde un arreglo con nombre, como `ref points[1..3]`) usando rangos semiabiertos `start..end`:

```c
let ref u8[] full = b"Fairy C";
let ref u8[] head = full[0..5];   // b"Fairy"
let ref u8[] tail = full[6..];    // b"C"
let ref u8[] all = full[..];      // vista completa
```

Los rangos inválidos (`start > end` o `end > len`) se detienen inmediatamente con `bounds`.

## 3. Comparación de literales de cadena y bytes

| Literal | Tipo | Propósito |
| --- | --- | --- |
| `"Fairy"` | `ref str` | Vista de texto UTF-8 validado; expone `.len` en bytes y `.is_empty` |
| `b"Fairy\xff"` | `ref u8[]` | Slice de bytes sin procesar; admite indexación `b[0]` y segmentación `b[1..3]` |
| `c"Fairy"` | `ref CStr` | Vista terminada en NUL (`\0`) garantizada para APIs de C |

`ref str` impide indexar bytes directamente con `s[0]` para que no se dividan por accidente los límites válidos de UTF-8. Cuando necesites los bytes sin procesar, convierte explícitamente con `std::text::as_bytes(s)`.

---

- Anterior: [Estructuras, enums y switch](../structs-enums-and-errors/)
- Siguiente: [Propiedad, préstamos y recursos](../memory-and-resources/)
