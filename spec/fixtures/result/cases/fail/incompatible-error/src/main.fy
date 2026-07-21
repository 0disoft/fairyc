module main;

enum SourceError {
    Failed;
}

enum TargetError {
    Failed;
}

fn Result<i32, TargetError> invalid(Result<i32, SourceError> input, Result<i32, TargetError> fallback) {
    let i32 value = input?;
    return fallback;
}
