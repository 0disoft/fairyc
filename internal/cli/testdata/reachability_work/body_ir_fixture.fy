
// Appended only to the temporary body-model copy. Production APIs and their
// ordinary verifier entry points are untouched.
pub fn Program reachability_fixture(clean_ir::Program cleanup_ir,v::Vec<Body> bodies,v::Vec<interfaces::Callable> callables){
  return Program{.states=state::empty(),.sdk_sources=sdk_sources::empty(),.cleanup_ir,.bodies,.layouts=v::empty<Layout>(),.fields=v::empty<interfaces::Field>(),.callables,.parameters=v::empty<usize>(),.names=bytes::empty(),.work=0};
}
