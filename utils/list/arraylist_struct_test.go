package list

import (
	"fmt"
	"testing"
)

type Person struct {
	id   int
	name string
	mail string
}

var persons ArrayList[Person]

func TestArrayList(t *testing.T) {
	setupPersons()

	if persons.Size() != 3 {
		t.Errorf("Expected size 3, got %d", persons.Size())
	}

	value, err := persons.Dequeue()
	if err != nil || value.id != 1 {
		t.Errorf("Expected id 1 at index 0, got %d", value.id)
	}

	size := persons.Size()
	if size != 2 {
		t.Errorf("Expected size 2, got %d", size)
	}

	value, err = persons.Get(0)
	if err != nil || value.id != 2 {
		t.Errorf("Expected id 2 at index 0, got %d", value.id)
	}
}

func setupPersons() {
	persons = ArrayList[Person]{}
	for i := 1; i <= 3; i++ {
		persons.Add(Person{
			id:   i,
			name: fmt.Sprintf("test%d", i),
			mail: fmt.Sprintf("test%d@mail.com", i),
		})
	}
}
