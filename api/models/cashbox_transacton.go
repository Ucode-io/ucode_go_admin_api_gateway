package models

type CreateCashboxTransactionRequest struct {
	AmountOfMoney int32 `json:"amount_of_money"`
	Comment       string  `json:"comment"`
	Status        string  `json:"status"`
}
