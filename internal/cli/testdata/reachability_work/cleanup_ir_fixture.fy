
// Appended only to the temporary cleanup-model copy by the Go probe. Raw
// components isolate traversal; this does not authenticate whole-program IR.
pub fn Program reachability_fixture(v::Vec<types::Type> types,v::Vec<Frame> frames,v::Vec<Glue> glues,v::Vec<Field> fields,v::Vec<Operand> operands,v::Vec<Drop> drops){
  return Program{.types,.frames,.glues,.fields,.operands,.drops,.work=0,.project=false,.aggregates=false};
}
