package contractstate_test

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/contractstate"
)

func ExampleExecute() {
	builder, _ := contractstate.NewLayoutBuilder("example.token")
	totalSupply, _ := contractstate.DeclareCell(builder, "total_supply", contractstate.U64)
	balances, _ := contractstate.DeclareMap(builder, "balances", contractstate.Byte20, contractstate.U64)
	layout, _ := builder.Build()
	store, _ := contractstate.NewMemoryStore(layout)
	limits := contractstate.Limits{
		MaxWrites:      16,
		MaxWriteBytes:  4096,
		MaxValueBytes:  1024,
		MaxMapKeyBytes: 64,
	}
	var alice contractstate.Bytes20
	alice[19] = 1

	receipt, _ := contractstate.Execute(store, layout, limits, func(transaction *contractstate.Transaction) error {
		if err := totalSupply.Set(transaction, 100); err != nil {
			return err
		}
		return balances.Set(transaction, alice, 100)
	})

	var balance uint64
	_ = contractstate.View(store, layout, limits, func(transaction *contractstate.Transaction) error {
		balance, _, _ = balances.Get(transaction, alice)
		return nil
	})
	fmt.Println(receipt.Version, balance)
	// Output: 1 100
}
