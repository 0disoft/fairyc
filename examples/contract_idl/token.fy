module contract_idl::token;

extern contract Token {
    @view
    fn Result<u64, ContractError> balance_of(Address owner);

    @payable
    fn Result<void, ContractError> transfer(Address to, u256 amount);
}

@event
struct Transfer {
    Address from;
    Address to;
    u256 amount;
}
