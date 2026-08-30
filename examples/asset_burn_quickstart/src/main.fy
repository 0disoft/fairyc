module main;

@abilities(store, key)
resource Coin {
    i32 amount;
    cleanup {
        assert(false);
    }
}

fn i32 main() {
    let Coin coin = Coin { .amount = 42 };
    burn coin;
    return 42;
}
