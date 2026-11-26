// Testa validações e cálculos de vendas em centavos.
package vendas

import (
	"context"
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
	input, err := normalizeSale(testOwner, exampleSale())
	require.NoError(t, err)
	require.NotNil(t, input.Discount)
	assert.Equal(t, "Desconto", input.Discount.Reason)
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

func TestCreateSaleRejectsInvalidTotals(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*CreateSaleInput)
		want   error
	}{
		{"desconto maior que bruto", func(i *CreateSaleInput) { i.Discount.AmountCents = 55001 }, ErrDiscountExceedsGross},
		{"estouro na multiplicação", func(i *CreateSaleInput) { i.Items[0].UnitPriceCents = math.MaxInt64 }, ErrSaleOverflow},
		{
			name: "estouro na soma",
			change: func(i *CreateSaleInput) {
				for index := range i.Items {
					i.Items[index].Quantity = 1
					i.Items[index].UnitPriceCents = math.MaxInt64/2 + 1
				}
			},
			want: ErrSaleOverflow,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := exampleSale()
			tc.change(&input)
			// Campos válidos seguem para o serviço, que confere os totais
			// antes que seja necessário acessar o repositório ou a cifra.
			input, err := normalizeSale(testOwner, input)
			require.NoError(t, err)
			_, err = NewService(nil, nil).Create(context.Background(), testOwner, input)
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestOpenSoldSaleRejectsInvalidStoredItems(t *testing.T) {
	service := NewService(nil, nil)
	for _, tc := range []struct {
		name string
		item ProtectedSaleItem
		want error
	}{
		{
			name: "classificação inválida",
			item: ProtectedSaleItem{SaleType: "normal", Quantity: 1},
			want: ErrInvalidSaleType,
		},
		{
			name: "quantidade zero",
			item: ProtectedSaleItem{SaleType: SaleTypeMercado},
			want: ErrInvalidProtectedSaleAmount,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// A venda deve rejeitar o item antes de tentar abrir o envelope.
			_, err := service.openSoldSale(testOwner, ProtectedSale{Items: []ProtectedSaleItem{tc.item}})
			assert.ErrorIs(t, err, tc.want)
		})
	}
}
