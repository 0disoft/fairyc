package cli

const candidateBodyTreeWriter = `
struct ExportBudget {usize nodes; bool emit;}
@effects(console) fn bool record(ref mut ExportBudget budget,usize depth,usize kind,usize start,usize end,usize children,usize name_start,usize name_end){
 if(depth>=64 || budget.nodes>=256){return false;}budget.nodes=budget.nodes+1;
 if(budget.emit){number(kind);number(start);number(end);number(children);number(name_start);number(name_end);}return true;
}
@effects(console) fn bool type_tree(ref syntax::Parsed p,usize id,ref mut ExportBudget budget,usize depth){
 if(id==0){return false;}let t=syntax::type_at(p,id);
 var usize kind=0;var usize count=0;
 if(t.kind==1){kind=301;}
 else if(t.kind==2){kind=302;count=1;}else if(t.kind==3){kind=303;count=1;}
 else if(t.kind==4){kind=304;count=1;}else if(t.kind==5){kind=305;count=1;}
 else if(t.kind==6){kind=306;count=syntax::type_argument_count(p,id);count+=1;}
 else if(t.kind==8){kind=308;count=2;}else if(t.kind==9){kind=309;}else if(t.kind==10){kind=310;}
 else if(t.kind==11){kind=311;count=1;}else if(t.kind==12){kind=312;count=1;}
 else if(t.kind==15){kind=315;count=1;var usize link=t.right;while(link!=0){let item=syntax::type_at(p,link);if(item.kind!=16 || count>=256){return false;}count+=1;link=item.right;}}
 else{return false;}
 if(!record(budget,depth,kind,t.start,t.end,count,0,0)){return false;}
 if(budget.emit){number(0);}
 if(count==0){return true;}
 if(!type_tree(p,t.left,budget,depth+1)){return false;}
 if(kind==308){return type_tree(p,t.right,budget,depth+1);}
 if(kind==306){var usize i=0;while(i+1<count){if(!type_tree(p,syntax::type_argument(p,id,i),budget,depth+1)){return false;}i+=1;}}
 if(kind==315){var usize i=0;while(i+1<count){var usize link=t.right;var usize skip=count-2-i;while(skip>0){link=syntax::type_at(p,link).right;skip-=1;}if(!type_tree(p,syntax::type_at(p,link).left,budget,depth+1)){return false;}i+=1;}}
 return true;
}
@effects(console) fn bool pattern_tree(ref syntax::Parsed p,usize id,ref mut ExportBudget budget,usize depth){
 if(id==0){return false;}let pattern=syntax::expression_at(p,id);
 var usize kind=225;var usize count=1;
 if(pattern.kind==32){kind=227;count+=syntax::pattern_field_count(p,id);}
 else if(pattern.kind==16){if(pattern.left!=0){kind=226;count=2;}}else{return false;}
 if(!record(budget,depth,kind,pattern.start,pattern.end,count,0,0)){return false;}if(budget.emit){number(0);}
 let variant=syntax::comparison_token(p,pattern.token);
 if(!record(budget,depth+1,201,pattern.start,variant.end,0,0,0)){return false;}if(budget.emit){number(0);}
 if(kind==226){let binding=syntax::expression_at(p,pattern.left);if(!record(budget,depth+1,229,binding.start,binding.end,0,0,0)){return false;}if(budget.emit){number(0);}}
 if(kind==227){var usize i=0;while(i+1<count){
  let field=syntax::expression_at(p,syntax::pattern_field(p,id,i));let name=syntax::comparison_token(p,field.token);
  if(!record(budget,depth+1,228,field.start,field.end,1,name.start,name.end)){return false;}if(budget.emit){number(0);}
  let binding=syntax::expression_at(p,field.left);
  if(!record(budget,depth+2,229,binding.start,binding.end,0,0,0)){return false;}if(budget.emit){number(0);}
  i+=1;
 }}return true;
}
@effects(console) fn bool expression_tree(ref syntax::Parsed p,usize id,ref mut ExportBudget budget,usize depth){
 if(id==0){return false;}let e=syntax::expression_at(p,id);
 var usize kind=0;var usize count=0;
 if(e.kind==1){kind=200;}else if(e.kind==2 || e.kind==28){kind=201;}
 else if(e.kind==3){kind=202;count=1;}else if(e.kind==4){kind=203;count=1;}
 else if(e.kind==5){kind=204;count=2;}else if(e.kind==6){kind=205;count=syntax::call_argument_count(p,id);count+=1;}
 else if(e.kind==12){kind=206;count=1;}else if(e.kind==13){kind=207;count=if(syntax::expression_at(p,e.right).right==0){2}else{3};}
 else if(e.kind==23){kind=208;count=2;}else if(e.kind==20){kind=209;count=syntax::specialization_argument_count(p,id);if(count==0){return false;}count+=1;}
 else if(e.kind==8){kind=210;count=1;}else if(e.kind==9){kind=211;count=2;}else if(e.kind==26){kind=212;count=3;}
 else if(e.kind==24){kind=215;count=syntax::array_element_count(p,id);}
 else if(e.kind==29){kind=218;count=2;}
 else if(e.kind==21){kind=219;count=1;}else if(e.kind==22){kind=220;count=1;}
 else if(e.kind==30){kind=221;count=1;}else if(e.kind==31){kind=222;count=1;}
 else if(e.kind==15){kind=223;count=syntax::switch_case_count(p,id);count+=1;}
 else if(e.kind==10){kind=216;count=syntax::record_field_count(p,id);count+=1;let target=syntax::expression_at(p,e.left);if(target.kind==20){count+=syntax::specialization_argument_count(p,e.left);}}
 else{return false;}
 var usize name_start=0;var usize name_end=0;
 if(kind==210){let ref u8[] name=syntax::expression_text(p,id);name_start=e.end-name.len;name_end=e.end;}
 if(!record(budget,depth,kind,e.start,e.end,count,name_start,name_end)){return false;}
 if(budget.emit){if(kind==203 || kind==204){let ref u8[] op=syntax::expression_text(p,id);number(op.len);var usize i=0;while(i<op.len){number(op[i] as usize);i+=1;}}else{number(0);}}
 if(kind==200 || kind==201){return true;}
 if(kind==206 || kind==222){return statement_tree(p,e.left,budget,depth+1);}
 if(kind==215){var usize i=0;while(i<count){if(!expression_tree(p,syntax::array_element(p,id,i),budget,depth+1)){return false;}i+=1;}return true;}
 if(kind==216){
  let target=syntax::expression_at(p,e.left);
  if(target.kind==20){
   if(!expression_tree(p,target.left,budget,depth+1)){return false;}
   var usize i=0;while(i<syntax::specialization_argument_count(p,e.left)){if(!type_tree(p,syntax::specialization_argument(p,e.left,i),budget,depth+1)){return false;}i+=1;}
  }else{if(!expression_tree(p,e.left,budget,depth+1)){return false;}}
  var usize i=0;while(i<syntax::record_field_count(p,id)){
   let field=syntax::expression_at(p,syntax::record_field(p,id,i));let name=syntax::comparison_token(p,field.token);
   if(!record(budget,depth+1,217,field.start,field.end,1,name.start,name.end)){return false;}if(budget.emit){number(0);}
   if(!expression_tree(p,field.left,budget,depth+2)){return false;}i+=1;
  }return true;
 }
 if(!expression_tree(p,e.left,budget,depth+1)){return false;}
 if(kind==223){var usize i=0;while(i+1<count){
  let arm=syntax::expression_at(p,syntax::switch_case(p,id,i));
  if(!record(budget,depth+1,224,arm.start,arm.end,2,0,0)){return false;}if(budget.emit){number(0);}
  if(!pattern_tree(p,arm.left,budget,depth+2) || !statement_tree(p,arm.right,budget,depth+2)){return false;}i+=1;
 }return true;}
 if(kind==204){return expression_tree(p,e.right,budget,depth+1);}
 if(kind==211){return expression_tree(p,e.right,budget,depth+1);}
 if(kind==218){return expression_tree(p,e.right,budget,depth+1);}
 if(kind==212){
  let bounds=syntax::expression_at(p,e.right);
  if(bounds.left!=0){if(!expression_tree(p,bounds.left,budget,depth+1)){return false;}}
  else{if(!record(budget,depth+1,213,bounds.start,bounds.start+2,0,0,0)){return false;}if(budget.emit){number(0);}}
  if(bounds.right!=0){return expression_tree(p,bounds.right,budget,depth+1);}
  if(!record(budget,depth+1,214,e.end-1,e.end-1,0,0,0)){return false;}if(budget.emit){number(0);}
 }
 if(kind==208){return type_tree(p,e.right,budget,depth+1);}
 if(kind==209){var usize i=0;while(i+1<count){if(!type_tree(p,syntax::specialization_argument(p,id,i),budget,depth+1)){return false;}i+=1;}}
 if(kind==205){var usize i=0;while(i+1<count){
  let entry=syntax::expression_at(p,syntax::call_argument_entry(p,id,i));let name=syntax::comparison_token(p,entry.token);
  var usize child_depth=depth+1;
  if(name.start>entry.start){
   if(name.kind!=3 || !record(budget,child_depth,230,entry.start,entry.end,1,name.start,name.end)){return false;}if(budget.emit){number(0);}child_depth+=1;
  }
  if(!expression_tree(p,syntax::call_argument(p,id,i),budget,child_depth)){return false;}i+=1;
 }}
 if(kind==207){
  let branches=syntax::expression_at(p,e.right);
  if(!statement_tree(p,branches.left,budget,depth+1)){return false;}
  if(branches.right!=0){let other=syntax::expression_at(p,branches.right);if(other.kind==12){return statement_tree(p,other.left,budget,depth+1);}return expression_tree(p,branches.right,budget,depth+1);}
 }
 return true;
}
@effects(console) fn bool statement_tree(ref syntax::Parsed p,usize id,ref mut ExportBudget budget,usize depth){
 if(id==0){return false;}let s=syntax::statement_at(p,id);
 var usize kind=0;var usize count=0;var usize name_start=0;var usize name_end=0;
 if(s.kind==1){kind=100;count=syntax::block_statement_count(p,id);if(s.expression!=0){count+=1;}}
 else if(s.kind==2){let entry=syntax::entry_at(p,s.entry-1);kind=if(entry.kind==2){101}else{102};count=1;if(entry.type_root!=0){kind+=8;count+=1;}name_start=entry.name_start;name_end=entry.name_end;}
 else if(s.kind==3){kind=103;count=if(s.expression==0){0}else{1};}
 else if(s.kind==4){kind=104;count=2;}else if(s.kind==6){kind=105;count=1;}
 else if(s.kind==8){kind=106;count=2;}else if(s.kind==9){kind=107;}else if(s.kind==10){kind=108;}
 else if(s.kind==14){kind=111;count=4;}
 else if(s.kind==11){kind=115;count=1;}else if(s.kind==13){kind=116;count=1;}
 else if(s.kind==12){kind=117;count=1;}else if(s.kind==16){kind=118;count=1;}
 else if(s.kind==17){kind=119;count=2;let name=syntax::expression_at(p,s.target);name_start=name.start;name_end=name.end;}
 else{return false;}
 if(kind>=115 && kind<=118 && s.expression==0){return false;}
 if(!record(budget,depth,kind,s.start,s.end,count,name_start,name_end)){return false;}
 if(budget.emit){if(kind==104){let ref u8[] op=syntax::assignment_operator(p,id);number(op.len);var usize i=0;while(i<op.len){number(op[i] as usize);i+=1;}}else{number(0);}}
 if(kind==100){var usize i=0;while(i<syntax::block_statement_count(p,id)){if(!statement_tree(p,syntax::block_statement(p,id,i),budget,depth+1)){return false;}i+=1;}return s.expression==0 || expression_tree(p,s.expression,budget,depth+1);}
 if(kind==111){
  let header=syntax::statement_at(p,s.second);
  if(header.first!=0){if(!statement_tree(p,header.first,budget,depth+1)){return false;}}
  else{if(!record(budget,depth+1,112,0,0,0,0,0)){return false;}if(budget.emit){number(0);}}
  if(s.expression!=0){if(!expression_tree(p,s.expression,budget,depth+1)){return false;}}
  else{if(!record(budget,depth+1,113,0,0,0,0,0)){return false;}if(budget.emit){number(0);}}
  if(header.second!=0){if(!statement_tree(p,header.second,budget,depth+1)){return false;}}
  else{if(!record(budget,depth+1,114,0,0,0,0,0)){return false;}if(budget.emit){number(0);}}
  return statement_tree(p,s.first,budget,depth+1);
 }
 if(kind==104 && !expression_tree(p,s.target,budget,depth+1)){return false;}
 if((kind==109 || kind==110) && !type_tree(p,syntax::entry_at(p,s.entry-1).type_root,budget,depth+1)){return false;}
 if(s.expression!=0 && !expression_tree(p,s.expression,budget,depth+1)){return false;}
 if(kind==106 || kind==119){return statement_tree(p,s.first,budget,depth+1);}return true;
}
@effects(console) fn i32 report(syntax::Parsed parsed){
 if(syntax::problem_count(ref parsed)!=0 || !integrity::valid(ref parsed) || syntax::declaration_count(ref parsed)!=2){return 4;}
 let d=syntax::declaration_at(ref parsed,1);if(d.kind!=3){return 4;}
 var ExportBudget budget=ExportBudget{.nodes=0,.emit=false};
 if(!statement_tree(ref parsed,d.body,ref mut budget,0)){return 4;}
 let ref u8[] raw=syntax::source_view(ref parsed);number(1);number(raw.len);
 var usize i=0;while(i<raw.len){number(raw[i] as usize);i+=1;}
 budget.nodes=0;budget.emit=true;
 if(!statement_tree(ref parsed,d.body,ref mut budget,0)){return 4;}return 0;
}
`

