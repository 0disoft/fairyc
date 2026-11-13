---
title: Propiedad, préstamos y recursos
description: Comprende Copy vs Move, préstamos ref y ref mut, bloques cleanup de resource, arenas region, @abilities y burn, @effects y punteros unsafe.
---

Este capítulo cubre las ideas que definen a Fairy C: evitar el uso después de liberar y los olvidos de limpieza en tiempo de compilación, permitiéndote al mismo tiempo bajar a `unsafe` y punteros de C cuando hablas con el sistema operativo.

## 1. Copy vs. Move

Los valores pequeños e inertes (`i32`, `bool`, los `struct` simples) se copian automáticamente al asignarse o pasarse a una función.

Sin embargo, los valores declarados como `resource` representan una propiedad única (como un manejador de archivo abierto o un búfer asignado). Pasar un `resource` por valor **mueve** la propiedad a la función llamada; el compilador rechaza usar la variable original después.

## 2. Préstamos con `ref` y `ref mut`

Para permitir que otra función lea o modifique datos sin ceder la propiedad, préstalos:

- **`ref T` (préstamo compartido)**: Pueden coexistir múltiples lectores.
- **`ref mut T` (préstamo exclusivo)**: Solo puede haber un prestatario mutable activo a la vez.

```c
fn i32 read_num(ref i32 n) {
    return *n;
}

fn void add_ten(ref mut i32 n) {
    *n += 10;
}

fn i32 main() {
    var i32 score = 67;
    add_ten(ref mut score);
    return read_num(ref score);   // 77
}
```

Lee o actualiza un préstamo escalar mediante `*n`. Las funciones que devuelven una referencia prestada deben vincularla al origen de un parámetro de entrada: devolver una referencia a una variable local se rechaza para que no puedan ocurrir punteros colgantes.

## 3. Limpieza automática: `resource` y `cleanup`

En lugar de emparejar manualmente cada `CreateFile` o `malloc` en cada `return` anticipado, adjunta un único bloque `cleanup { ... }` a un `resource`:

```c
resource Buffer {
    ptr mut u8 data;

    cleanup {
        // Se ejecuta automáticamente al salir del ámbito, en break, continue o return!
    }
}
```

Los campos de un recurso permanecen privados fuera del módulo que lo define, manteniendo los manejadores sin procesar encapsulados detrás de funciones fábrica y métodos seguros.

## 4. `region`: Ámbitos de arena en bloque

Cuando asignas muchos búferes de vida corta que comparten un mismo tiempo de vida, usa un bloque `region`:

```c
fn Result<i32, MyError> parse_temp(Region arena, ref u8[] input) {
    region scratch in arena {
        // Las asignaciones temporales dentro de scratch se reinician juntas al salir del bloque
    }
    return .ok(0);
}
```

## 5. `@abilities` y `burn` lineal

Por defecto, un `resource` se libera automáticamente cuando su dueño sale de ámbito. Si quieres un comportamiento lineal donde olvidar un valor (un ticket de obligación, token o recibo) sea un error de compilación, declara `@abilities` sin `drop`:

```c
@abilities(store, key)   // ¡omitir drop prohíbe la limpieza implícita!
resource Receipt {
    u32 amount;
    cleanup {}
}
```

Un valor así debe pasarse o devolverse (`E1108` si queda vivo al salir del ámbito), y solo puede destruirse explícitamente dentro de su módulo declarante usando `burn receipt;`. (Las habilidades válidas son `drop`, `store` y `key`, donde `key` requiere `store`.)

## 6. `@effects(...)`: Autoridad externa explícita

Las funciones que tocan E/S externa o recursos del host declaran `@effects(...)`:

```c
@effects(console)
fn void greet() {
    // Tiene permiso para llamar a funciones de consola
}
```

Algunos nombres comunes son `console`, `filesystem`, `environment`, `process`, `network`, `clock` y `random`. Cualquier llamador que invoque una función `@effects(...)` —o que libere un recurso cuyo `cleanup` declare efectos— también debe incluir esos efectos (`E7102`).

## 7. `unsafe` y punteros sin procesar (`ptr` / `ptr mut`)

Al llamar a funciones de Win32 o C (`extern C`), usa `ptr T`, `ptr mut T` y una función `unsafe` o un bloque `unsafe { ... }`:

```c
unsafe extern C fn void c_free(ptr mut void p);
```

Ten en cuenta que `unsafe` solo habilita conversiones de punteros sin procesar, desreferencias y llamadas externas; **no** omite la comprobación de `@effects(...)`.

---

- Anterior: [Arreglos, slices y vistas](../arrays-and-slices/)
- Siguiente: [Módulos, paquetes y std](../modules-and-projects/)
