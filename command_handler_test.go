package main

import (
	"context"
	"fmt"
	"testing"
)

func TestGetGroupTagsEng(t *testing.T) {
	ctx := context.Background()
	title := "币圈和矿业"
	desc := `
    hello world rain bitcoin group crypto currencies
区块链；加密货币；币圈；挖矿；加密币交易
區塊鏈；加密貨幣；幣圈；挖礦；加密幣交易
立志于成为华语圈最好的匿名币圈讨论组
立志於成為華語圈最好的匿名幣圈討論組
`
	tags := getGroupTagsEng(ctx, title, desc)
	fmt.Println(tags)
}

func TestGetGroupTagsCN(t *testing.T) {
	ctx := context.Background()
	title := "币圈和矿业"
	desc := `
区块链；加密货币；币圈；挖矿；加密币交易
區塊鏈；加密貨幣；幣圈；挖礦；加密幣交易
立志于成为华语圈最好的匿名币圈讨论组
立志於成為華語圈最好的匿名幣圈討論組
`
	tags := getGroupTagsCN(ctx, title, desc)
	fmt.Println(tags)

}
