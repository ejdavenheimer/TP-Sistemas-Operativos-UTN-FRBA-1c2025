package list

import "fmt"

// List Definir la interfaz List
type List[T any] interface {
	Add(item T)                                          // Añadir un elemento al final de la lista
	Dequeue() (T, error)                                 // Eliminar y devolver el primer elemento de la lista
	Filter(value T, predicate func(a, b T) bool) List[T] // Filtra elementos de la lista
	ForEach(callback func(T))                            // A cada elemento de la lista se le va aplicar la función que le pase
	Get(index int) (T, error)                            // Obtener un elemento a partir de un índice dado
	Insert(index int, item T) error                      // Insertar un elemento en el índice dado
	Pop() (T, error)                                     // Remover el último elemento de la lista
	Remove(index int)                                    // Eliminar un elemento en el índice dado
	Size() int                                           // Retornar el tamaño de la lista
	Sort(less func(a, b T) bool)                         // Ordena una Lista de acuerdo al criterio
}

// ArrayList implements List
type ArrayList[T any] struct {
	items []T
}

// Add: Inserta un elemento al final de la lista.
//
// Parámetros:
//   - item: Elemento a insertar.
//
// Ejemplo:
//
//	func main() {
//		list := &ArrayList[int]{}
//		list.Add(10)
//		list.Add(20)
//	}
func (list *ArrayList[T]) Add(item T) {
	list.items = append(list.items, item)
}

// Dequeue: Elimina y devuelve el primer elemento de la cola.
// En caso que la lista se encuentre vacío retorna el valor "cero" del tipo T y un error indicando que esta vacía.
//
// Ejemplo:
//
//	func main() {
//		numbers := &list.ArrayList[int]{}
//		numbers.Add(10)
//		numbers.Add(20)
//		numbers.Add(30)
//		value, _ := numbers.Dequeue()
//		fmt.Println("Valor: ", value) //output: 10
//	}
func (list *ArrayList[T]) Dequeue() (T, error) {
	if len(list.items) == 0 {
		var zero T // Devuelve el valor "cero" del tipo T
		return zero, fmt.Errorf("La cola está vacía")
	}
	valor := list.items[0]
	list.items = list.items[1:]
	return valor, nil
}

// Filter: Filtra elementos de la lista en base a un predicado.
//
// Parámetros:
//   - value: Valor a comparar.
//   - predicate: Función que compara dos elementos.
//
// Ejemplo:
//
//	func main() {
//		list := &ArrayList[int]{}
//		list.Add(10)
//		list.Add(20)
//		list.Add(30)
//		list.Add(40)
//		predicate := func(a int, b int) bool {
//			return b > a
//		}
//		//en este caso devuelva una nueva lista con aquellos que que son mayores que 20
//		filtered := list.Filter(20, predicate)
//	}
func (list *ArrayList[T]) Filter(value T, predicate func(a, b T) bool) List[T] {
	filteredList := &ArrayList[T]{}

	for _, item := range list.items {
		if predicate(value, item) {
			filteredList.Add(item)
		}
	}

	return filteredList
}

// Get: Devuelve el elemento en el índice proporcionado.
//
// Parámetros:
//   - index: Índice del elemento a obtener.
//
// Ejemplo:
//
//	func main() {
//		list := &ArrayList[int]{}
//		list.Add(10)
//		list.Add(20)
//		list.Add(30)
//
//		value, _ := list.Get(1)
//		fmt.Println("Valor: ", value) //Output: 20
//	}
func (list *ArrayList[T]) Get(index int) (T, error) {
	// Validar si el índice está dentro del rango
	if index < 0 || index >= len(list.items) {
		// Get item from a List
		var zero T // Crear un valor cero del tipo genérico T
		return zero, fmt.Errorf("index out of range: %d", index)
	}
	return list.items[index], nil
}

