package list

import (
	"testing"
)

func TestArrayList_Add(t *testing.T) {
	list := &ArrayList[int]{}

	list.Add(10)
	list.Add(20)

	if list.Size() != 2 {
		t.Errorf("Expected size 2, got %d", list.Size())
	}
}

func TestArrayList_Remove(t *testing.T) {
	list := &ArrayList[int]{}

	list.Add(10)
	list.Add(20)
	list.Add(30)

	list.Remove(1) // Eliminar el elemento en índice 1

	if list.Size() != 2 {
		t.Errorf("Expected size 2, got %d", list.Size())
	}

	value, _ := list.Get(1)

	if value != 30 {
		t.Errorf("Expected 30 at index 1, got %d", value)
	}
}

func TestArrayList_Size(t *testing.T) {
	list := &ArrayList[int]{}

	if list.Size() != 0 {
		t.Errorf("Expected size 0, got %d", list.Size())
	}

	list.Add(10)

	if list.Size() != 1 {
		t.Errorf("Expected size 1, got %d", list.Size())
	}
}

func TestArrayList_Filter(t *testing.T) {
	list := &ArrayList[int]{}

	list.Add(10)
	list.Add(20)
	list.Add(30)
	list.Add(40)

	predicate := func(a int, b int) bool {
		return b > a
	}

	//en este caso devuelva una nueva lista con aquellos que que son mayores que 20
	filtered := list.Filter(20, predicate)

	if filtered.Size() != 2 {
		t.Errorf("Expected size 2, got %d", filtered.Size())
	}

	value, err := filtered.Get(0)
	if err != nil || value != 30 {
		t.Errorf("Expected 30 at index 0, got %d", value)
	}

	value, err = filtered.Get(1)
	if err != nil || value != 40 {
		t.Errorf("Expected 40 at index 1, got %d", value)
	}
}

func TestArrayList_Sort(t *testing.T) {
	list := &ArrayList[int]{}

	list.Add(40)
	list.Add(20)
	list.Add(30)
	list.Add(10)

	predicate := func(a int, b int) bool {
		return a < b
	}

	list.Sort(predicate)

	value, err := list.Get(0)
	if err != nil || value != 10 {
		t.Errorf("Expected 10 at index 0, got %d", value)
	}

	value, err = list.Get(2)
	if err != nil || value != 30 {
		t.Errorf("Expected 30 at index 2, got %d", value)
	}
}

func TestArrayList_Pop(t *testing.T) {
	list := &ArrayList[int]{}

	list.Add(10)
	list.Add(20)
	list.Add(30)

	value := list.Size()
	if value != 3 {
		t.Errorf("Expected size 3, got %d", value)
	}

	value, err := list.Pop()
	if err != nil || value != 30 {
		t.Errorf("Expected 10 at index 0, got %d", value)
	}

	value = list.Size()
	if value != 2 {
		t.Errorf("Expected size 2, got %d", value)
	}
}

func TestArrayList_Dequeue(t *testing.T) {
	list := &ArrayList[int]{}

	list.Add(10)
	list.Add(20)
	list.Add(30)

	value := list.Size()
	if value != 3 {
		t.Errorf("Expected size 3, got %d", value)
	}

	value, err := list.Dequeue()
	if err != nil || value != 10 {
		t.Errorf("Expected 10 at index 0, got %d", value)
	}

	value = list.Size()
	if value != 2 {
		t.Errorf("Expected size 2, got %d", value)
	}

	value, err = list.Get(0)
	if err != nil || value != 20 {
		t.Errorf("Expected 20 at index 0, got %d", value)
	}
}

func TestArrayList_Dequeue_ThrowError(t *testing.T) {
	list := &ArrayList[int]{}

	_, err := list.Dequeue()
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestArrayList_Pop_ThrowError(t *testing.T) {
	list := &ArrayList[int]{}

	_, err := list.Pop()
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestArrayList_Insert(t *testing.T) {
	list := &ArrayList[int]{}

	list.Add(10)
	list.Add(20)

	value := list.Size()
	if value != 2 {
		t.Errorf("Expected size 2, got %d", value)
	}

	err := list.Insert(1, 30)
	if err != nil {
		t.Errorf("Expected nil, got %v", err)
	}

	value, err = list.Get(1)
	if err != nil || value != 30 {
		t.Errorf("Expected 30 at index 1, got %d", value)
	}

	value = list.Size()
	if value != 3 {
		t.Errorf("Expected size 3, got %d", value)
	}
}

func TestArrayList_Insert_ThrowError(t *testing.T) {
	list := &ArrayList[int]{}

	list.Add(10)
	list.Add(20)

	err := list.Insert(4, 30)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestArrayList_Find(t *testing.T) {
	list := &ArrayList[int]{}

	list.Add(10)
	list.Add(20)
	list.Add(30)

	number, found := list.Find(func(number int) bool {
		return number == 20
	})

	if !found {
		t.Errorf("Expected true, got %v", found)
	}

	if number != 20 {
		t.Errorf("Expected to find 20 at index 0, got %d", number)
	}
}
