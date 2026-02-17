package apivalidation_test

import (
	"encoding/json"
	"fmt"
	"time"

	v "github.com/Gobd/apivalidation"
)

type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

func (u *User) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&u.Name, v.Required, v.Length(1, 100)),
		v.Field(&u.Email, v.Required),
		v.Field(&u.Age, v.Min(0), v.Max(150)),
	}
}

func ExampleValidate() {
	user := &User{Name: "Alice", Email: "alice@example.com", Age: 30}
	if err := v.Validate(user); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("valid")
	// Output: valid
}

func ExampleValidate_error() {
	user := &User{Age: -1}
	err := v.Validate(user)
	fmt.Println(err)
	// Output: age: must be no less than 0; email: cannot be blank; name: cannot be blank.
}

func ExampleUnmarshalAndValidate() {
	body := []byte(`{"name":"Bob","email":"bob@example.com","age":25}`)
	var user User
	if err := v.UnmarshalAndValidate(body, &user); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(user.Name)
	// Output: Bob
}

type Event struct {
	StartDate string `json:"start_date"`
}

func (e *Event) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&e.StartDate, v.Required, v.Date("2006-01-02").
			Min(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)).
			Max(time.Date(2030, 12, 31, 0, 0, 0, 0, time.UTC))),
	}
}

func ExampleDate() {
	e := &Event{StartDate: "2025-06-15"}
	if err := v.Validate(e); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("valid")
	// Output: valid
}

type Payment struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	IsDraft  bool    `json:"-"`
}

func (p *Payment) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&p.Amount, v.When(!p.IsDraft, "not draft", v.Required, v.Min(0.01)).
			Else(v.Min(0.0))),
		v.Field(&p.Currency, v.Required, v.In("USD", "EUR", "GBP")),
	}
}

func ExampleWhen() {
	p := &Payment{Amount: 10.00, Currency: "USD"}
	if err := v.Validate(p); err != nil {
		fmt.Println(err)
		return
	}

	b, _ := json.Marshal(p)
	fmt.Println(string(b))
	// Output: {"amount":10,"currency":"USD"}
}
