module main;

enum Kind {
    Unknown;
    Disk;
}

struct Metadata {
    usize size;
    Kind kind;
    bool read_only;
}

fn i32 score(Metadata value) {
    return if (value.size == 42 && value.read_only) {
        switch (value.kind) {
            case .Unknown : { 1 }
            case .Disk : { 42 }
        }
    } else {
        2
    };
}

fn i32 main() {
    return score(Metadata {
        .size = 42,
        .kind = .Disk,
        .read_only = true,
    });
}