const candidateFunctionTreeWriter = `
@effects(console) fn bool function_tree(ref syntax::Parsed p,ref mut ExportBudget budget){
 let d=syntax::declaration_at(p,1);
 if(d.kind!=3){return false;}
 var usize parameters=0;var usize i=0;
 while(i<d.entry_count){let e=syntax::entry_at(p,d.entry_start+i);if(e.kind==1){parameters+=1;}i+=1;}
 var usize children=parameters+2+d.attribute_count;
 let usize generics=syntax::generic_parameter_count(p,1);
 if(generics!=0){children+=1;}
 if(d.public){children+=1;}if(d.unsafe_end!=0){children+=1;}if(d.owner_end!=0){children+=1;}
 if(!record(budget,0,90,d.start,d.end,children,d.name_start,d.name_end)){return false;}if(budget.emit){number(0);}
 i=0;while(i<d.attribute_count){
  let a=syntax::attribute_at(p,d.attribute_start+i);
  if(!record(budget,1,400,a.start,a.end,a.argument_count,a.name_start,a.name_end)){return false;}if(budget.emit){number(0);}
  var usize j=0;while(j<a.argument_count){
   let arg=syntax::attribute_argument_at(p,a.argument_start+j);
   if(arg.kind<1 || arg.kind>3){return false;}
   var usize kind=400;kind+=arg.kind as usize;
   if(!record(budget,2,kind,arg.start,arg.end,0,0,0)){return false;}if(budget.emit){number(0);}j+=1;
  }i+=1;
 }
 if(d.public){if(!record(budget,1,92,d.public_start,d.public_end,0,0,0)){return false;}if(budget.emit){number(0);}}
 if(d.unsafe_end!=0){if(!record(budget,1,93,d.unsafe_start,d.unsafe_end,0,0,0)){return false;}if(budget.emit){number(0);}}
 if(d.owner_end!=0){if(!record(budget,1,94,d.owner_start,d.owner_end,0,0,0)){return false;}if(budget.emit){number(0);}}
 if(generics!=0){
  if(!record(budget,1,95,d.generic_start,d.generic_end,generics,0,0)){return false;}if(budget.emit){number(0);}
  i=0;while(i<generics){
   let name=syntax::generic_parameter_at(p,1,i);
   if(name.kind!=13 || !record(budget,2,96,name.start,name.end,0,0,0)){return false;}if(budget.emit){number(0);}i+=1;
  }
 }
 if(!type_tree(p,d.result_type,budget,1)){return false;}
 i=0;while(i<d.entry_count){
  let e=syntax::entry_at(p,d.entry_start+i);
  if(e.kind==1){
   if(!record(budget,1,91,e.start,e.end,1,e.name_start,e.name_end)){return false;}if(budget.emit){number(0);}
   if(!type_tree(p,e.type_root,budget,2)){return false;}
  }i+=1;
 }
 return statement_tree(p,d.body,budget,1);
}
`
