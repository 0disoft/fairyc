module hir_validate_enums;
import hir_model as h;
import enum_types as enums;
import tagged_types as tagged;
import std::vec as v;
import hir_shapes as shapes;

fn bool terminal_arm(ref v::Vec<h::Block> blocks,ref v::Vec<h::Statement> statements,ref v::Vec<h::Expression> expressions,ref v::Vec<usize> edges,usize id,bool allow_return) {
    if(id==0 || id>v::len<h::Block>(blocks)){return false;}let b=*v::at<h::Block>(blocks,id-1);
    if(b.tail!=0 || b.edge_count==0 || b.edge_start>v::len<usize>(edges) || b.edge_count>v::len<usize>(edges)-b.edge_start){return false;}
    let usize sid=*v::at<usize>(edges,b.edge_start+b.edge_count-1);if(sid==0 || sid>v::len<h::Statement>(statements)){return false;}
    let s=*v::at<h::Statement>(statements,sid-1);if(allow_return && h::statement_code(s.kind)==2){return true;}if(h::statement_code(s.kind)!=9 || s.expression==0 || s.expression>v::len<h::Expression>(expressions)){return false;}
    let e=*v::at<h::Expression>(expressions,s.expression-1);return h::expression_code(e.kind)==2 && h::type_code(e.type)==6 && !e.boolean;
}
pub fn bool enum_expression(ref shapes::Table nominal, ref v::Vec<h::Expression> expressions, ref v::Vec<h::Block> blocks, ref v::Vec<h::Statement> statements, ref v::Vec<h::Binding> bindings, ref v::Vec<usize> edges, h::Expression e, usize i) {
    let u32 kind=h::expression_code(e.kind);let u32 type=h::type_code(e.type);let usize nb=v::len<h::Binding>(bindings);
        if (kind == 23) {
            return e.left != 0 && type == 6 && tagged::result(h::type_code(v::at<h::Expression>(expressions,e.left-1).type))
                && e.binding == 0 && e.right == 0 && e.alternative == 0 && e.block == 0 && e.edge_count == 0 && e.edge_start == 0
                && e.payload_start == 0 && e.payload_end == 0 && !e.boolean && h::operator_code(e.operator) == 0;
        }
        if (kind == 22) {
            if (e.left == 0 || e.binding == 0 || e.binding > nb || e.right != 0 || e.alternative != 0 || e.block != 0
                || e.edge_count != 0 || e.edge_start != 0 || e.payload_start != 0 || e.payload_end != 0 || e.boolean || h::operator_code(e.operator) != 0) { return false; }
            let u32 operand = h::type_code(v::at<h::Expression>(expressions,e.left-1).type);
            let h::Binding owner = *v::at<h::Binding>(bindings,e.binding-1); let u32 returned = h::type_code(owner.type);
            return tagged::result(operand) && tagged::result(returned) && h::binding_code(owner.kind) == 1
                && tagged::failure(operand) == tagged::failure(returned) && type == tagged::success(operand);
        }
        if (kind == 18 || kind == 20) {
            if (e.binding == 0 || e.binding > shapes::field_count(nominal) || e.left != 0 || e.right != 0 || e.alternative != 0 || e.boolean || e.payload_start != 0 || e.payload_end != 0 || h::operator_code(e.operator) != 0) { return false; }
            let shapes::Field member = shapes::field_at(nominal,e.binding);
            if (!enums::valid(member.owner) || member.type != 0 || kind == 18 && (type != member.owner || e.block != 0) || kind == 20 && e.block == 0) { return false; }
            if(e.edge_count!=member.payload_count){return false;}
            var u8[1024] payload_seen=[0;1024];var usize n=0;
            while(n<e.edge_count){let usize id=*v::at<usize>(edges,e.edge_start+n);if(id==0 || id>i){return false;}
                let h::Expression f=*v::at<h::Expression>(expressions,id-1);if(h::expression_code(f.kind)!=if(kind==18){11}else{21} || f.binding<=member.payload_start || f.binding>member.payload_start+member.payload_count || payload_seen[f.binding-1]!=0){return false;}
                payload_seen[f.binding-1]=1;
                if(kind==20 && f.edge_count==1){let usize binding=*v::at<usize>(edges,f.edge_start);let h::Binding b=*v::at<h::Binding>(bindings,binding-1);let h::Block arm=*v::at<h::Block>(blocks,e.block-1);if(b.scope!=arm.scope || b.owner!=arm.owner){return false;}}
                n+=1;
            }
            if (kind == 20) { let h::Block arm = *v::at<h::Block>(blocks,e.block-1); if (arm.tail > i) { return false; } if (type != if(arm.tail==0){7}else{h::type_code(v::at<h::Expression>(expressions,arm.tail-1).type)} && !(tagged::builtin(member.owner) && terminal_arm(blocks,statements,expressions,edges,e.block,tagged::result(member.owner)))) { return false; } }
        }
        if(kind==21){
            if(e.binding==0 || e.binding>shapes::field_count(nominal) || e.edge_count>1 || e.left!=0 || e.right!=0 || e.alternative!=0 || e.block!=0 || e.boolean || e.payload_start!=0 || e.payload_end!=0 || h::operator_code(e.operator)!=0){return false;}
            let shapes::Field field=shapes::field_at(nominal,e.binding);if(!enums::valid(field.owner) || field.type==0 || type!=field.type){return false;}
            if(e.edge_count==1){let usize binding=*v::at<usize>(edges,e.edge_start);if(binding==0 || binding>nb){return false;}let h::Binding b=*v::at<h::Binding>(bindings,binding-1);if(h::binding_code(b.kind)!=3 || h::type_code(b.type)!=type){return false;}}
        }
        if (kind == 19) {
            if(e.left==0 || e.right!=0 || e.alternative!=0 || e.block!=0 || e.binding!=0 || e.boolean || e.payload_start!=0 || e.payload_end!=0 || h::operator_code(e.operator)!=0){return false;}
            let u32 subject = h::type_code(v::at<h::Expression>(expressions,e.left-1).type); let usize shape = shapes::find(nominal,subject);
            if(!enums::valid(subject) || shape==0){return false;}
            let shapes::Shape layout = shapes::at(nominal,shape-1);if(e.edge_count!=shapes::member_count(nominal,subject)){return false;}
            var u8[1024] seen=[0;1024];var usize j=0;
            while(j<e.edge_count){let usize id=*v::at<usize>(edges,e.edge_start+j);if(id==0 || id>i){return false;}
                let h::Expression arm=*v::at<h::Expression>(expressions,id-1);if(h::expression_code(arm.kind)!=20 || h::type_code(arm.type)!=type || arm.binding<=layout.field_start || arm.binding>layout.field_start+layout.field_count || seen[arm.binding-1]!=0){return false;}
                seen[arm.binding-1]=1;j+=1;
            }
        }
    return true;
}
