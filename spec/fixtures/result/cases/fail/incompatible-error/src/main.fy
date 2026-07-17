module main;

enum SourceError {
    Failed;
}

enum TargetError {
    Failed;
}

fn invalid(
    input: Result<i32, SourceError>,
    fallback: Result<i32, TargetError>,
) -> Result<i32, TargetError> {
    let value: i32 = input?;
    fallback
}
