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

	person, err := persons.Dequeue()
	if err != nil || person.id != 1 {
		t.Errorf("Expected id 1 at index 0, got %d", person.id)
	}

	size := persons.Size()
	if size != 2 {
		t.Errorf("Expected size 2, got %d", size)
	}

	person, err = persons.Get(0)
	if err != nil || person.id != 2 {
		t.Errorf("Expected id 2 at index 0, got %d", person.id)
	}

	persons.Add(Person{id: 1, name: "Jack", mail: "jack@mail.com"})

	person, index, isFound := persons.Find(func(person Person) bool {
		return person.name == "Jack"
	})

	if !isFound || person.id != 1 {
		t.Errorf("Expected id 1 at index %d, got %d", index, person.id)
	}

	person.name = "Pepe"
	err = persons.Set(index, person)

	if err != nil || person.name != "Pepe" {
		t.Errorf("Expected Pepe, got %s", person.name)
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
