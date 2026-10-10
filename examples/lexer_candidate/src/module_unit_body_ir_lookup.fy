module module_unit_body_ir_lookup;
import module_unit_body_ir_model as ir;
pub fn usize node(ref ir::Body b,usize id){var usize low=0;var usize high=ir::node_count(b);while(low<high){let mid=low+(high-low)/2;if(ir::node_at(b,mid).value.source_index<id){low=mid+1;}else{high=mid;}}return if(low<ir::node_count(b) && ir::node_at(b,low).value.source_index==id){low+1}else{0};}
pub fn usize statement(ref ir::Body b,usize id){var usize low=0;var usize high=ir::statement_count(b);while(low<high){let mid=low+(high-low)/2;if(ir::statement_at(b,mid).source_index<id){low=mid+1;}else{high=mid;}}return if(low<ir::statement_count(b) && ir::statement_at(b,low).source_index==id){low+1}else{0};}
pub fn usize local(ref ir::Body b,usize source_index){var usize i=1;while(i<=ir::local_count(b)){if(ir::local_at(b,i).source_index==source_index){return i;}i+=1;}return 0;}
