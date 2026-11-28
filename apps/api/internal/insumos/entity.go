// Define os dados de um insumo cadastrado e as categorias aceitas.
package insumos

type Category string

const (
	CategoryFertilizer Category = "fertilizante"
	CategoryPesticide  Category = "defensivo"
)

const stockPending = "pending"

type Product struct {
	ID                string
	UserID            string
	Name              string
	Category          Category
	DefaultUnit       string
	Notes             string
	StockControlState string
}
