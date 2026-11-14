// Testa validações e cálculos de vendas em centavos.
package vendas

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testOwner = "11111111-1111-4111-8111-111111111111"

func exampleSale() CreateSaleInput {
	return CreateSaleInput{SoldAt: time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC), Items: []SaleItemInput{{SaleType: SaleTypeExportacao, Quantity: 10, UnitPriceCents: 4000}, {SaleType: SaleTypeMercado, Quantity: 5, UnitPriceCents: 3000}}, Discount: &SaleDiscountInput{Reason: " Desconto ", AmountCents: 2000}}
}

func TestNormalizeSale(t *testing.T) {
	sale, err := normalizeSale(testOwner, exampleSale())
	require.NoError(t, err)
	assert.EqualValues(t, 55000, sale.GrossCents)
	assert.EqualValues(t, 2000, sale.DiscountCents)
	assert.EqualValues(t, 53000, sale.NetCents)
	assert.EqualValues(t, 0, sale.ReceivedCents)
	assert.EqualValues(t, 53000, sale.ReceivableCents)
	assert.Equal(t, "Desconto", sale.Discount.Reason)
	cases := []struct {
		name   string
		change func(*CreateSaleInput)
		want   error
	}{
		{"sem itens", func(i *CreateSaleInput) { i.Items = nil }, ErrInvalidSaleItems},
		{"data ausente", func(i *CreateSaleInput) { i.SoldAt = time.Time{} }, ErrInvalidSaleDate},
		{"classificação inválida", func(i *CreateSaleInput) { i.Items[0].SaleType = "normal" }, ErrInvalidSaleType},
		{"classificação repetida", func(i *CreateSaleInput) { i.Items[1].SaleType = i.Items[0].SaleType }, ErrDuplicateSaleType},
		{"quantidade zero", func(i *CreateSaleInput) { i.Items[0].Quantity = 0 }, ErrInvalidSaleQuantity},
		{"preço zero", func(i *CreateSaleInput) { i.Items[0].UnitPriceCents = 0 }, ErrInvalidSalePrice},
		{"desconto sem motivo", func(i *CreateSaleInput) { i.Discount.Reason = " " }, ErrInvalidSaleDiscount},
		{"desconto maior que bruto", func(i *CreateSaleInput) { i.Discount.AmountCents = 55001 }, ErrDiscountExceedsGross},
		{"estouro na multiplicação", func(i *CreateSaleInput) { i.Items[0].UnitPriceCents = math.MaxInt64 }, ErrSaleOverflow},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := exampleSale()
			tc.change(&in)
			_, err := normalizeSale(testOwner, in)
			assert.ErrorIs(t, err, tc.want)
		})
	}
	_, err = normalizeSale("inválido", exampleSale())
	assert.ErrorIs(t, err, ErrInvalidSaleOwner)
}
