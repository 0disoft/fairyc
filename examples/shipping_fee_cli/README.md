# Shipping fee CLI

Enter an order subtotal in whole Korean won. The command prints a shipping
fee of 3,000 won for subtotals below 30,000 won and zero otherwise.
The subtotal must be a nonnegative decimal integer in the signed 64-bit range.
Invalid input prints a message to stderr and exits with status 2.

Run these commands from this directory:

```text
fairyc check --locked
fairyc test --native --locked
fairyc run --locked -- 29999
fairyc run --locked -- 30000
```

The last two commands print `3000` and `0`, respectively.