// Insert: Inserta un elemento en la lista en el índice proporcionado.
//
// Parámetros:
//   - index: Índice donde se va a ingresar el elemento.
//   - item:  Elemento a ingresar.
//
// Ejemplo:
//
//	func main() {
//		list := &ArrayList[int]{}
//		list.Add(10)
//		list.Add(30)
//
//		_ := list.Insert(1, 100) [10, 100, 30]
//	}
func (list *ArrayList[T]) Insert(index int, item T) error {
	if index < 0 || index > len(list.items) {
		return fmt.Errorf("index out of range: %d", index)
	}
	list.items = append(list.items[:index], append([]T{item}, list.items[index:]...)...)
	return nil
}

// Pop: Remueve el último elemento de la lista y lo devuelve.
//
// Parámetros:
//   - value: Valor a comparar.
//   - predicate: Función que compara dos elementos.
//
// Ejemplo:
//
//	func main() {
//		list := &ArrayList[int]{}
//		list.Add(10)
//		list.Add(20)
//		list.Add(30)
//
//		value, _ := list.Pop()
//		fmt.Println("Valor: ", value) //Output: 30
//	}
func (list *ArrayList[T]) Pop() (T, error) {
	if len(list.items) == 0 {
		var zero T
		return zero, fmt.Errorf("list is empty")
	}
	lastIndex := len(list.items) - 1
	item := list.items[lastIndex]
	list.items = list.items[:lastIndex]
	return item, nil
}

// Remove: Remueve un elemento de la lista en base a su índice.
//
// Parámetros:
//   - list: lista de cualquier tipo.
//   - index: Índice del elemento a remover.
//
// Ejemplo:
//
//	func main() {
//		list := &ArrayList[int]{}
//		list.Add(10)
//		list.Add(20)
//		list.Add(30)
//		// Eliminar el elemento en índice 1
//		list.Remove(1)  //[10, 30]
//	}
func (list *ArrayList[T]) Remove(index int) {
	if index >= 0 && index < len(list.items) {
		list.items = append(list.items[:index], list.items[index+1:]...)
	}
}

// Size: Devuelve el tamaño de la lista.
//
// Ejemplo
//
//	func main() {
//	    list := &ArrayList[int]{}
//
//	    list.Add(10)
//	    list.Add(20)
//	    list.Add(30)
//
//	    size := list.Size()
//	    fmt.Println("Size: ", size) //output: 3
//	}
func (list *ArrayList[T]) Size() int {
	return len(list.items)
}

// Sort: Ordena una lista de acuerdo a un criterio.
//
// Parámetros:
//   - value: Valor a comparar.
//   - predicate: Función que compara dos elementos.
//
// Ejemplo:
//
//	func main() {
//		list := &ArrayList[int]{}
//		list.Add(40)
//		list.Add(20)
//		list.Add(30)
//		list.Add(10)
//
//		predicate := func(a int, b int) bool {
//			return a < b
//		}
//
//		list.Sort(predicate) //[10, 20, 30, 40]
//	}
func (list *ArrayList[T]) Sort(less func(a, b T) bool) {
	size := list.Size()
	for i := 0; i < size-1; i++ {
		for j := 0; j < size-i-1; j++ {
			if !less(list.items[j], list.items[j+1]) {
				// Intercambiar los elementos si están en el orden incorrecto
				list.items[j], list.items[j+1] = list.items[j+1], list.items[j]
			}
		}
	}
}

// ForEach: a cada elemento de la lista se le va aplicar la función que le pase.
//
// Parámetros:
//   - callback: es una función que se ejecuta para cada elemento de la lista.
//
// Ejemplo
//
//	func main() {
//	    list := &ArrayList[int]{}
//
//	    list.Add(10)
//	    list.Add(20)
//	    list.Add(30)
//
//	    list.ForEach(func(number int) {
//	    	fmt.Println("Valor:", number)
//	    })
//	}
func (list *ArrayList[T]) ForEach(callback func(T)) {
	for _, item := range list.items {
		callback(item)
	}
}
