module borrow_shapes;
import scalar_views as views;
import reference_types as refs;
import tagged_encoding as tagged;
// Public Stage 0 admits only one safe-borrow leaf through builtin tagged values.
pub fn usize count(u32 type){if(type==9 || views::valid(type) || type==12 || refs::valid(type)){return 1;}if(tagged::option(type)){return count(tagged::inner(type));}if(tagged::result(type)){return count(tagged::success(type))+count(tagged::failure(type));}return 0;}
pub fn u32 leaf(u32 type){if(type==9 || views::valid(type) || type==12 || refs::valid(type)){return type;}if(tagged::option(type)){return leaf(tagged::inner(type));}if(tagged::result(type)){let u32 yes=leaf(tagged::success(type));return if(yes!=0){yes}else{leaf(tagged::failure(type))};}return 0;}
pub fn bool mutable(u32 type){let u32 t=leaf(type);return views::mutable(t) || refs::mutable(t);}
