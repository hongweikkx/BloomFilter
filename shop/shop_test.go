package shop

import (
	"testing"
)

func TestShop(t *testing.T) {
	shop := NewShop()
	userId1 := "1"
	userId2 := "2"
	shop.NewUser(userId1, 100, "ItemA.1", "ItemB.1", "ItemC.1")
	shop.NewUser(userId2, 100, "ItemD.2")
	err := shop.AddToShop(userId1, "ItemA.1", 30)
	if err != nil {
		t.Error("err:", err)
	}
	err = shop.AddToShop(userId2, "ItemD.2", 50)
	if err != nil {
		t.Error("err:", err)
	}
	err = shop.PurchaseItem(userId1, "ItemD.2")
	if err != nil {
		t.Error("err:", err)
	}
	if shop.GetUserFounds(userId1) != 50 {
		t.Error("err")
	}
	if shop.GetUserFounds(userId2) != 150 {
		t.Error("err")
	}
}
