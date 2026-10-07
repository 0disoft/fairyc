module hir_origin_targets;
import hir_model as h;
// One-based parameter position; zero means no proven input-origin substitution.
pub fn usize parameter(ref h::Program p,usize callee,usize origin){var usize i=0;while(i<h::function_count(p)){let f=h::function_at(p,i);if(f.binding==callee){var usize j=0;while(j<f.parameter_count){if(h::edge_at(p,f.parameter_start+j)==origin){return j+1;}j+=1;}return 0;}i+=1;}return 0;}
