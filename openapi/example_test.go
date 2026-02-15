package openapi_test

import (
	"fmt"

	v "github.com/Gobd/apivalidation"
	"github.com/Gobd/apivalidation/openapi"
)

type Item struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

func (it *Item) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&it.Name, v.Required, v.Length(1, 200)),
		v.Field(&it.Price, v.Required, v.Min(0.01)),
	}
}

func ExamplePost() {
	doc := openapi.DocBase("Shop API", "Example API", "1.0.0")

	openapi.Post(doc, "/items", "createItem", openapi.Endpoint{
		Summary:  "Create an item",
		Request:  Item{},
		Response: Item{},
	})

	fmt.Println(doc.Paths.Value("/items").Post.OperationID)
	// Output: createItem
}

func ExampleDocBase() {
	doc := openapi.DocBase("My Service", "A cool service", "0.1.0")
	fmt.Println(doc.Info.Title)
	fmt.Println(doc.OpenAPI)
	// Output:
	// My Service
	// 3.0.3
}

func ExampleGet() {
	doc := openapi.DocBase("Shop API", "Example API", "1.0.0")

	openapi.Get(doc, "/items", "listItems", openapi.Endpoint{
		Summary:  "List all items",
		Response: []Item{},
	})

	fmt.Println(doc.Paths.Value("/items").Get.OperationID)
	// Output: listItems
}
