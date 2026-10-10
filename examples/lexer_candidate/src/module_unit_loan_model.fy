module module_unit_loan_model;

// Loan identity names one frame-local storage root and source borrow site.
pub struct Loan {bool fact_only;bool retained;usize binding;usize last_start;usize last_end;bool escapes;bool subview;usize origin;usize parent;usize owner;usize start;usize end;bool mutable;}
pub struct Held {bool iteration;bool store;usize loan;usize origin;usize start;usize end;bool mutable;bool argument;}
pub struct Conflict {u32 code;usize owner;usize frame;usize start;usize end;usize borrow_start;usize borrow_end;usize use_start;usize use_end;}

// One published origin leaf; last_end==0 means the retained value was unused.
pub struct Fact {usize frame;usize binding;usize origin;bool mutable;usize start;usize end;usize last_start;usize last_end;bool escapes;}
