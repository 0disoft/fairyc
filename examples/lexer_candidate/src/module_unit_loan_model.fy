module module_unit_loan_model;

// Loan identity names one frame-local storage root and source borrow site.
pub struct Loan {bool subview;usize origin;usize parent;usize owner;usize start;usize end;bool mutable;}
pub struct Held {bool store;usize loan;usize origin;usize start;usize end;bool mutable;bool argument;}
pub struct Conflict {u32 code;usize owner;usize frame;usize start;usize end;usize borrow_start;usize borrow_end;usize use_start;usize use_end;}
