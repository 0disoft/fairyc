// Test-only construction: exercise check/encode on an otherwise valid 2049-row
// component, which the checked public assembler must refuse to return.
pub fn Metadata project_layout_limit_fixture(v::Vec<ir::Layout> layouts,v::Vec<interfaces::Field> fields,bytes::Buffer names){return Metadata{.layouts,.fields,.callables=v::empty<interfaces::Callable>(),.parameters=v::empty<usize>(),.names};}
